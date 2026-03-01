package rules

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"siem-server/alerts"
	logstructure "siem-server/internal/logsstructure"
	"siem-server/state"
	"strings"
	"sync"
	"time"
)

// Engine представляет движок обработки правил
type Engine struct {
	ruleStorage *RuleStorage

	// ИСПРАВЛЕНИЕ: Используем правильные имена полей
	alertMgr   *alerts.AlertManager // ← Исправлено!
	actionDisp ActionDispatcher     // ← Исправлено!
	stateStore *state.StateStorage  // ← Исправлено!

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

	log.Printf("✓ Loaded %d enabled rules", len(e.rules))
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

// checkCondition проверяет одно условие
func (e *Engine) checkCondition(cond Condition, normLog *logstructure.NormalizedLog) bool {
	// Получаем значение поля из лога
	fieldValue := e.getFieldValue(cond.Field, normLog)
	if fieldValue == nil {
		return false
	}

	// Применяем оператор
	switch cond.Operator {
	case OpEquals:
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", cond.Value)
	case OpNotEquals:
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", cond.Value)
	case OpContains:
		return strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(fmt.Sprintf("%v", cond.Value)),
		)
	case OpNotContains:
		return !strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(fmt.Sprintf("%v", cond.Value)),
		)
	case OpStartsWith:
		return strings.HasPrefix(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(fmt.Sprintf("%v", cond.Value)),
		)
	case OpEndsWith:
		return strings.HasSuffix(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(fmt.Sprintf("%v", cond.Value)),
		)
	case OpRegex:
		pattern, ok := cond.Value.(string)
		if !ok {
			return false
		}
		matched, err := regexp.MatchString(pattern, fmt.Sprintf("%v", fieldValue))
		return err == nil && matched
	default:
		log.Printf("Unknown operator: %s", cond.Operator)
		return false
	}
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

// handleCountAggregation обрабатывает подсчет событий
func (e *Engine) handleCountAggregation(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog) error {
	// Получаем значение поля для группировки
	groupValue := e.getFieldValue(rule.Aggregation.Field, normLog)
	if groupValue == nil {
		groupValue = "default"
	}
	groupKey := fmt.Sprintf("%s:%v", rule.Aggregation.Field, groupValue)

	// Парсим временное окно
	window, err := parseDuration(rule.Aggregation.TimeWindow)
	if err != nil {
		return fmt.Errorf("invalid time window: %v", err)
	}

	// Увеличиваем счетчик
	state, err := e.stateStore.IncrementCounter(ctx, rule.ID, groupKey, window)
	if err != nil {
		return fmt.Errorf("failed to increment counter: %v", err)
	}

	log.Printf("Rule %s: %s counter = %d/%d", rule.ID, groupKey, state.Counter, rule.Aggregation.Threshold)

	// Проверяем порог
	if state.Counter >= rule.Aggregation.Threshold {
		// Триггерим правило
		stateData := map[string]interface{}{
			"counter":    state.Counter,
			"first_seen": state.FirstSeen,
			"last_seen":  state.LastSeen,
			"group_key":  groupKey,
		}

		// Сбрасываем счетчик чтобы не триггерить повторно
		e.stateStore.DeleteState(ctx, rule.ID, groupKey)

		return e.triggerRule(ctx, rule, normLog, stateData)
	}

	return nil
}

// triggerRule запускает действия при срабатывании правила
func (e *Engine) triggerRule(ctx context.Context, rule *Rule, normLog *logstructure.NormalizedLog, stateData map[string]interface{}) error {
	log.Printf("🚨 Rule triggered: %s (%s)", rule.Name, rule.Severity)

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

	log.Printf("✓ Alert created: ID=%d", alert.ID)

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
