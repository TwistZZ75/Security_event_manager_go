package actions

import "time"

// представляет лог выполненного действия
type ActionLog struct {
	ID         int64                  `json:"id"`
	AlertID    int64                  `json:"alert_id"`
	ActionType string                 `json:"action_type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Status     string                 `json:"status"`
	ExecutedAt time.Time              `json:"executed_at"`
	Result     string                 `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// Статусы действия
const (
	ActionStatusPending = "pending"
	ActionStatusSuccess = "success"
	ActionStatusFailed  = "failed"
)
