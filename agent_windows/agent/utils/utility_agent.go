package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"siem-agent/config"
	"siem-agent/proto/pkg/pb"
	"time"

	"golang.org/x/sys/windows/svc/eventlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewAgent создает нового агента
func NewAgent(serverAddr string, cfg *config.Config, elog *eventlog.Log) *Agent {
	return &Agent{
		serverAddr:   serverAddr,
		cfg:          cfg,
		eventChan:    make(chan *AgentCommand, 100),
		stopChan:     make(chan struct{}),
		elog:         elog,
		pollInterval: 10 * time.Second, // Опрос каждые 10 секунд
	}
}

// Start запускает агента
func (a *Agent) Start() error {
	// Получаем имя хоста
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}
	a.hostname = hostname

	// Подключаемся к серверу для команд
	conn, err := grpc.NewClient(
		a.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to command server: %v", err)
	}

	a.grpcConn = conn
	a.grpcClient = pb.NewAgentServiceClient(conn)

	// Регистрируем агента
	if err := a.registerAgent(); err != nil {
		a.logError("Failed to register agent: %v", err)
		// Не критично, продолжаем работу
	} else {
		a.logInfo("Agent registered successfully")
	}

	// Запускаем горутины
	go a.pollCommands()    // Опрос команд
	go a.processCommands() // Обработка команд
	go a.sendHeartbeats()  // Отправка heartbeat

	a.logInfo("Agent started on %s", a.hostname)
	return nil
}

// Stop останавливает агента
func (a *Agent) Stop() {
	close(a.stopChan)

	if a.grpcConn != nil {
		a.grpcConn.Close()
	}

	a.logInfo("Agent stopped")
}

// registerAgent регистрирует агента на сервере
func (a *Agent) registerAgent() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseInfo := a.cfg.Agent

	req := &pb.RegisterAgentRequest{
		Hostname:     a.hostname,
		Os:           baseInfo.OS,
		OsVersion:    "Windows 10/11", // Можно получить динамически
		AgentVersion: "1.0.0",
		IpAddress:    getLocalIP(),
		Metadata: map[string]string{
			"username": getUsername(),
		},
	}

	resp, err := a.grpcClient.RegisterAgent(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	// Обновляем интервал polling если сервер его указал
	if resp.PollingInterval > 0 {
		a.pollInterval = time.Duration(resp.PollingInterval) * time.Second
		a.logInfo("Polling interval set to %d seconds", resp.PollingInterval)
	}

	return nil
}

// sendHeartbeats отправляет heartbeat каждые 60 секунд
func (a *Agent) sendHeartbeats() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			req := &pb.HeartbeatRequest{
				Hostname:     a.hostname,
				AgentVersion: "1.0.0",
				Status:       "running",
				Metrics:      map[string]string{},
			}

			resp, err := a.grpcClient.Heartbeat(ctx, req)
			cancel()

			if err != nil {
				a.logWarning("Failed to send heartbeat: %v", err)
				continue
			}

			if resp.Success {
				// Обновляем интервал если сервер его изменил
				if resp.NextHeartbeatInterval > 0 {
					ticker.Reset(time.Duration(resp.NextHeartbeatInterval) * time.Second)
				}
			}

		case <-a.stopChan:
			return
		}
	}
}

// logInfo записывает информационное сообщение
func (a *Agent) logInfo(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if a.elog != nil {
		a.elog.Info(1, msg)
	} else {
		log.Println("[INFO]", msg)
	}
}

// logWarning записывает предупреждение
func (a *Agent) logWarning(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if a.elog != nil {
		a.elog.Warning(2, msg)
	} else {
		log.Println("[WARNING]", msg)
	}
}

// logError записывает ошибку
func (a *Agent) logError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if a.elog != nil {
		a.elog.Error(3, msg)
	} else {
		log.Println("[ERROR]", msg)
	}
}

// getLocalIP получает локальный IP адрес
func getLocalIP() string {
	// Упрощенная версия, можно улучшить
	return "127.0.0.1"
}

// getUsername получает имя текущего пользователя
func getUsername() string {
	username := os.Getenv("USERNAME")
	if username == "" {
		return "SYSTEM"
	}
	return username
}
