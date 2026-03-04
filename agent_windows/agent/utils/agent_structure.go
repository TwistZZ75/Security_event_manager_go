package utils

import (
	"siem-agent/config"
	"siem-agent/proto/pkg/pb"
	"time"

	"golang.org/x/sys/windows/svc/eventlog"
	"google.golang.org/grpc"
)

// Agent представляет агент SIEM
type Agent struct {
	serverAddr   string
	hostname     string
	cfg          *config.Config
	grpcClient   pb.AgentServiceClient
	grpcConn     *grpc.ClientConn
	eventChan    chan *AgentCommand
	stopChan     chan struct{}
	elog         *eventlog.Log
	pollInterval time.Duration
}

// AgentCommand команда от сервера
type AgentCommand struct {
	ID         int64
	Type       string
	Parameters map[string]string
	AlertID    int64
	Priority   string
}
