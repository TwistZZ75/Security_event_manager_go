package rules

import (
	"context"
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

// Evaluate оценивает лог на соответствие всем правилам
func (e *Engine) Evaluate(ctx context.Context, normLog *logstructure.NormalizedLog) error {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	for _, rule := range e.rules {
		if err := e.evaluateRule(ctx, rule, normLog); err != nil {
			log.Printf("Error evaluating rule %s: %v", rule.ID, err)
			// Продолжаем обработку других правил
		}
	}

	return nil
}

// evaluateRule оценивает одно правило
func (e *Engine) evaluateRule(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	// Проверяем условия
	if !e.checkConditions(rule.Conditions, normLog) {
		return nil // Условия не выполнены
	}

	// Если нет агрегации - сразу триггерим
	if rule.Aggregation == nil {
		return e.triggerRule(ctx, rule, normLog, nil)
	}

	// Обработка агрегации
	return e.handleAggregation(ctx, rule, normLog)
}

// checkConditions проверяет все условия правила
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

	// ── Строковые операторы (регистронезависимые) ──────────────────────────

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
			log.Printf("checkCondition: regex value must be string, got %T", cond.Value)
			return false
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("checkCondition: invalid regex %q: %v", pattern, err)
			return false
		}
		return re.MatchString(fieldStr)

	// ── in / not_in ────────────────────────────────────────────────────────
	//
	// cond.Value может прийти как:
	//   []interface{}{"Logon Failure", "SSH Auth Failure"} из JSON (SaveRule)
	//   []string{"Logon Failure", "SSH Auth Failure"} из кода
	//
	// Сравнение регистронезависимое.

	case OpIn:
		items := toStringSlice(cond.Value)
		if items == nil {
			log.Printf("checkCondition: 'in' operator requires array value, got %T", cond.Value)
			return false
		}
		for _, item := range items {
			if fieldLow == strings.ToLower(item) {
				return true
			}
		}
		return false

	case OpNotIn:
		items := toStringSlice(cond.Value)
		if items == nil {
			log.Printf("checkCondition: 'not_in' operator requires array value, got %T", cond.Value)
			return true // если массив не распознан — не блокируем
		}
		for _, item := range items {
			if fieldLow == strings.ToLower(item) {
				return false
			}
		}
		return true

	// ── Числовые операторы ─────────────────────────────────────────────────
	//
	// Оба значения — поле лога и значение условия — парсятся в float64.
	// Если поле содержит нечисловую строку (например "Warning") — возвращаем false.

	case OpGreaterThan:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		if !ok {
			return false
		}
		return fv > cv

	case OpLessThan:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		if !ok {
			return false
		}
		return fv < cv

	case OpGreaterOrEq:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		if !ok {
			return false
		}
		return fv >= cv

	case OpLessOrEq:
		fv, cv, ok := numericPair(fieldStr, cond.Value)
		if !ok {
			return false
		}
		return fv <= cv

	// ── IP-операторы ───────────────────────────────────────────────────────
	//
	// Для ip_equals:   "value": "192.168.1.10"
	// Для ip_in_range: "value": "192.168.1.0/24"  (CIDR-нотация)

	case OpIPEquals:
		target, ok := cond.Value.(string)
		if !ok {
			return false
		}
		ip := net.ParseIP(strings.TrimSpace(fieldStr))
		targetIP := net.ParseIP(strings.TrimSpace(target))
		if ip == nil || targetIP == nil {
			return false
		}
		return ip.Equal(targetIP)

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
			log.Printf("checkCondition: invalid CIDR %q: %v", cidr, err)
			return false
		}
		return network.Contains(ip)

	default:
		log.Printf("checkCondition: unknown operator %q in rule condition", cond.Operator)
		return false
	}
}

// =============================================================================
// Вспомогательные функции
// =============================================================================

// toStringSlice преобразует interface{} в []string.
// Поддерживает []interface{} (из JSON) и []string (из кода).
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

// numericPair парсит значение поля и значение условия в float64.
// Возвращает (fieldFloat, condFloat, ok).
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

// getFieldValue извлекает значение поля из нормализованного лога
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

// handleAggregation обрабатывает агрегацию событий
func (e *Engine) handleAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	switch rule.Aggregation.Type {
	case AggregationCount:
		return e.handleCountAggregation(ctx, rule, normLog)
	default:
		log.Printf("Unsupported aggregation type: %s", rule.Aggregation.Type)
		return nil
	}
}

// handleCountAggregation — обработка агрегации типа "count".
// Считает количество подходящих событий в заданном временном окне.
// Если счётчик достигает порога — правило срабатывает.
func (e *Engine) handleCountAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {

	// ── Валидация полей агрегации ─────────────────────────────────────────────

	if rule.Aggregation.TimeWindow == "" {
		return fmt.Errorf("rule %q: aggregation.time_window is empty (use format: 5m, 1h, 30s)", rule.Name)
	}

	window, err := parseDuration(rule.Aggregation.TimeWindow)
	if err != nil {
		return fmt.Errorf("rule %q: invalid aggregation.time_window %q: %w (use format: 5m, 1h, 30s)", rule.Name, rule.Aggregation.TimeWindow, err)
	}

	if rule.Aggregation.Threshold <= 0 {
		return fmt.Errorf("rule %q: aggregation.threshold must be > 0, got %d", rule.Name, rule.Aggregation.Threshold)
	}

	// ── Формирование ключа группировки ───────────────────────────────────────
	// aggregation.field задаёт поле, по которому считаются отдельные счётчики.
	// Например, field="pc_name" → отдельный счётчик на каждый хост.
	// Если field пустой — один глобальный счётчик для всего правила.

	var groupKey string
	if rule.Aggregation.Field != "" {
		groupValue := e.getFieldValue(rule.Aggregation.Field, normLog)
		if groupValue == nil {
			// Поле есть в правиле, но не найдено в событии — пропускаем.
			log.Printf("Rule %q: aggregation field %q not found in event, skipping", rule.Name, rule.Aggregation.Field)
			return nil
		}
		groupKey = fmt.Sprintf("%s:%v", rule.Aggregation.Field, groupValue)
	} else {
		// Нет поля группировки — один счётчик на всё правило.
		groupKey = "global"
	}

	// ── Инкрементируем счётчик ────────────────────────────────────────────────

	st, err := e.stateStore.IncrementCounter(ctx, rule.ID, groupKey, window)
	if err != nil {
		return fmt.Errorf("rule %q: failed to increment counter: %w", rule.Name, err)
	}

	log.Printf("[aggregation] rule=%q key=%q counter=%d/%d window=%s",
		rule.Name, groupKey, st.Counter, rule.Aggregation.Threshold, rule.Aggregation.TimeWindow)

	// ── Проверяем порог ───────────────────────────────────────────────────────

	if st.Counter >= rule.Aggregation.Threshold {
		stateData := map[string]interface{}{
			"counter":    st.Counter,
			"threshold":  rule.Aggregation.Threshold,
			"group_key":  groupKey,
			"field":      rule.Aggregation.Field,
			"first_seen": st.FirstSeen,
			"last_seen":  st.LastSeen,
			"window":     rule.Aggregation.TimeWindow,
		}

		// Сбрасываем счётчик до срабатывания следующего алёрта.
		if err := e.stateStore.DeleteState(ctx, rule.ID, groupKey); err != nil {
			log.Printf("Rule %q: failed to reset state: %v", rule.Name, err)
		}

		return e.triggerRule(ctx, rule, normLog, stateData)
	}

	return nil
}

// triggerRule запускает действия при срабатывании правила
func (e *Engine) triggerRule(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog, stateData map[string]interface{}) error {
	log.Printf("Rule triggered: %s (%s)", rule.Name, rule.Severity)

	// Создаем алерт
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

	// Добавляем данные агрегации, если есть
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

	// Сохраняем алерт через AlertManager
	if err := e.alertMgr.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %v", err)
	}

	log.Printf("Alert created: ID=%d", alert.ID)

	// Выполняем действия через ActionDispatcher
	for _, action := range rule.Actions {
		if err := e.actionDisp.Execute(ctx, alert, action); err != nil {
			log.Printf("Action %s failed: %v", action.Type, err)
			// Продолжаем выполнение других действий
		}
	}

	// Обновляем статистику правила
	if err := e.ruleStorage.UpdateRuleTrigger(ctx, rule.ID); err != nil {
		log.Printf("Failed to update rule trigger stats: %v", err)
	}

	return nil
}

// generateDescription генерирует описание алерта
func generateDescription(rule *Rule, normLog *logstructure.NormalizedLog) string {
	return fmt.Sprintf(
		"Rule '%s' triggered. User: %s, PC: %s, Event: %s",
		rule.Name,
		normLog.Username,
		normLog.PC_name,
		normLog.Event_description,
	)
}

// parseDuration парсит строку длительности (5m, 1h, etc.)
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ReloadRules перезагружает правила из хранилища
func (e *Engine) ReloadRules(ctx context.Context) error {
	return e.LoadRules(ctx)
}

// AddRule добавляет новое правило в движок
func (e *Engine) AddRule(rule *Rule) {
	if !rule.Enabled {
		return
	}

	e.rulesMutex.Lock()
	defer e.rulesMutex.Unlock()

	e.rules[rule.ID] = rule
	log.Printf("Rule %s added to engine", rule.ID)
}

// RemoveRule удаляет правило из движка
func (e *Engine) RemoveRule(ruleID string) {
	e.rulesMutex.Lock()
	defer e.rulesMutex.Unlock()

	delete(e.rules, ruleID)
	log.Printf("Rule %s removed from engine", ruleID)
}

// GetRulesCount возвращает количество загруженных правил
func (e *Engine) GetRulesCount() int {
	e.rulesMutex.RLock()
	defer e.rulesMutex.RUnlock()

	return len(e.rules)
}
