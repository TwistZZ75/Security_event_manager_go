package rules

import (
	"time"
)

// Rule представляет правило безопасности
type Rule struct {
	ID            string       `json:"id"`                    //идентификатор правила
	Name          string       `json:"name"`                  //имя правила
	OS            string       `json:"os"`                    //для какой ОС правило
	Enabled       bool         `json:"enabled"`               //статус правила: включено/выключено
	Severity      string       `json:"severity"`              //значимость правила: low, medium, high, critical
	Conditions    []Condition  `json:"conditions"`            //условия срабатывания
	Aggregation   *Aggregation `json:"aggregation,omitempty"` //если нужна обработка серии действий
	Actions       []Action     `json:"actions"`               //какие действия предполагает правило
	Tags          []string     `json:"tags"`
	CreatedBy     string       `json:"created_by"`               //кем создано правило
	CreatedAt     time.Time    `json:"created_at"`               //когда создано правило
	UpdatedBy     string       `json:"updated_by"`               //кем обновлено правило
	UpdatedAt     *time.Time   `json:"updated_at"`               //когда обновлено правило
	LastTriggered *time.Time   `json:"last_triggered,omitempty"` //время последнего срабатывания
	TriggerCount  int64        `json:"trigger_count"`            //количество срабатываний
}

// представляет условие срабатывания правила
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// представляет агрегацию нескольких событий
type Aggregation struct {
	Type string `json:"type"` // count - посчитать количество событий
	// sequence - последовательность событий (сначала A, потом B, потом C)
	// threshold - порог по значению (например, сумма байт > n)
	Field      string `json:"field,omitempty"`
	Threshold  int    `json:"threshold"`
	TimeWindow string `json:"time_window"`        // 5m, 1h, etc.
	Operator   string `json:"operator,omitempty"` // sum, avg, max, min
}

// Action представляет действие при срабатывании правила
type Action struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Операторы для условий
const (
	OpEquals      = "equals"
	OpNotEquals   = "not_equals"
	OpContains    = "contains"
	OpNotContains = "not_contains"
	OpStartsWith  = "starts_with"
	OpEndsWith    = "ends_with"
	OpRegex       = "regex"
	OpIn          = "in"
	OpNotIn       = "not_in"
	OpGreaterThan = "greater_than"
	OpLessThan    = "less_than"
	OpGreaterOrEq = "greater_or_equal"
	OpLessOrEq    = "less_or_equal"
	OpIPInRange   = "ip_in_range"
	OpIPEquals    = "ip_equals"
)

// Типы агрегации
const (
	AggregationCount     = "count"
	AggregationSequence  = "sequence"
	AggregationThreshold = "threshold"
)

// Типы действий
const (
	ActionNotify         = "notify"
	ActionBlockAccount   = "block_account"
	ActionBlockNetwork   = "block_network"
	ActionIsolateHost    = "isolate_host"
	ActionKillProcess    = "kill_process"
	ActionQuarantineFile = "quarantine_file"
	ActionRunScript      = "run_script"
)

// Уровни серьезности
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

type RuleFilter struct {
	Severity   string
	Enabled    bool
	EnabledSet bool // Флаг, указывающий что фильтр Enabled был установлен
	Tags       []string
	Limit      int
	Offset     int
}
