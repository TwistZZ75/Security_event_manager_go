package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"regexp"
	"siem-server/alerts"
	logstructure "siem-server/internal/logsstructure"
	"siem-server/state"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Engine представляет движок обработки правил
type Engine struct {
	ruleStorage *RuleStorage
	alertMgr    *alerts.AlertManager
	actionDisp  ActionDispatcher
	stateStore  *state.StateStorage

	rules      map[string]*Rule
	rulesMutex sync.RWMutex
}

// ActionDispatcher интерфейс для выполнения действий
type ActionDispatcher interface {
	Execute(ctx context.Context, alert *alerts.Alert, action Action) error
}

// NewEngine создает новый движок правил
func NewEngine(
	ruleStorage *RuleStorage,
	alertMgr *alerts.AlertManager,
	actionDisp ActionDispatcher,
	stateStore *state.StateStorage,
) *Engine {
	return &Engine{
		ruleStorage: ruleStorage,
		alertMgr:    alertMgr,
		actionDisp:  actionDisp,
		stateStore:  stateStore,
		rules:       make(map[string]*Rule),
	}
}

// LoadRules загружает все правила из хранилища
func (e *Engine) LoadRules(ctx context.Context) error {
	rules, err := e.ruleStorage.GetAllRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load rules: %v", err)
	}

	e.rulesMutex.Lock()
	defer e.rulesMutex.Unlock()

	e.rules = make(map[string]*Rule)
	for _, rule := range rules {
		if rule.Enabled {
			e.rules[rule.ID] = rule
		}
	}

	log.Printf("Loaded %d enabled rules", len(e.rules))
	return nil
}

// Evaluate оценивает лог против всех активных правил
func (e *Engine) Evaluate(ctx context.Context, normLog *logstructure.NormalizedLog) error {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	for _, rule := range e.rules {
		if err := e.evaluateRule(ctx, rule, normLog); err != nil {
			log.Printf("Error evaluating rule %s: %v", rule.ID, err)
		}
	}
	return nil
}

// evaluateRule оценивает одно правило
func (e *Engine) evaluateRule(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	// Проверяем условия верхнего уровня (они же шаг 1 для sequence)
	if !e.checkConditions(rule.Conditions, normLog) {
		// Для sequence: даже если шаг 1 не совпал, проверяем промежуточные шаги
		if rule.Aggregation != nil && rule.Aggregation.Type == AggregationSequence {
			return e.handleSequenceIntermediateSteps(ctx, rule, normLog)
		}
		return nil
	}

	if rule.Aggregation == nil {
		return e.triggerRule(ctx, rule, normLog, nil)
	}

	switch rule.Aggregation.Type {
	case AggregationCount:
		return e.handleCountAggregation(ctx, rule, normLog)
	case AggregationThreshold:
		return e.handleThresholdAggregation(ctx, rule, normLog)
	case AggregationSequence:
		return e.handleSequenceAggregation(ctx, rule, normLog)
	default:
		log.Printf("Unsupported aggregation type: %s", rule.Aggregation.Type)
		return nil
	}
}

// ============================================================================
// COUNT агрегация — tumbling window
// ============================================================================

func (e *Engine) handleCountAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	if rule.Aggregation.TimeWindow == "" {
		return fmt.Errorf("rule %q: aggregation.time_window is empty", rule.Name)
	}
	window, err := parseDuration(rule.Aggregation.TimeWindow)
	if err != nil {
		return fmt.Errorf("rule %q: invalid time_window %q: %w", rule.Name, rule.Aggregation.TimeWindow, err)
	}
	if rule.Aggregation.Threshold <= 0 {
		return fmt.Errorf("rule %q: threshold must be > 0", rule.Name)
	}

	groupKey := e.buildGroupKey(rule.Aggregation.Field, normLog)
	if groupKey == "" {
		return nil
	}

	st, err := e.stateStore.IncrementCounter(ctx, rule.ID, groupKey, window)
	if err != nil {
		return fmt.Errorf("rule %q: failed to increment counter: %w", rule.Name, err)
	}

	log.Printf("[count] rule=%q key=%q counter=%d/%d", rule.Name, groupKey, st.Counter, rule.Aggregation.Threshold)

	if st.Counter >= rule.Aggregation.Threshold {
		stateData := map[string]interface{}{
			"aggregation_type": "count",
			"counter":          st.Counter,
			"threshold":        rule.Aggregation.Threshold,
			"group_key":        groupKey,
			"field":            rule.Aggregation.Field,
			"first_seen":       st.FirstSeen,
			"last_seen":        st.LastSeen,
			"window":           rule.Aggregation.TimeWindow,
		}
		if err := e.stateStore.DeleteState(ctx, rule.ID, groupKey); err != nil {
			log.Printf("Rule %q: failed to reset state: %v", rule.Name, err)
		}
		return e.triggerRule(ctx, rule, normLog, stateData)
	}
	return nil
}

// ============================================================================
// THRESHOLD агрегация — накопление числового значения
// ============================================================================

// thresholdState — структура состояния для threshold-агрегации.
// Хранится в state_data как JSON.
type thresholdState struct {
	Accumulated float64  `json:"accumulated"` // сумма / текущий max / сумма для avg
	Count       int64    `json:"count"`       // количество событий (для avg)
	Distinct    []string `json:"distinct"`    // уникальные значения (для distinct_count)
}

func (e *Engine) handleThresholdAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	agg := rule.Aggregation

	if agg.TimeWindow == "" {
		return fmt.Errorf("rule %q: threshold aggregation requires time_window", rule.Name)
	}
	if agg.Operator == "" {
		return fmt.Errorf("rule %q: threshold aggregation requires operator (sum/max/avg/distinct_count)", rule.Name)
	}
	if agg.ValueField == "" {
		return fmt.Errorf("rule %q: threshold aggregation requires value_field", rule.Name)
	}

	// Эффективный порог: threshold_value имеет приоритет над threshold
	thresholdValue := agg.ThresholdValue
	if thresholdValue == 0 {
		thresholdValue = float64(agg.Threshold)
	}
	if thresholdValue <= 0 {
		return fmt.Errorf("rule %q: threshold_value must be > 0", rule.Name)
	}

	window, err := parseDuration(agg.TimeWindow)
	if err != nil {
		return fmt.Errorf("rule %q: invalid time_window: %w", rule.Name, err)
	}

	groupKey := e.buildGroupKey(agg.Field, normLog)
	if groupKey == "" {
		return nil
	}

	// Извлекаем значение из события
	rawValue := e.extractValueField(agg.ValueField, normLog)

	// Загружаем текущее состояние (или создаём пустое)
	existingState, err := e.stateStore.GetState(ctx, rule.ID, groupKey)
	if err != nil {
		return fmt.Errorf("rule %q: failed to get state: %w", rule.Name, err)
	}

	var ts thresholdState
	if existingState != nil && existingState.StateData != nil {
		if b, err := json.Marshal(existingState.StateData); err == nil {
			_ = json.Unmarshal(b, &ts)
		}
	}

	// Обновляем накопленное значение согласно оператору
	triggered := false
	var currentValue float64

	switch agg.Operator {
	case ThresholdOpSum:
		numVal, ok := parseFloat(rawValue)
		if !ok {
			log.Printf("[threshold/sum] rule=%q: cannot parse numeric value from field %q: %q", rule.Name, agg.ValueField, rawValue)
			return nil
		}
		ts.Accumulated += numVal
		ts.Count++
		currentValue = ts.Accumulated
		triggered = currentValue >= thresholdValue
		log.Printf("[threshold/sum] rule=%q key=%q sum=%.2f/%.2f", rule.Name, groupKey, currentValue, thresholdValue)

	case ThresholdOpMax:
		numVal, ok := parseFloat(rawValue)
		if !ok {
			log.Printf("[threshold/max] rule=%q: cannot parse numeric value from field %q: %q", rule.Name, agg.ValueField, rawValue)
			return nil
		}
		if numVal > ts.Accumulated {
			ts.Accumulated = numVal
		}
		currentValue = ts.Accumulated
		triggered = currentValue >= thresholdValue
		log.Printf("[threshold/max] rule=%q key=%q max=%.2f/%.2f", rule.Name, groupKey, currentValue, thresholdValue)

	case ThresholdOpAvg:
		numVal, ok := parseFloat(rawValue)
		if !ok {
			log.Printf("[threshold/avg] rule=%q: cannot parse numeric value from field %q: %q", rule.Name, agg.ValueField, rawValue)
			return nil
		}
		ts.Accumulated += numVal
		ts.Count++
		currentValue = ts.Accumulated / float64(ts.Count)
		triggered = currentValue >= thresholdValue
		log.Printf("[threshold/avg] rule=%q key=%q avg=%.2f/%.2f (n=%d)", rule.Name, groupKey, currentValue, thresholdValue, ts.Count)

	case ThresholdOpDistinctCount:
		// rawValue — строковое значение, считаем уникальные
		strVal := rawValue
		if !containsStr(ts.Distinct, strVal) && strVal != "" {
			ts.Distinct = append(ts.Distinct, strVal)
		}
		currentValue = float64(len(ts.Distinct))
		triggered = currentValue >= thresholdValue
		log.Printf("[threshold/distinct] rule=%q key=%q distinct=%.0f/%.0f", rule.Name, groupKey, currentValue, thresholdValue)

	default:
		return fmt.Errorf("rule %q: unknown threshold operator %q (use: sum, max, avg, distinct_count)", rule.Name, agg.Operator)
	}

	if triggered {
		stateData := map[string]interface{}{
			"aggregation_type": "threshold",
			"operator":         agg.Operator,
			"value_field":      agg.ValueField,
			"current_value":    currentValue,
			"threshold_value":  thresholdValue,
			"group_key":        groupKey,
			"event_count":      ts.Count,
			"window":           agg.TimeWindow,
		}
		if agg.Operator == ThresholdOpDistinctCount {
			stateData["distinct_values"] = ts.Distinct
		}
		// Сбрасываем состояние после срабатывания
		if err := e.stateStore.DeleteState(ctx, rule.ID, groupKey); err != nil {
			log.Printf("Rule %q: failed to reset threshold state: %v", rule.Name, err)
		}
		return e.triggerRule(ctx, rule, normLog, stateData)
	}

	// Сохраняем обновлённое состояние
	stData := map[string]interface{}{
		"accumulated": ts.Accumulated,
		"count":       ts.Count,
		"distinct":    ts.Distinct,
	}

	if existingState == nil {
		newState := &state.RuleState{
			RuleID:    rule.ID,
			GroupKey:  groupKey,
			Counter:   1,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			StateData: stData,
			ExpiresAt: time.Now().Add(window),
		}
		return e.stateStore.SaveState(ctx, newState)
	}

	existingState.Counter++
	existingState.LastSeen = time.Now()
	existingState.StateData = stData
	return e.stateStore.SaveState(ctx, existingState)
}

// ============================================================================
// SEQUENCE агрегация — A → B → C в порядке
// ============================================================================

// sequenceState — состояние прогресса последовательности
type sequenceState struct {
	// CurrentStep — текущий ожидаемый шаг (0 = ожидаем шаг 1, уже выполнен шаг 0)
	CurrentStep int `json:"current_step"`
	// StepTimes — временные метки каждого выполненного шага
	StepTimes []time.Time `json:"step_times"`
	// FirstEventData — данные первого события (для алерта)
	FirstEventData map[string]interface{} `json:"first_event_data,omitempty"`
}

// handleSequenceAggregation — вызывается когда событие совпало с шагом 0 (Rule.Conditions)
func (e *Engine) handleSequenceAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	agg := rule.Aggregation

	if len(agg.Steps) == 0 {
		// Если шагов нет — sequence с одним условием эквивалентна count=1
		return e.triggerRule(ctx, rule, normLog, map[string]interface{}{
			"aggregation_type": "sequence",
			"step":             0,
		})
	}

	window, err := parseDuration(agg.TimeWindow)
	if err != nil {
		return fmt.Errorf("rule %q: invalid time_window: %w", rule.Name, err)
	}

	groupKey := e.buildGroupKey(agg.Field, normLog)
	if groupKey == "" {
		return nil
	}

	// Шаг 0 совпал — создаём или перезаписываем состояние
	// (новая последовательность всегда сбрасывает старую для данного groupKey)
	ss := sequenceState{
		CurrentStep: 1, // ожидаем шаг 1
		StepTimes:   []time.Time{time.Now()},
		FirstEventData: map[string]interface{}{
			"pc_name":           normLog.PC_name,
			"username":          normLog.Username,
			"event_category":    normLog.Event_category,
			"event_description": normLog.Event_description,
		},
	}

	stData, _ := structToMap(ss)
	newState := &state.RuleState{
		RuleID:    rule.ID,
		GroupKey:  groupKey,
		Counter:   1,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		StateData: stData,
		ExpiresAt: time.Now().Add(window),
	}

	log.Printf("[sequence] rule=%q key=%q step=0 completed, waiting for step 1/%d", rule.Name, groupKey, len(agg.Steps))
	return e.stateStore.SaveState(ctx, newState)
}

// handleSequenceIntermediateSteps — проверяет, соответствует ли текущее событие
// одному из промежуточных шагов последовательности (Steps[1..N]).
// Вызывается даже если топ-левел условия не совпали.
func (e *Engine) handleSequenceIntermediateSteps(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	agg := rule.Aggregation
	if agg == nil || agg.Type != AggregationSequence || len(agg.Steps) == 0 {
		return nil
	}

	groupKey := e.buildGroupKey(agg.Field, normLog)
	if groupKey == "" {
		return nil
	}

	existingState, err := e.stateStore.GetState(ctx, rule.ID, groupKey)
	if err != nil || existingState == nil {
		return nil // нет активной последовательности для этого ключа
	}

	// Декодируем состояние
	var ss sequenceState
	if b, err := json.Marshal(existingState.StateData); err == nil {
		if err := json.Unmarshal(b, &ss); err != nil {
			return nil
		}
	}

	totalSteps := len(agg.Steps) + 1 // шаг 0 + Steps
	if ss.CurrentStep >= totalSteps {
		return nil // уже завершено
	}

	// Проверяем, совпадает ли событие с текущим ожидаемым шагом
	stepConditions := agg.Steps[ss.CurrentStep-1]
	if !e.checkConditions(stepConditions, normLog) {
		return nil
	}

	// Шаг совпал
	ss.StepTimes = append(ss.StepTimes, time.Now())
	ss.CurrentStep++

	log.Printf("[sequence] rule=%q key=%q step=%d/%d completed", rule.Name, groupKey, ss.CurrentStep-1, totalSteps-1)

	// Все шаги выполнены — срабатывание!
	if ss.CurrentStep >= totalSteps {
		stateData := map[string]interface{}{
			"aggregation_type": "sequence",
			"steps_total":      totalSteps,
			"group_key":        groupKey,
			"first_event":      ss.FirstEventData,
			"step_durations":   buildStepDurations(ss.StepTimes),
			"total_duration_s": time.Since(ss.StepTimes[0]).Seconds(),
			"window":           agg.TimeWindow,
		}
		if err := e.stateStore.DeleteState(ctx, rule.ID, groupKey); err != nil {
			log.Printf("Rule %q: failed to reset sequence state: %v", rule.Name, err)
		}
		return e.triggerRule(ctx, rule, normLog, stateData)
	}

	// Обновляем состояние — ждём следующего шага
	window, _ := parseDuration(agg.TimeWindow)
	stData, _ := structToMap(ss)
	existingState.Counter = int(ss.CurrentStep)
	existingState.LastSeen = time.Now()
	existingState.StateData = stData
	existingState.ExpiresAt = ss.StepTimes[0].Add(window) // окно от первого шага
	return e.stateStore.SaveState(ctx, existingState)
}

// ============================================================================
// Вспомогательные функции
// ============================================================================

// buildGroupKey возвращает ключ группировки.
// Если field пустой — "global" (один счётчик на всё правило).
func (e *Engine) buildGroupKey(field string, normLog *logstructure.NormalizedLog) string {
	if field == "" {
		return "global"
	}
	val := e.getFieldValue(field, normLog)
	if val == nil {
		log.Printf("aggregation field %q not found in event, skipping", field)
		return ""
	}
	return fmt.Sprintf("%s:%v", field, val)
}

// extractValueField извлекает строковое представление значения поля.
// Поддерживает:
//   - стандартные поля NormalizedLog: "username", "pc_name" и т.д.
//   - dot-нотацию для полей внутри raw_log (JSON): "raw_log.bytes_toserver"
func (e *Engine) extractValueField(valueField string, normLog *logstructure.NormalizedLog) string {
	// Если поле начинается с "raw_log." — пробуем распарсить raw_log как JSON
	if strings.HasPrefix(valueField, "raw_log.") {
		subField := valueField[len("raw_log."):]
		return extractJSONField(normLog.Raw_log, subField)
	}

	val := e.getFieldValue(valueField, normLog)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// extractJSONField извлекает значение вложенного JSON-поля.
// Поддерживает однократную точку: "alert.severity", "net.bytes_toserver".
func extractJSONField(jsonStr, path string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}

	parts := strings.SplitN(path, ".", 2)
	val, ok := data[parts[0]]
	if !ok {
		return ""
	}

	if len(parts) == 1 {
		return fmt.Sprintf("%v", val)
	}

	// Рекурсивно для вложенных объектов
	if nested, ok := val.(map[string]interface{}); ok {
		if v, ok := nested[parts[1]]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// parseFloat пытается распарсить строку как float64.
func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// containsStr проверяет наличие строки в срезе.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// structToMap конвертирует структуру в map через JSON.
func structToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	return m, json.Unmarshal(b, &m)
}

// buildStepDurations вычисляет интервалы между шагами в секундах.
func buildStepDurations(times []time.Time) []float64 {
	if len(times) < 2 {
		return nil
	}
	durations := make([]float64, len(times)-1)
	for i := 1; i < len(times); i++ {
		durations[i-1] = times[i].Sub(times[i-1]).Seconds()
	}
	return durations
}

// ============================================================================
// Проверка условий
// ============================================================================

func (e *Engine) checkConditions(conditions []Condition, normLog *logstructure.NormalizedLog) bool {
	for _, cond := range conditions {
		if !e.checkCondition(cond, normLog) {
			return false
		}
	}
	return true
}

func (e *Engine) checkCondition(cond Condition, normLog *logstructure.NormalizedLog) bool {
	fieldValue := e.getFieldValue(cond.Field, normLog)
	if fieldValue == nil {
		return false
	}

	fieldStr := fmt.Sprintf("%v", fieldValue)
	fieldLow := strings.ToLower(fieldStr)

	switch cond.Operator {
	case OpEquals:
		return fieldLow == strings.ToLower(fmt.Sprintf("%v", cond.Value))
	case OpNotEquals:
		return fieldLow != strings.ToLower(fmt.Sprintf("%v", cond.Value))
	case OpContains:
		return strings.Contains(fieldLow, strings.ToLower(fmt.Sprintf("%v", cond.Value)))
	case OpNotContains:
		return !strings.Contains(fieldLow, strings.ToLower(fmt.Sprintf("%v", cond.Value)))
	case OpStartsWith:
		return strings.HasPrefix(fieldLow, strings.ToLower(fmt.Sprintf("%v", cond.Value)))
	case OpEndsWith:
		return strings.HasSuffix(fieldLow, strings.ToLower(fmt.Sprintf("%v", cond.Value)))
	case OpRegex:
		pattern, ok := cond.Value.(string)
		if !ok {
			return false
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(fieldStr)
	case OpIn:
		items := toStringSlice(cond.Value)
		for _, item := range items {
			if fieldLow == strings.ToLower(item) {
				return true
			}
		}
		return false
	case OpNotIn:
		items := toStringSlice(cond.Value)
		for _, item := range items {
			if fieldLow == strings.ToLower(item) {
				return false
			}
		}
		return true
	case OpGreaterThan:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		return ok && fv > cv
	case OpLessThan:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		return ok && fv < cv
	case OpGreaterOrEq:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		return ok && fv >= cv
	case OpLessOrEq:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		return ok && fv <= cv
	case OpIPEquals:
		target, ok := cond.Value.(string)
		if !ok {
			return false
		}
		ip := net.ParseIP(strings.TrimSpace(fieldStr))
		targetIP := net.ParseIP(strings.TrimSpace(target))
		return ip != nil && targetIP != nil && ip.Equal(targetIP)
	case OpIPInRange:
		cidr, ok := cond.Value.(string)
		if !ok {
			return false
		}
		ip := net.ParseIP(strings.TrimSpace(fieldStr))
		if ip == nil {
			return false
		}
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			return false
		}
		return network.Contains(ip)
	default:
		log.Printf("unknown operator %q", cond.Operator)
		return false
	}
}

func toStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	}
	return nil
}

func numericPair(fieldStr string, condValue interface{}) (float64, float64, bool) {
	fv, err := strconv.ParseFloat(strings.TrimSpace(fieldStr), 64)
	if err != nil {
		return 0, 0, false
	}
	cv, err := strconv.ParseFloat(fmt.Sprintf("%v", condValue), 64)
	if err != nil {
		return 0, 0, false
	}
	return fv, cv, true
}

func (e *Engine) getFieldValue(field string, normLog *logstructure.NormalizedLog) interface{} {
	switch strings.ToLower(field) {
	case "id":
		return normLog.ID
	case "pc_name", "pcname":
		return normLog.PC_name
	case "username":
		return normLog.Username
	case "event_description":
		return normLog.Event_description
	case "event_category":
		return normLog.Event_category
	case "process_name":
		return normLog.Process_name
	case "severity":
		return normLog.Severity
	case "timestamp":
		return normLog.Timestamp
	case "os":
		return normLog.OS
	case "source", "log_source":
		return normLog.Source
	case "raw_log":
		return normLog.Raw_log
	default:
		return nil
	}
}

// ============================================================================
// Срабатывание правила
// ============================================================================

func (e *Engine) triggerRule(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog, stateData map[string]interface{}) error {
	log.Printf("Rule triggered: %s (%s)", rule.Name, rule.Severity)

	eventData := map[string]interface{}{
		"id":                normLog.ID,
		"pc_name":           normLog.PC_name,
		"username":          normLog.Username,
		"event_description": normLog.Event_description,
		"event_category":    normLog.Event_category,
		"process_name":      normLog.Process_name,
		"severity":          normLog.Severity,
		"timestamp":         normLog.Timestamp,
		"os":                normLog.OS,
		"source":            normLog.Source,
	}

	for k, v := range stateData {
		eventData[k] = v
	}

	alert := &alerts.Alert{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		Title:       fmt.Sprintf("Rule triggered: %s", rule.Name),
		Description: generateDescription(rule, normLog),
		EventData:   eventData,
		Status:      alerts.StatusOpen,
		CreatedAt:   time.Now(),
	}

	if err := e.alertMgr.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %v", err)
	}

	log.Printf("Alert created: ID=%d", alert.ID)

	for _, action := range rule.Actions {
		if err := e.actionDisp.Execute(ctx, alert, action); err != nil {
			log.Printf("Action %s failed: %v", action.Type, err)
		}
	}

	if err := e.ruleStorage.UpdateRuleTrigger(ctx, rule.ID); err != nil {
		log.Printf("Failed to update rule trigger stats: %v", err)
	}

	return nil
}

func generateDescription(rule *Rule, normLog *logstructure.NormalizedLog) string {
	return fmt.Sprintf(
		"Rule '%s' triggered. User: %s, PC: %s, Event: %s",
		rule.Name, normLog.Username, normLog.PC_name, normLog.Event_description,
	)
}

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ── Управление движком ───────────────────────────────────────────────────────

func (e *Engine) ReloadRules(ctx context.Context) error {
	return e.LoadRules(ctx)
}

func (e *Engine) AddRule(rule *Rule) {
	if !rule.Enabled {
		return
	}
	e.rulesMutex.Lock()
	defer e.rulesMutex.Unlock()
	e.rules[rule.ID] = rule
	log.Printf("Rule %s added to engine", rule.ID)
}

func (e *Engine) RemoveRule(ruleID string) {
	e.rulesMutex.Lock()
	defer e.rulesMutex.Unlock()
	delete(e.rules, ruleID)
	log.Printf("Rule %s removed from engine", ruleID)
}

func (e *Engine) GetRulesCount() int {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()
	return len(e.rules)
}
