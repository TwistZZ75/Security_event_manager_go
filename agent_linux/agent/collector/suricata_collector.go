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

type SuricataCollector struct {
	path    string
	tailer  *tail.Tail
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

func NewSuricataCollector(path string) *SuricataCollector {
	return &SuricataCollector{
		path:    path,
		logChan: make(chan *pb.RequestRawLog, 100),
	}
}

func (suricata *SuricataCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	suricata.cancel = cancel
	if _, err := os.Stat(suricata.path); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", suricata.path)
	}

	config := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	}

	tailer, err := tail.TailFile(suricata.path, config)
	if err != nil {
		return fmt.Errorf("failed to tail file: %v", err)
	}

	suricata.tailer = tailer

	go suricata.CollectSuricata(ctx)

	return nil
}

func (suricata *SuricataCollector) Stop_collect() error {
	if suricata.cancel != nil {
		suricata.cancel()
	}
	if suricata.tailer != nil {
		suricata.tailer.Stop()
	}
	close(suricata.logChan)
	return nil
}

func (suricata *SuricataCollector) GetLogs() <-chan *pb.RequestRawLog {
	return suricata.logChan
}

func (suricata *SuricataCollector) CollectSuricata(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-suricata.tailer.Lines:
			if line.Err != nil {
				fmt.Println("Error reading line")
				continue
			}
			//fmt.Println(line)
			log := &pb.RequestRawLog{
				Username:  baseInfo.Username,
				PcName:    baseInfo.PC_name,
				Os:        baseInfo.OS,
				LogSource: "Suricata",
				Timestamp: time.Now().Format(time.RFC3339),
				LogFormat: "json",
				Content:   line.Text,
			}

			select {
			case suricata.logChan <- log:
			default:
				fmt.Println("Log channel full, dropping event")
			}
		}
	}
}
