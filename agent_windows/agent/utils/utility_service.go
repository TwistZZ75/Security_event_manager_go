package utils

import (
	"context"
	"fmt"
	"siem-agent/collector"
	"siem-agent/grpc_ag"

	"golang.org/x/sys/windows/svc"
)

type Handler interface {
	Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32)
}

// WindowsService представляет Windows Service
type WindowsService struct {
	Agent      *Agent
	logClient  *grpc_ag.LogClient
	collectors []collector.LogCollector
	ctx        context.Context
	cancel     context.CancelFunc
}

// Execute выполняет Windows Service
func (ws *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Создаем контекст для коллекторов
	ws.ctx, ws.cancel = context.WithCancel(context.Background())

	// Запускаем коллекторы логов
	ws.collectors = grpc_ag.StartAllCollectors(ws.ctx, ws.Agent.cfg)
	ws.Agent.logInfo("Started %d log collectors", len(ws.collectors))

	// Подключаемся к серверу для отправки логов
	serverAddr := fmt.Sprintf("%s:%s", ws.Agent.cfg.Server.Address, ws.Agent.cfg.Server.Port)
	logClient, err := grpc_ag.NewLogClient(serverAddr)
	if err != nil {
		ws.Agent.logError("Failed to connect to log server: %v", err)
		return false, 1
	}
	ws.logClient = logClient

	// Запускаем отправку логов
	go grpc_ag.BatchLogs(ws.ctx, logClient, ws.collectors)

	// Запускаем агент (команды)
	if err := ws.Agent.Start(); err != nil {
		ws.Agent.logError("Failed to start agent: %v", err)
		return false, 1
	}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	ws.Agent.logInfo("SIEM Agent service started successfully")

	// Обработка команд управления сервисом

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus

		case svc.Stop, svc.Shutdown:
			ws.Agent.logInfo("SIEM Agent service stopping...")

			// Останавливаем всё
			ws.Agent.Stop()
			ws.cancel()
			for _, col := range ws.collectors {
				col.Stop_collect()
			}
			if ws.logClient != nil {
				ws.logClient.Close()
			}

			// Выходим из цикла (и из функции Execute)
			changes <- svc.Status{State: svc.StopPending}
			return

		default:
			ws.Agent.logError("Unexpected control request #%d", c)
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}
