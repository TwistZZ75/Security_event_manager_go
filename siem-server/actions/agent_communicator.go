package actions

import (
	"context"
	"fmt"
	"log/slog"
	"siem-server/agent"
	"siem-server/rules"
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
	if gac.commandQueue == nil {
		return fmt.Errorf("command queue is not initialized")
	}
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
		return fmt.Errorf("failed to create command: %w", err)
	}

	slog.Debug("command was succesfuly send to host")

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
	case rules.ActionBlockAccount, rules.ActionBlockNetwork, rules.ActionIsolateHost:
		return "high"
	case rules.ActionKillProcess, rules.ActionQuarantineFile:
		return "medium"
	default:
		return "low"
	}
}
