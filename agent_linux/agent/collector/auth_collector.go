package collector

import (
	"context"
	"fmt"
	"io"
	"os"
	"siem-agent/proto/pkg/pb"
	"time"

	"github.com/hpcloud/tail"
)

type AuthCollector struct {
	path    string
	tailer  *tail.Tail
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

func NewAuthCollector(path string) *AuthCollector {
	return &AuthCollector{
		path:    path,
		logChan: make(chan *pb.RequestRawLog, 100),
	}
}

func (auth *AuthCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	auth.cancel = cancel
	if _, err := os.Stat(auth.path); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", auth.path)
	}

	config := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	}

	tailer, err := tail.TailFile(auth.path, config)
	if err != nil {
		return fmt.Errorf("failed to tail file: %v", err)
	}

	auth.tailer = tailer

	go auth.CollectSyslog(ctx)

	return nil
}

func (auth *AuthCollector) Stop_collect() error {
	if auth.cancel != nil {
		auth.cancel()
	}
	if auth.tailer != nil {
		auth.tailer.Stop()
	}
	close(auth.logChan)
	return nil
}

func (auth *AuthCollector) GetLogs() <-chan *pb.RequestRawLog {
	return auth.logChan
}

func (auth *AuthCollector) CollectSyslog(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-auth.tailer.Lines:
			if line.Err != nil {
				fmt.Println("Error reading line")
				continue
			}
			//fmt.Println(line)
			log := &pb.RequestRawLog{
				Username:  baseInfo.Username,
				PcName:    baseInfo.PC_name,
				Os:        baseInfo.OS,
				LogSource: "Auth",
				Timestamp: time.Now().Format(time.RFC3339),
				LogFormat: "auth",
				Content:   line.Text,
			}

			select {
			case auth.logChan <- log:
			default:
				fmt.Println("Log channel full, dropping event")
			}
		}
	}
}
