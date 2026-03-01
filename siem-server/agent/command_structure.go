package agent

import "time"

// AgentCommand представляет команду для агента
type AgentCommand struct {
	ID          int64
	Hostname    string
	CommandType string
	Parameters  map[string]string
	AlertID     int64
	Priority    string
	Status      string
	CreatedAt   time.Time
	SentAt      *time.Time
	CompletedAt *time.Time
	Result      string
	Error       string
}
