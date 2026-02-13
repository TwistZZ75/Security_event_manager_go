package collector

import (
	"context"
	"fmt"
	"log"
	"siem-agent/proto/pkg/pb"
	"time"

	winlog "github.com/jgunnink/gowinlog"
)

// структура сборщика событий журнала sysmon
// содержит:
// путь к журналу
// строку с информацией о том с каким id собирать события
type SysmonCollector struct {
	path    string
	events  string
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

// конструктор сборщика sysmon
// принимает путь к журналу и id собираемых событий
// возвращает ссылку на экземпляр структуры
func NewSysmonCollector(path, event string) *SysmonCollector {
	return &SysmonCollector{
		path:    path,
		events:  event,
		logChan: make(chan *pb.RequestRawLog, 100),
	}
}

// реализация функции из интерфейса базового коллектора
// получает контекст
func (sysmon *SysmonCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	sysmon.cancel = cancel
	go sysmon.CollectSysmon(ctx)
	return nil
}

// функция остановки коллектора
func (sysmon *SysmonCollector) Stop_collect() error {
	if sysmon.cancel != nil {
		sysmon.cancel()
	}
	close(sysmon.logChan)
	return nil
}

// функция получения логов из канала
func (sysmon *SysmonCollector) GetLogs() <-chan *pb.RequestRawLog {
	return sysmon.logChan
}

// функция сбора логов
// принимает контекст
// помещает логи в канал
func (sysmon *SysmonCollector) CollectSysmon(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()

	watcher, err := winlog.NewWinLogWatcher()
	if err != nil {
		fmt.Printf("Couldn't create watcher: %v\n", err)
		return
	}
	defer watcher.Shutdown()

	watcher.SubscribeFromNow(sysmon.path, sysmon.events)
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Sysmon collection")
			return
		case evt := <-watcher.Event():
			// Получаем XML представление события
			eventXml := evt.Xml

			if eventXml != "" {
				logEntry := &pb.RequestRawLog{
					Username:  baseInfo.Username,
					PcName:    baseInfo.PC_name,
					Os:        baseInfo.OS,
					LogSource: "Sysmon",
					Timestamp: time.Now().Format(time.RFC3339),
					LogFormat: "xml",
					Content:   eventXml,
				}

				// Отправляем в канал (non-blocking)
				select {
				case sysmon.logChan <- logEntry: // Успешно отправлено
				default:
					log.Println("Warning: log channel is full, dropping event")
				}
			} else {
				log.Printf("Ошибка XML: %v", err)
				continue
			}
		}
	}
}
