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

type SyslogCollector struct {
	path    string
	tailer  *tail.Tail
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

func NewSyslogCollector(path string) *SyslogCollector {
	return &SyslogCollector{
		path:    path,
		logChan: make(chan *pb.RequestRawLog, 100),
	}
}

func (syslog *SyslogCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	syslog.cancel = cancel
	if _, err := os.Stat(syslog.path); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", syslog.path)
	}

	config := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	}

	tailer, err := tail.TailFile(syslog.path, config)
	if err != nil {
		return fmt.Errorf("failed to tail file: %v", err)
	}

	syslog.tailer = tailer

	go syslog.CollectSyslog(ctx)

	return nil
}

func (syslog *SyslogCollector) Stop_collect() error {
	if syslog.cancel != nil {
		syslog.cancel()
	}
	if syslog.tailer != nil {
		syslog.tailer.Stop()
	}
	close(syslog.logChan)
	return nil
}

func (syslog *SyslogCollector) GetLogs() <-chan *pb.RequestRawLog {
	return syslog.logChan
}

func (syslog *SyslogCollector) CollectSyslog(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-syslog.tailer.Lines:
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
			case syslog.logChan <- log:
			default:
				fmt.Println("Log channel full, dropping event")
			}
		}
	}
}
