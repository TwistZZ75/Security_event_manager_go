package main

import (
	"fmt"
	"siem-agent/config"
	"siem-agent/utils"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	serviceName = "SIEMAgent"
	serviceDesc = "SIEM Agent for Windows - Collects logs and executes commands"
)

func main() {
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
