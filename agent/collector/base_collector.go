package collector

import (
	"context"
	"os"
	"os/user"
	"runtime"
	"siem-agent/proto/pkg/pb"
)

type LogCollector interface {
	Start_collect(ctx context.Context) error
	Stop_collect() error
	Logs() <-chan *pb.RequestRawLog
}

type BaseLogCollectorInfo struct {
	Username string
	PC_name  string
	OS       string
}

func NewBaseCollectorInfo() *BaseLogCollectorInfo {
	return &BaseLogCollectorInfo{
		Username: defineUsername(),
		PC_name:  definePCname(),
		OS:       defineOS(),
	}
}

func defineUsername() string {
	user, err := user.Current()
	if err != nil {
		return "Guest"
	}
	return user.Username
}

func definePCname() string {
	os, err := os.Hostname()
	if err != nil {
		return defineUsername()
	}
	return os
}

func defineOS() string {
	return runtime.GOOS + " " + runtime.GOARCH
}
