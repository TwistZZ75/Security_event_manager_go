package utils

import "siem-agent/proto/pkg/pb"

const (
	serviceName = "siem-agent"
	serviceDesc = "SIEM Security Agent for log collection and response actions"
)

// Agent представляет агент SIEM для Linux
type Agent struct {
	serverAddr string
	hostname   string
	grpcClient pb.AgentServiceClient
	eventChan  chan *AgentCommand
	stopChan   chan struct{}
}

// AgentCommand команда от сервера
type AgentCommand struct {
	Type       string
	Parameters map[string]string
	AlertID    int64
}
