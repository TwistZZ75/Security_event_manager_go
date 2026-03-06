package agent

import (
	"context"
	"fmt"
	"log"
	agentpb "siem-server/proto/server/pkg/pb"
	"time"
)

// AgentServiceHandler обработчик gRPC запросов от агентов
type AgentServiceHandler struct {
	agentpb.UnimplementedAgentServiceServer
	agentStorage   *AgentStorage
	commandStorage *CommandQueue
}

// NewAgentServiceHandler создает новый handler
func NewAgentServiceHandler(agentStorage *AgentStorage, commandStorage *CommandQueue) *AgentServiceHandler {
	return &AgentServiceHandler{
		agentStorage:   agentStorage,
		commandStorage: commandStorage,
	}
}

// RegisterAgent регистрирует нового агента
func (h *AgentServiceHandler) RegisterAgent(ctx context.Context, req *agentpb.RegisterAgentRequest) (*agentpb.RegisterAgentResponse, error) {
	log.Printf("Registering agent: %s (OS: %s)", req.Hostname, req.Os)

	// Создаем или обновляем агента в БД
	agent := &AgentInfo{
		AgentID:      generateAgentID(req.Hostname),
		Hostname:     req.Hostname,
		IPAddress:    req.IpAddress,
		OS:           req.Os,
		OSVersion:    req.OsVersion,
		AgentVersion: req.AgentVersion,
		Status:       "online",
		LastSeen:     time.Now(),
		Metadata:     req.Metadata,
	}

	if err := h.agentStorage.UpsertAgent(ctx, agent); err != nil {
		log.Printf("Failed to register agent %s: %v", req.Hostname, err)
		return &agentpb.RegisterAgentResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to register: %v", err),
		}, nil
	}

	log.Printf("✓ Agent registered: %s", req.Hostname)

	return &agentpb.RegisterAgentResponse{
		Success:         true,
		Message:         "Agent registered successfully",
		AgentId:         agent.AgentID,
		PollingInterval: 10, // 10 секунд между опросами
	}, nil
}

// GetPendingCommands возвращает ожидающие команды для агента
func (h *AgentServiceHandler) GetPendingCommands(ctx context.Context, req *agentpb.GetPendingCommandsRequest) (*agentpb.GetPendingCommandsResponse, error) {
	// Обновляем last_seen
	h.agentStorage.UpdateAgentLastSeen(ctx, req.Hostname)

	// Получаем pending команды для этого хоста
	commands, err := h.commandStorage.GetPendingCommandsForHost(ctx, req.Hostname)
	if err != nil {
		log.Printf("Failed to get pending commands for %s: %v", req.Hostname, err)
		return &agentpb.GetPendingCommandsResponse{
			Commands: []*agentpb.AgentCommand{},
		}, nil
	}

	// Конвертируем в protobuf формат
	pbCommands := make([]*agentpb.AgentCommand, 0, len(commands))
	for _, cmd := range commands {
		pbCommands = append(pbCommands, &agentpb.AgentCommand{
			CommandId:   cmd.ID,
			CommandType: cmd.CommandType,
			Parameters:  cmd.Parameters,
			AlertId:     cmd.AlertID,
			Priority:    cmd.Priority,
		})
	}

	if len(pbCommands) > 0 {
		log.Printf("Sending %d pending commands to %s", len(pbCommands), req.Hostname)
	}

	return &agentpb.GetPendingCommandsResponse{
		Commands: pbCommands,
	}, nil
}

// ReportCommandResult принимает результат выполнения команды
func (h *AgentServiceHandler) ReportCommandResult(ctx context.Context, req *agentpb.ReportCommandResultRequest) (*agentpb.ReportCommandResultResponse, error) {
	result := req.Result

	log.Printf("Received command result from %s: Command=%d, Success=%v",
		req.Hostname, result.CommandId, result.Success)

	// Обновляем статус команды в БД
	status := "success"
	if !result.Success {
		status = "failed"
	}

	err := h.commandStorage.UpdateCommandStatus(ctx, result.CommandId, status, result.Result, result.Error)
	if err != nil {
		log.Printf("Failed to update command %d status: %v", result.CommandId, err)
		return &agentpb.ReportCommandResultResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to update status: %v", err),
		}, nil
	}

	return &agentpb.ReportCommandResultResponse{
		Success: true,
		Message: "Result received",
	}, nil
}

// GetAgentStatus возвращает статус агента
func (h *AgentServiceHandler) GetAgentStatus(ctx context.Context, req *agentpb.GetAgentStatusRequest) (*agentpb.GetAgentStatusResponse, error) {
	agent, err := h.agentStorage.GetAgentByHostname(ctx, req.Hostname)
	if err != nil {
		return &agentpb.GetAgentStatusResponse{
			Online: false,
		}, nil
	}

	// Считаем агента онлайн если last_seen < 5 минут назад
	online := time.Since(agent.LastSeen) < 5*time.Minute

	return &agentpb.GetAgentStatusResponse{
		Agent: &agentpb.AgentInfo{
			Hostname:     agent.Hostname,
			Os:           agent.OS,
			OsVersion:    agent.OSVersion,
			AgentVersion: agent.AgentVersion,
			IpAddress:    agent.IPAddress,
			Status:       agent.Status,
			Metadata:     agent.Metadata,
		},
		Online: online,
	}, nil
}

// generateAgentID генерирует ID агента на основе hostname
func generateAgentID(hostname string) string {
	return fmt.Sprintf("agent_%s_%d", hostname, time.Now().Unix())
}
