package rules

import (
	"regexp"
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
	Field         string         `json:"field"`
	Operator      string         `json:"operator"`
	Value         interface{}    `json:"value"`
	CompiledRegex *regexp.Regexp `json:"-"`
}

// представляет агрегацию нескольких событий
type Aggregation struct {
	// Тип агрегации: count | threshold | sequence
	Type string `json:"type"`

	// ── Общие поля ──────────────────────────────────────────────────────────

	// Field — поле события для группировки (например "pc_name" или "username").
	// Если пустое — один глобальный счётчик для всего правила.
	Field string `json:"field,omitempty"`

	// TimeWindow — ширина временного окна: "30s", "5m", "1h", "24h".
	TimeWindow string `json:"time_window"`

	// ── Поля для type=count ──────────────────────────────────────────────────

	// Threshold — количество событий, при достижении которого правило срабатывает.
	// Используется в type=count.
	Threshold int `json:"threshold"`

	// ── Поля для type=threshold ──────────────────────────────────────────────

	// ValueField — поле события, из которого извлекается числовое значение.
	// Поддерживается dot-нотация для JSON в raw_log: "raw_log.bytes_toserver".
	// Для type=threshold и operator=distinct_count — поле для подсчёта уникальных значений.
	ValueField string `json:"value_field,omitempty"`

	// Operator — операция агрегации числовых значений:
	//   sum           — сумма значений
	//   max           — максимальное значение
	//   avg           — среднее значение
	//   distinct_count — количество уникальных значений поля ValueField
	Operator string `json:"operator,omitempty"`

	// ThresholdValue — пороговое значение для type=threshold.
	// Используется float64 для поддержки дробных порогов (байты, секунды и т.п.).
	// Если не задан — используется Threshold (int) как fallback.
	ThresholdValue float64 `json:"threshold_value,omitempty"`

	// ── Поля для type=sequence ───────────────────────────────────────────────

	// Steps — шаги последовательности начиная со второго.
	// Первый шаг — это условия верхнего уровня правила (Rule.Conditions).
	// Steps[0] — условия второго шага, Steps[1] — третьего и т.д.
	//
	// Каждый шаг должен сработать в пределах TimeWindow с момента первого шага.
	// Порядок обязателен: шаг N засчитывается только если уже выполнен шаг N-1.
	Steps [][]Condition `json:"steps,omitempty"`
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

// Операторы threshold-агрегации

const (
	ThresholdOpSum           = "sum"
	ThresholdOpMax           = "max"
	ThresholdOpAvg           = "avg"
	ThresholdOpDistinctCount = "distinct_count"
)

// Типы действий
const (
	ActionNotify         = "notify"
	ActionBlockAccount   = "block_account"
	ActionUnblockAccount = "unblock_account"
	ActionBlockNetwork   = "block_network"
	ActionUnblockNetwork = "unblock_network"
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
