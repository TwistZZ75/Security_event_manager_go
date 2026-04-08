package utils

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"siem-agent/config"
	"siem-agent/proto/pkg/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewAgent создает нового агента для Linux
func NewAgent(serverAddr string, cfg *config.Config) *Agent {
	return &Agent{
		serverAddr:   serverAddr,
		cfg:          cfg,
		eventChan:    make(chan *AgentCommand, 100),
		stopChan:     make(chan struct{}),
		pollInterval: 10 * time.Second,
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
		OsVersion:    getOSVersion(),
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

	// Обновляем интервал polling, если сервер его указал
	if resp.PollingInterval > 0 {
		a.pollInterval = time.Duration(resp.PollingInterval) * time.Second
		a.logInfo("Polling interval set to %d seconds", resp.PollingInterval)
	}

	return nil
}

// logInfo записывает информационное сообщение
func (a *Agent) logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

// logWarning записывает предупреждение
func (a *Agent) logWarning(format string, v ...interface{}) {
	log.Printf("[WARNING] "+format, v...)
}

// logError записывает ошибку
func (a *Agent) logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// getLocalIP получает локальный IP адрес
func getLocalIP() string {
	cfg := &config.Config{}
	agent_interface := cfg.Agent.Interface
	//получаем структуру inteface по имени интерфейса из конфига
	iface, err := net.InterfaceByName(agent_interface)
	if err != nil {
		return "Unknown interface"
	}

	// получаем все принадлежащие интерфейсу ip
	addrs, err := iface.Addrs()
	if err != nil {
		return "No ip"
	}

	// ищем нужный IPv4
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}

// getUsername получает имя текущего пользователя в Linux
func getUsername() string {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user == "" {
		return "unknown"
	}
	return user
}

// getOSVersion возвращает версию ОС
func getOSVersion() string {
	return runtime.GOOS
}
