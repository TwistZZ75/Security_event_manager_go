package collector

import (
	"context"
	"fmt"
	"log"
	"siem-agent/proto/pkg/pb"
	"time"

	winlog "github.com/jgunnink/gowinlog"
)

type WinEventCollector struct {
	path    string
	events  string
	logChan chan *pb.RequestRawLog
	cancel  context.CancelFunc
}

// конструктор WinEventCollector
// принимает путь к журналу и анализируемые eventID из конфига
// объект WinEventCollector
func NewWinEventCollector(path, events string) *WinEventCollector {
	return &WinEventCollector{
		path:    path,
		events:  events,
		logChan: make(chan *pb.RequestRawLog, 100), //создаётся буфферизированный канал,
		// который позволяет не блокировать отправку данных в канал пока их оттуда не заберут
	}
}

// реализация функции из интерфейса LogCollector
// принимает контекст
// возвращает ошибку
func (win *WinEventCollector) Start_collect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	win.cancel = cancel
	go win.CollectFromChannel(ctx)
	return nil
}

// функция сбора логов winEvent
// получает контекст
// ничего не возвращает
func (win *WinEventCollector) CollectFromChannel(ctx context.Context) {
	baseInfo := NewBaseCollectorInfo()

	watcher, err := winlog.NewWinLogWatcher()
	if err != nil {
		fmt.Printf("Couldn't create watcher: %v\n", err)
		return
	}
	defer watcher.Shutdown()

	watcher.SubscribeFromNow(win.path, win.events)
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping WinEvent collection")
			return
		case evt := <-watcher.Event():
			// Получаем XML представление события
			eventXml := evt.Xml

			if eventXml != "" {
				logEntry := &pb.RequestRawLog{
					Username:  baseInfo.Username,
					PcName:    baseInfo.PC_name,
					Os:        baseInfo.OS,
					LogSource: "WinEvent",
					Timestamp: time.Now().Format(time.RFC3339),
					LogFormat: "xml",
					Content:   eventXml,
				}

				// Отправляем в канал (non-blocking)
				select {
				case win.logChan <- logEntry:
					// Успешно отправлено
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

// Stop останавливает коллектор
func (win *WinEventCollector) Stop_collect() error {
	if win.cancel != nil {
		win.cancel()
	}
	close(win.logChan)
	return nil
}

// GetLogs возвращает канал для чтения логов
func (win *WinEventCollector) GetLogs() <-chan *pb.RequestRawLog {
	return win.logChan
}
