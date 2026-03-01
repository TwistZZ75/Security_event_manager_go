package actions

import (
	"context"
	"fmt"
	"siem-server/agent"
)

// GRPCAgentComm реализует AgentCommunicator через CommandQueue
type GRPCAgentComm struct {
	commandQueue *agent.CommandQueue
}

// NewGRPCAgentComm создает новый коммуникатор с агентами
func NewGRPCAgentComm(commandQueue *agent.CommandQueue) *GRPCAgentComm {
	return &GRPCAgentComm{
		commandQueue: commandQueue,
	}
}

// SendCommand отправляет команду агенту через очередь команд
func (gac *GRPCAgentComm) SendCommand(ctx context.Context, host string, command AgentCommand) error {
	// Преобразуем AgentCommand в agent.AgentCommand
	agentCmd := &agent.AgentCommand{
		Hostname:    host,
		CommandType: command.Type,
		Parameters:  convertParameters(command.Parameters),
		AlertID:     command.AlertID,
		Priority:    determinePriority(command.Type),
		Status:      "pending",
	}

	// Добавляем команду в очередь
	if err := gac.commandQueue.CreateCommand(ctx, agentCmd); err != nil {
		return fmt.Errorf("failed to create command: %v", err)
	}

	return nil
}

// convertParameters конвертирует map[string]interface{} в map[string]string
func convertParameters(params map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range params {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// determinePriority определяет приоритет команды на основе типа
func determinePriority(commandType string) string {
	switch commandType {
	case "block_account", "block_network", "isolate_host":
		return "high"
	case "kill_process", "quarantine_file":
		return "medium"
	default:
		return "low"
	}
}
