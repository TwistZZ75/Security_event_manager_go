package utils

import (
	"siem-agent/config"
	"siem-agent/proto/pkg/pb"
	"time"

	"google.golang.org/grpc"
)

const (
	serviceName = "siem-agent"
	serviceDesc = "SIEM Security Agent for log collection and response actions"
)

// Agent представляет агент SIEM для Linux
type Agent struct {
	serverAddr   string
	hostname     string
	cfg          *config.Config
	grpcClient   pb.AgentServiceClient
	grpcConn     *grpc.ClientConn
	eventChan    chan *AgentCommand
	stopChan     chan struct{}
	pollInterval time.Duration
}

// AgentCommand команда от сервера
type AgentCommand struct {
	Type       string
	Parameters map[string]string
	AlertID    int64
}
