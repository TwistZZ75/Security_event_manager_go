package collector

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"siem-agent/proto/pkg/pb"
	"strings"
)

type LogCollector interface {
	Start_collect(ctx context.Context) error
	Stop_collect() error
	GetLogs() <-chan *pb.RequestRawLog
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
	cmd := exec.Command("powershell", "-Command",
		`(Get-WmiObject -Class Win32_ComputerSystem).UserName.Split('\')[1]`)
	output, err := cmd.Output()
	if err == nil {
		username := strings.TrimSpace(string(output))
		if username != "" {
			return username
		}
	}
	return "NoUserLoggedIn"
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
