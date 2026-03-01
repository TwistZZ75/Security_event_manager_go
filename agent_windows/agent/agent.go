package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	// Если запущено из командной строки
	if isIntSess {
		if len(os.Args) < 2 {
			fmt.Printf("Usage: %s <install|remove|start|stop>\n", os.Args[0])
			return
		}

		cmd := os.Args[1]
		switch cmd {
		case "install":
			err = installService()
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			fmt.Println("Service installed successfully")
		case "remove":
			err = removeService()
			if err != nil {
				log.Fatalf("Failed to remove service: %v", err)
			}
			fmt.Println("Service removed successfully")
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
		return
	}

	// Запускаем как сервис
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return
	}
	defer elog.Close()

	agent := &Agent{
		serverAddr: os.Getenv("SIEM_SERVER_ADDR"),
		eventChan:  make(chan *AgentCommand, 100),
		stopChan:   make(chan struct{}),
		elog:       elog,
	}

	winService := &WindowsService{agent: agent}

	err = svc.Run(serviceName, winService)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to run service: %v", err))
		return
	}
}
