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

type ActionFilter struct {
	AlertID    int64
	ActionType string
	Status     string
	Target     string
	Limit      int
	Offset     int
}

type ActionStats struct {
	Total        int64
	Success      int64
	Failed       int64
	Pending      int64
	BlockAccount int64
	BlockNetwork int64
	KillProcess  int64
	Notify       int64
}
