package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"siem-agent/collector"
	"siem-agent/config"
	"siem-agent/grpc_ag"
	"siem-agent/utils"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	serviceName = "SIEMAgent"
	serviceDesc = "SIEM Agent for Windows - Collects logs and executes commands"
)

func main() {
	isIntSess, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	// Если запущено из командной строки - обрабатываем команды установки/удаления
	if isIntSess {
		if len(os.Args) < 2 {
			fmt.Printf("Usage: %s <install|remove|debug>\n", os.Args[0])
			fmt.Println("\nCommands:")
			fmt.Println("  install - Install service")
			fmt.Println("  remove  - Remove service")
			fmt.Println("  debug   - Run in debug mode (not as service)")
			return
		}

		cmd := os.Args[1]
		switch cmd {
		case "install":
			err = utils.InstallService(serviceName, serviceDesc)
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			fmt.Println("✓ Service installed successfully")
			fmt.Println("Start service with: sc start " + serviceName)

		case "remove":
			err = utils.RemoveService(serviceName)
			if err != nil {
				log.Fatalf("Failed to remove service: %v", err)
			}
			fmt.Println("✓ Service removed successfully")

		case "debug":
			// Запуск в режиме отладки (не как сервис)
			fmt.Println("Running in debug mode...")
			if err := runAgent(); err != nil {
				log.Fatalf("Failed to run agent: %v", err)
			}

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Use: install, remove, or debug")
		}
		return
	}

	// Запускаем как Windows Service
	runService()
}

// runService запускает агент как Windows Service
func runService() {
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(1, "Starting SIEM Agent service")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".")
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	// Получаем адрес сервера
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)

	// Создаем агента
	agent := utils.NewAgent(serverAddr, cfg, elog)

	// Создаем Windows Service wrapper
	winService := &utils.WindowsService{Agent: agent}

	// Запускаем сервис
	err = svc.Run(serviceName, winService)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to run service: %v", err))
		return
	}
}

// runAgent запускает агент в режиме отладки (не как сервис)
func runAgent() error {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Информация о системе
	baseInfo := collector.NewBaseCollectorInfo()
	log.Printf("Starting SIEM Agent")
	log.Printf("  Hostname: %s", baseInfo.PC_name)
	log.Printf("  OS: %s", baseInfo.OS)
	log.Printf("  Username: %s", baseInfo.Username)

	// Создаем контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключаемся к серверу для отправки логов
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)
	logClient, err := grpc_ag.NewLogClient(serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to log server: %v", err)
	}
	defer logClient.Close()
	log.Printf("✓ Connected to log server: %s", serverAddr)

	// Запускаем коллекторы логов
	collectors := grpc_ag.StartAllCollectors(ctx, cfg)
	log.Printf("✓ Started %d log collectors", len(collectors))

	// Запускаем отправку логов
	go grpc_ag.BatchLogs(ctx, logClient, collectors)

	// Создаем агента для получения команд
	agent := utils.NewAgent(serverAddr, cfg, nil)

	// Запускаем агента
	if err := agent.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %v", err)
	}

	log.Println("✓ SIEM Agent is running in debug mode")
	log.Println("Press Ctrl+C to stop...")

	// Ждем сигнала завершения
	select {}
}
