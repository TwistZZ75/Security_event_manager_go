package alerts

import (
	"time"
)

// представляет алерт безопасности
type Alert struct {
	ID             int64                  `json:"id"`
	RuleID         string                 `json:"rule_id"`
	RuleName       string                 `json:"rule_name"`
	Severity       string                 `json:"severity"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	EventData      map[string]interface{} `json:"event_data"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy     string                 `json:"resolved_by,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
}

// Статусы алерта
const (
	StatusOpen          = "open"
	StatusAcknowledged  = "acknowledged"
	StatusResolved      = "resolved"
	StatusFalsePositive = "false_positive"
)

// фильтр для поиска алертов
type AlertFilter struct {
	RuleID   string
	Severity string
	Status   string
	From     time.Time
	To       time.Time
	Limit    int
	Offset   int
}

// статистика по алертам
type AlertStats struct {
	Total          int64            `json:"total"`
	BySeverity     map[string]int64 `json:"by_severity"`
	ByStatus       map[string]int64 `json:"by_status"`
	AverageResTime float64          `json:"average_resolution_time_hours"`
}
