package utils

import (
	"context"
	"fmt"
	"os"
	"siem-agent/collector"
	"siem-agent/grpc_ag"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

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

// InstallService устанавливает Windows Service
func InstallService(serviceName, serviceDesc string) error {
	exepath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	s, err = m.CreateService(serviceName, exepath, mgr.Config{
		DisplayName: serviceName,
		Description: serviceDesc,
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	defer s.Close()

	// Создаем event log source
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("failed to setup event log: %v", err)
	}

	return nil
}

// RemoveService удаляет Windows Service
func RemoveService(serviceName string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	// Останавливаем сервис если он запущен
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %v", err)
	}

	if status.State != svc.Stopped {
		_, err = s.Control(svc.Stop)
		if err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}
	}

	// Удаляем сервис
	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %v", err)
	}

	// Удаляем event log source
	err = eventlog.Remove(serviceName)
	if err != nil {
		return fmt.Errorf("failed to remove event log source: %v", err)
	}

	return nil
}
