package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"siem-agent/proto/pkg/pb"
	"time"
)

// pollCommands опрашивает сервер на наличие команд
func (a *Agent) pollCommands() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Запрашиваем команды у сервера
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := a.grpcClient.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{
				Hostname: a.hostname,
			})
			cancel()

			if err != nil {
				log.Printf("Failed to get commands: %v", err)
				continue
			}

			// Отправляем команды на обработку
			for _, cmd := range resp.Commands {
				a.eventChan <- &AgentCommand{
					Type:       cmd.CommandType,
					Parameters: cmd.Parameters,
					AlertID:    cmd.AlertId,
				}
			}

		case <-a.stopChan:
			return
		}
	}
}

// processCommands обрабатывает команды
func (a *Agent) processCommands() {
	for {
		select {
		case cmd := <-a.eventChan:
			if err := a.executeCommand(cmd); err != nil {
				log.Printf("Failed to execute command %s: %v", cmd.Type, err)
			} else {
				log.Printf("Command %s executed successfully", cmd.Type)
			}

		case <-a.stopChan:
			return
		}
	}
}

// executeCommand выполняет команду
func (a *Agent) executeCommand(cmd *AgentCommand) error {
	switch cmd.Type {
	case "block_account":
		return a.blockAccount(cmd.Parameters)
	case "block_network":
		return a.blockNetwork(cmd.Parameters)
	case "kill_process":
		return a.killProcess(cmd.Parameters)
	case "quarantine_file":
		return a.quarantineFile(cmd.Parameters)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// blockAccount блокирует учетную запись Linux
func (a *Agent) blockAccount(params map[string]string) error {
	username, ok := params["username"]
	if !ok {
		return fmt.Errorf("missing username parameter")
	}

	// Блокируем аккаунт через usermod
	cmd := exec.Command("usermod", "-L", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to block account %s: %v, output: %s", username, err, output)
	}

	log.Printf("Account %s blocked", username)
	return nil
}

// unblockAccount разблокирует учетную запись
func (a *Agent) unblockAccount(username string) error {
	cmd := exec.Command("usermod", "-U", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unblock account %s: %v, output: %s", username, err, output)
	}

	log.Printf("Account %s unblocked", username)
	return nil
}

// blockNetwork блокирует сетевой доступ
func (a *Agent) blockNetwork(params map[string]string) error {
	// Создаем iptables правило, блокирующее весь исходящий трафик
	rules := [][]string{
		{"iptables", "-I", "OUTPUT", "1", "-j", "DROP"},
		{"iptables", "-I", "OUTPUT", "1", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		{"iptables", "-I", "OUTPUT", "1", "-o", "lo", "-j", "ACCEPT"},
	}

	for _, rule := range rules {
		cmd := exec.Command(rule[0], rule[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to block network: %v, output: %s", err, output)
		}
	}

	log.Println("Network access blocked")
	return nil
}

// unblockNetwork разблокирует сетевой доступ
func (a *Agent) unblockNetwork() error {
	// Удаляем блокирующие правила
	cmd := exec.Command("iptables", "-D", "OUTPUT", "-j", "DROP")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unblock network: %v, output: %s", err, output)
	}

	log.Println("Network access restored")
	return nil
}

// killProcess завершает процесс
func (a *Agent) killProcess(params map[string]string) error {
	processName, ok := params["process_name"]
	if !ok {
		return fmt.Errorf("missing process_name parameter")
	}

	// Завершаем процесс через pkill
	cmd := exec.Command("pkill", "-9", processName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill process %s: %v, output: %s", processName, err, output)
	}

	log.Printf("Process %s terminated", processName)
	return nil
}

// quarantineFile перемещает файл в карантин
func (a *Agent) quarantineFile(params map[string]string) error {
	filePath, ok := params["path"]
	if !ok {
		return fmt.Errorf("missing path parameter")
	}

	quarantineDir := "/var/siem/quarantine"

	// Создаем директорию карантина если не существует
	if err := os.MkdirAll(quarantineDir, 0700); err != nil {
		return fmt.Errorf("failed to create quarantine directory: %v", err)
	}

	// Перемещаем файл
	fileName := filepath.Base(filePath)
	destPath := filepath.Join(quarantineDir, fmt.Sprintf("%d_%s", time.Now().Unix(), fileName))

	if err := os.Rename(filePath, destPath); err != nil {
		return fmt.Errorf("failed to quarantine file: %v", err)
	}

	log.Printf("File %s quarantined to %s", filePath, destPath)
	return nil
}
