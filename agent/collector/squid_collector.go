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

type SquidCollector struct {
	path    string
	tailer  *tail.Tail
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

func NewSquidCollector(path string) *SquidCollector {
	return &SquidCollector{
		path:    path,
		logChan: make(chan *pb.RequestRawLog, 100),
	}
}

func (squid *SquidCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	squid.cancel = cancel
	if _, err := os.Stat(squid.path); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", squid.path)
	}

	config := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	}

	tailer, err := tail.TailFile(squid.path, config)
	if err != nil {
		return fmt.Errorf("failed to tail file: %v", err)
	}

	squid.tailer = tailer

	go squid.CollectSquid(ctx)

	return nil
}

func (squid *SquidCollector) Stop_collect() error {
	if squid.cancel != nil {
		squid.cancel()
	}
	if squid.tailer != nil {
		squid.tailer.Stop()
	}
	close(squid.logChan)
	return nil
}

func (squid *SquidCollector) GetLogs() <-chan *pb.RequestRawLog {
	return squid.logChan
}

func (squid *SquidCollector) CollectSquid(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-squid.tailer.Lines:
			if line.Err != nil {
				fmt.Println("Error reading line")
				continue
			}
			//fmt.Println(line)
			log := &pb.RequestRawLog{
				Username:  baseInfo.Username,
				PcName:    baseInfo.PC_name,
				Os:        baseInfo.OS,
				LogSource: "Sysmon",
				Timestamp: time.Now().Format(time.RFC3339),
				LogFormat: "xml",
				Content:   line.Text,
			}

			select {
			case squid.logChan <- log:
			default:
				fmt.Println("Log channel full, dropping event")
			}
		}
	}
}
