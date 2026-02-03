package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	collector "siem-agent/collector"
	"siem-agent/config"
	"sync"
	"syscall"
)

func main() {
	//загрузка конфига
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Println("config truble")
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	baseInfo := collector.NewBaseCollectorInfo()
	cfg.Agent.Hostname = baseInfo.PC_name
	cfg.Agent.OS = baseInfo.OS
	fmt.Printf("Config: %v\n", cfg)
	// Обработка Ctrl+C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	if cfg.Collectors.WinEvent.Enabled == true {
		winCollect := collector.NewWinEventCollector(cfg.Collectors.WinEvent.Channel, cfg.Collectors.WinEvent.EventID)

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := winCollect.Start_collect(ctx)
			if err != nil {
				fmt.Printf("Error starting WinEvent collector: %v\n", err)
				return
			}

			logEntry := <-winCollect.GetLogs()
			fmt.Printf("Received log: %v\n\n", logEntry)
			fmt.Println("======================================================")
		}()
		// fmt.Printf("WinEvents: %v\n", winCollect.Start_collect(ctx))

		// Ожидаем сигнал Ctrl+C
		<-signalChan
		fmt.Println("\nПолучен сигнал завершения. Останавливаем сбор логов...")

		// Останавливаем коллектор
		winCollect.Stop()
	}

}
