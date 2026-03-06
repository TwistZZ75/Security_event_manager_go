package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"siem-agent/proto/pkg/pb"
	"time"
)

// pollCommands опрашивает сервер на наличие команд
func (a *Agent) pollCommands() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	a.logInfo("Started polling for commands every %v", a.pollInterval)

	for {
		select {
		case <-ticker.C:
			// Запрашиваем команды у сервера
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := a.grpcClient.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{
				Hostname:     a.hostname,
				AgentVersion: "1.0.0",
			})
			cancel()

			if err != nil {
				a.logWarning("Failed to get pending commands: %v", err)
				continue
			}

			// Отправляем команды на обработку
			if len(resp.Commands) > 0 {
				a.logInfo("Received %d commands from server", len(resp.Commands))

				for _, cmd := range resp.Commands {
					a.eventChan <- &AgentCommand{
						ID:         cmd.CommandId,
						Type:       cmd.CommandType,
						Parameters: cmd.Parameters,
						AlertID:    cmd.AlertId,
						Priority:   cmd.Priority,
					}
				}
			}

		case <-a.stopChan:
			return
		}
	}
}

// processCommands обрабатывает команды из канала
func (a *Agent) processCommands() {
	a.logInfo("Started command processor")

	for {
		select {
		case cmd := <-a.eventChan:
			a.logInfo("Processing command: %s (ID=%d, Priority=%s)", cmd.Type, cmd.ID, cmd.Priority)

			result, err := a.executeCommand(cmd)

			// Отправляем результат на сервер
			a.reportCommandResult(cmd.ID, result, err)

		case <-a.stopChan:
			return
		}
	}
}

// executeCommand выполняет команду
func (a *Agent) executeCommand(cmd *AgentCommand) (string, error) {
	switch cmd.Type {
	case "block_account":
		return a.blockAccount(cmd.Parameters)
	case "unblock_account":
		return a.unblockAccount(cmd.Parameters)
	case "block_network":
		return a.blockNetwork()
	case "unblock_network":
		return a.unblockNetwork()
	case "kill_process":
		return a.killProcess(cmd.Parameters)
	case "quarantine_file":
		return a.quarantineFile(cmd.Parameters)
	default:
		return "", fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// reportCommandResult отправляет результат выполнения команды на сервер
func (a *Agent) reportCommandResult(commandID int64, result string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	req := &pb.ReportCommandResultRequest{
		Result: &pb.CommandResult{
			CommandId: commandID,
			Success:   success,
			Result:    result,
			Error:     errorMsg,
		},
		Hostname: a.hostname,
	}

	resp, reportErr := a.grpcClient.ReportCommandResult(ctx, req)
	if reportErr != nil {
		a.logError("Failed to report command result: %v", reportErr)
		return
	}

	if !resp.Success {
		a.logWarning("Server rejected command result: %s", resp.Message)
	}

	// Логируем результат
	if success {
		a.logInfo("✓ Command %d completed: %s", commandID, result)
	} else {
		a.logError("✗ Command %d failed: %v", commandID, err)
	}
}

// ============================================================================
// WINDOWS COMMAND EXECUTORS
// ============================================================================

// blockAccount блокирует учетную запись Windows
func (a *Agent) blockAccount(params map[string]string) (string, error) {
	username, ok := params["username"]
	if !ok {
		return "", fmt.Errorf("missing username parameter")
	}

	// Выполняем команду блокировки через net user
	cmd := exec.Command("net", "user", username, "/active:no")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to block account %s: %v, output: %s", username, err, output)
	}

	return fmt.Sprintf("Account %s blocked successfully", username), nil
}

// unblockAccount разблокирует учетную запись
func (a *Agent) unblockAccount(params map[string]string) (string, error) {
	username, ok := params["username"]
	if !ok {
		return "", fmt.Errorf("missing username parameter")
	}

	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to unblock account %s: %v, output: %s", username, err, output)
	}

	return fmt.Sprintf("Account %s unblocked successfully", username), nil
}

// blockNetwork блокирует сетевой доступ
func (a *Agent) blockNetwork() (string, error) {
	// Включаем Windows Firewall
	cmd1 := exec.Command("netsh", "advfirewall", "set", "allprofiles", "state", "on")
	if output, err := cmd1.CombinedOutput(); err != nil {
		a.logWarning("Failed to enable firewall: %v, output: %s", err, output)
	}

	// Блокируем все исходящие соединения
	cmd2 := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=SIEM_BLOCK_ALL_OUT",
		"dir=out",
		"action=block",
		"enable=yes")

	output, err := cmd2.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to block network: %v, output: %s", err, output)
	}

	// Блокируем входящие соединения
	cmd3 := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=SIEM_BLOCK_ALL_IN",
		"dir=in",
		"action=block",
		"enable=yes")

	cmd3.CombinedOutput() // Игнорируем ошибки

	return "Network access blocked successfully", nil
}

// unblockNetwork разблокирует сетевой доступ
func (a *Agent) unblockNetwork() (string, error) {
	// Удаляем правила блокировки
	cmd1 := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=SIEM_BLOCK_ALL_OUT")
	cmd1.CombinedOutput()

	cmd2 := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=SIEM_BLOCK_ALL_IN")
	output, err := cmd2.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to unblock network: %v, output: %s", err, output)
	}

	return "Network access restored successfully", nil
}

// killProcess завершает процесс
func (a *Agent) killProcess(params map[string]string) (string, error) {
	processName, ok := params["process_name"]
	if !ok {
		return "", fmt.Errorf("missing process_name parameter")
	}

	// Завершаем процесс через taskkill
	cmd := exec.Command("taskkill", "/F", "/IM", processName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// taskkill возвращает ошибку если процесс не найден
		// Проверяем что это именно эта ошибка
		if contains(string(output), "not found") || contains(string(output), "не найден") {
			return "", fmt.Errorf("process %s not found", processName)
		}
		return "", fmt.Errorf("failed to kill process %s: %v, output: %s", processName, err, output)
	}

	return fmt.Sprintf("Process %s terminated successfully", processName), nil
}

// quarantineFile перемещает файл в карантин
func (a *Agent) quarantineFile(params map[string]string) (string, error) {
	filePath, ok := params["file_path"]
	if !ok {
		return "", fmt.Errorf("missing file_path parameter")
	}

	// Проверяем что файл существует
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	quarantineDir := "C:\\ProgramData\\SIEM\\Quarantine"

	// Создаем директорию карантина если не существует
	if err := os.MkdirAll(quarantineDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create quarantine directory: %v", err)
	}

	// Формируем путь назначения
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102_150405")
	destPath := filepath.Join(quarantineDir, fmt.Sprintf("%s_%s", timestamp, fileName))

	// Перемещаем файл
	if err := os.Rename(filePath, destPath); err != nil {
		return "", fmt.Errorf("failed to move file to quarantine: %v", err)
	}

	// Удаляем права доступа
	cmd := exec.Command("icacls", destPath, "/deny", "*S-1-1-0:F")
	cmd.Run() // Игнорируем ошибки

	return fmt.Sprintf("File quarantined: %s → %s", filePath, destPath), nil
}

// contains проверяет содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
