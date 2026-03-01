package state

import "time"

// RuleState представляет состояние правила для агрегации
type RuleState struct {
	ID        int64                  `json:"id"`
	RuleID    string                 `json:"rule_id"`
	GroupKey  string                 `json:"group_key"`
	Counter   int                    `json:"counter"`
	FirstSeen time.Time              `json:"first_seen"`
	LastSeen  time.Time              `json:"last_seen"`
	StateData map[string]interface{} `json:"state_data"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// статистика по состояниям
type StateStats struct {
	Total      int64
	Active     int64
	Expired    int64
	AvgCounter float64
	MaxCounter int
	MinCounter int
}
