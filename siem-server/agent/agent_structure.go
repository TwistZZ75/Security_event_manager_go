package agent

import "time"

// AgentInfo информация об агенте
type AgentInfo struct {
	ID            int64
	AgentID       string
	Hostname      string
	OS            string
	OSVersion     string
	AgentVersion  string
	IPAddress     string
	Metadata      map[string]string
	Status        string
	RegisteredAt  time.Time
	LastSeen      time.Time
	LastHeartbeat *time.Time
}
