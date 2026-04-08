package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"siem-agent/collector"
	"siem-agent/config"
	"siem-agent/grpc_ag"
	"siem-agent/utils"
	"syscall"
)

func main() {
	//загрузка конфига
	cfg, err := config.LoadConfig("/usr/local/bin")
	if err != nil {
		fmt.Printf("config load failed: %v", err)
	}
	if len(os.Args) > 1 {
		svc := &utils.AgService{}
		switch os.Args[1] {
		case "install":
			if err := svc.InstallSystemdService(cfg); err != nil {
				log.Fatalf("Install failed: %v", err)
			}
			fmt.Println("Service installed successfully")
			return
		case "uninstall", "remove":
			if err := svc.RemoveSystemdService(); err != nil {
				log.Fatalf("Remove failed: %v", err)
			}
			fmt.Println("Service removed successfully")
			return
		default:
			log.Println("Wrong argument.")
			fmt.Println("Wrong argument. Argument can be `install` or `uninstall`/`remove`")
		}
	}

	//получение информации о системе и пользователе
	baseInfo := collector.NewBaseCollectorInfo()
	cfg.Agent.Hostname = baseInfo.PC_name
	cfg.Agent.OS = baseInfo.OS
	fmt.Printf("Config: %v\n", cfg)

	//получение адреса и порта из конфига
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)
	//создание соединения с сервером
	client, err := grpc_ag.NewLogClient(serverAddr)
	if err != nil {
		fmt.Printf("Failed to connect to server %v", err)
	}
	log.Printf("Connecting to server: %s", serverAddr)
	defer client.Close()

	//создание контекста
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//запуск коллекторов
	collectors := grpc_ag.StartAllCollectors(ctx, cfg)

	//запуск горутины для сбора логов и их буфферизации для каждого из коллекторов
	go grpc_ag.BatchLogs(ctx, client, collectors)

	// Создаем агента
	agent := utils.NewAgent(serverAddr, cfg)
	agent.Start()

	// Обработка Ctrl+C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Ожидаем сигнал Ctrl+C
	<-signalChan

	//остановка всех коллекторов
	for _, col := range collectors {
		col.Stop_collect()
	}
	fmt.Println("\nПолучен сигнал завершения. Останавливаем сбор логов...")
	agent.Stop()

	//cancel()

}
