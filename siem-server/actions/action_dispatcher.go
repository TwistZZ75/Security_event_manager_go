package actions

import (
	"context"
	"fmt"
	"log"
	"siem-server/alerts"
	"siem-server/rules"
	"time"
)

// Dispatcher управляет выполнением действий
type Dispatcher struct {
	executors map[string]Executor
	actionLog ActionLogger
	agentComm AgentCommunicator
	notifier  *MultiNotifier // ← ДОБАВЛЕНО для уведомлений!
}

// Executor интерфейс для выполнения действий
type Executor interface {
	Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error
	GetType() string
}

// ActionLogger интерфейс для логирования действий
type ActionLogger interface {
	LogAction(ctx context.Context, log *ActionLog) error
}

// AgentCommunicator интерфейс для связи с агентами на клиентах
type AgentCommunicator interface {
	SendCommand(ctx context.Context, host string, command AgentCommand) error
}

// AgentCommand команда для агента
type AgentCommand struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	AlertID    int64                  `json:"alert_id"`
}

// NewDispatcher создает новый диспетчер действий
func NewDispatcher(actionLog ActionLogger, agentComm AgentCommunicator, notifier *MultiNotifier) *Dispatcher {
	d := &Dispatcher{
		executors: make(map[string]Executor),
		actionLog: actionLog,
		agentComm: agentComm,
		notifier:  notifier, // ← ДОБАВЛЕНО!
	}

	d.RegisterExecutor(NewNotifyExecutor(notifier))
	d.RegisterExecutor(NewBlockAccountExecutor(agentComm))
	d.RegisterExecutor(NewUnblockAccountExecutor(agentComm))
	d.RegisterExecutor(NewBlockNetworkExecutor(agentComm))
	d.RegisterExecutor(NewUnblockNetworkExecutor(agentComm))
	d.RegisterExecutor(NewKillProcessExecutor(agentComm))

	return d
}

// RegisterExecutor регистрирует исполнителя действия
func (d *Dispatcher) RegisterExecutor(executor Executor) {
	d.executors[executor.GetType()] = executor
	log.Printf("Registered action executor: %s", executor.GetType())
}

// Execute выполняет действие
func (d *Dispatcher) Execute(ctx context.Context, alert *alerts.Alert, action rules.Action) error {
	executor, exists := d.executors[action.Type]
	if !exists {
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	// Создаем запись в логе действий
	actionLog := &ActionLog{
		AlertID:    alert.ID,
		ActionType: action.Type,
		Target:     d.DetermineTarget(alert, action.Type),
		Parameters: action.Parameters,
		Status:     ActionStatusPending,
		ExecutedAt: time.Now(),
	}

	if err := d.actionLog.LogAction(ctx, actionLog); err != nil {
		log.Printf("Failed to log action: %v", err)
		// Продолжаем выполнение
	}

	// Выполняем действие
	err := executor.Execute(ctx, alert, action.Parameters)

	// Обновляем статус
	if err != nil {
		actionLog.Status = ActionStatusFailed
		actionLog.Error = err.Error()
		log.Printf("✗ Action %s failed for alert %d: %v", action.Type, alert.ID, err)
	} else {
		actionLog.Status = ActionStatusSuccess
		actionLog.Result = "Action executed successfully"
		log.Printf("✓ Action %s succeeded for alert %d", action.Type, alert.ID)
	}

	if logErr := d.actionLog.LogAction(ctx, actionLog); logErr != nil {
		log.Printf("Failed to update action log: %v", logErr)
	}

	return err
}

// DetermineTarget определяет цель действия
func (d *Dispatcher) DetermineTarget(alert *alerts.Alert, actionType string) string {
	switch actionType {
	case rules.ActionBlockAccount:
		username := getStringFromEventData(alert.EventData, "username")
		pcName := getStringFromEventData(alert.EventData, "pc_name")
		return fmt.Sprintf("%s@%s", username, pcName)

	case rules.ActionBlockNetwork, rules.ActionIsolateHost:
		return getStringFromEventData(alert.EventData, "pc_name")

	case rules.ActionKillProcess:
		processName := getStringFromEventData(alert.EventData, "process_name")
		pcName := getStringFromEventData(alert.EventData, "pc_name")
		return fmt.Sprintf("%s:%s", pcName, processName)

	case rules.ActionQuarantineFile:
		filePath := getStringFromEventData(alert.EventData, "file_path")
		pcName := getStringFromEventData(alert.EventData, "pc_name")
		return fmt.Sprintf("%s:%s", pcName, filePath)

	case rules.ActionNotify:
		return getStringFromEventData(alert.EventData, "pc_name")

	default:
		return getStringFromEventData(alert.EventData, "pc_name")
	}
}

// getStringFromEventData извлекает строку из event data
func getStringFromEventData(eventData map[string]interface{}, key string) string {
	if val, ok := eventData[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return "unknown"
}

// ============================================================================
// NOTIFY EXECUTOR - ОТПРАВКА УВЕДОМЛЕНИЙ
// ============================================================================

// NotifyExecutor выполняет отправку уведомлений
type NotifyExecutor struct {
	notifier *MultiNotifier
}

// NewNotifyExecutor создает новый NotifyExecutor
func NewNotifyExecutor(notifier *MultiNotifier) *NotifyExecutor {
	return &NotifyExecutor{
		notifier: notifier,
	}
}

func (ne *NotifyExecutor) GetType() string {
	return rules.ActionNotify
}

func (ne *NotifyExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	// Определяем каналы для отправки
	channels := []string{}

	// Если в параметрах указаны каналы
	if channelsParam, ok := params["channels"]; ok {
		switch v := channelsParam.(type) {
		case string:
			channels = []string{v}
		case []string:
			channels = v
		case []interface{}:
			for _, ch := range v {
				channels = append(channels, fmt.Sprintf("%v", ch))
			}
		}
	}

	// Если каналы не указаны, отправляем только в telegram
	if len(channels) == 0 {
		channels = []string{"telegram"}
	}

	// Отправляем уведомление
	if err := ne.notifier.SendNotification(ctx, alert, channels); err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}

	log.Printf("✓ Notification sent for alert %d via channels: %v", alert.ID, channels)
	return nil
}

// ============================================================================
// BLOCK ACCOUNT EXECUTOR
// ============================================================================

type BlockAccountExecutor struct {
	agentComm AgentCommunicator
}

func NewBlockAccountExecutor(agentComm AgentCommunicator) *BlockAccountExecutor {
	return &BlockAccountExecutor{agentComm: agentComm}
}

func (bae *BlockAccountExecutor) GetType() string {
	return rules.ActionBlockAccount
}

func (bae *BlockAccountExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	username := getStringFromEventData(alert.EventData, "username")
	host := getStringFromEventData(alert.EventData, "pc_name")

	if username == "unknown" || host == "unknown" {
		return fmt.Errorf("username or host not found in alert data")
	}

	duration := "0"
	if d, ok := params["duration"]; ok {
		duration = fmt.Sprintf("%v", d)
	}

	command := AgentCommand{
		Type: "block_account",
		Parameters: map[string]interface{}{
			"username": username,
			"duration": duration,
		},
		AlertID: alert.ID,
	}

	if err := bae.agentComm.SendCommand(ctx, host, command); err != nil {
		return fmt.Errorf("failed to send block account command: %v", err)
	}

	log.Printf("Block account command sent for user %s on host %s", username, host)
	return nil
}

// ============================================================================
// UNBLOCK ACCOUNT EXECUTOR
// ============================================================================

type UnblockAccountExecutor struct {
	agentComm AgentCommunicator
}

func NewUnblockAccountExecutor(agentComm AgentCommunicator) *UnblockAccountExecutor {
	return &UnblockAccountExecutor{agentComm: agentComm}
}

func (uae *UnblockAccountExecutor) GetType() string {
	return rules.ActionUnblockAccount
}

func (uae *UnblockAccountExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	// Получаем username из параметров или из alert
	username := ""
	if u, ok := params["username"]; ok {
		username = fmt.Sprintf("%v", u)
	} else {
		username = getStringFromEventData(alert.EventData, "username")
	}

	// Получаем host из параметров или из alert
	host := ""
	if h, ok := params["host"]; ok {
		host = fmt.Sprintf("%v", h)
	} else {
		host = getStringFromEventData(alert.EventData, "pc_name")
	}

	if username == "unknown" || username == "" {
		return fmt.Errorf("username not specified")
	}

	if host == "unknown" || host == "" {
		return fmt.Errorf("host not specified")
	}

	command := AgentCommand{
		Type: "unblock_account",
		Parameters: map[string]interface{}{
			"username": username,
		},
		AlertID: alert.ID,
	}

	if err := uae.agentComm.SendCommand(ctx, host, command); err != nil {
		return fmt.Errorf("failed to send unblock account command: %v", err)
	}

	log.Printf("✓ Unblock account command sent for user %s on host %s", username, host)
	return nil
}

// ============================================================================
// BLOCK NETWORK EXECUTOR
// ============================================================================

type BlockNetworkExecutor struct {
	agentComm AgentCommunicator
}

func NewBlockNetworkExecutor(agentComm AgentCommunicator) *BlockNetworkExecutor {
	return &BlockNetworkExecutor{agentComm: agentComm}
}

func (bne *BlockNetworkExecutor) GetType() string {
	return rules.ActionBlockNetwork
}

func (bne *BlockNetworkExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	host := getStringFromEventData(alert.EventData, "pc_name")

	if host == "unknown" {
		return fmt.Errorf("host not found in alert data")
	}

	duration := "0"
	if d, ok := params["duration"]; ok {
		duration = fmt.Sprintf("%v", d)
	}

	command := AgentCommand{
		Type: "block_network",
		Parameters: map[string]interface{}{
			"duration": duration,
		},
		AlertID: alert.ID,
	}

	if err := bne.agentComm.SendCommand(ctx, host, command); err != nil {
		return fmt.Errorf("failed to send block network command: %v", err)
	}

	log.Printf("Block network command sent to host %s", host)
	return nil
}

// ============================================================================
// UNBLOCK NETWORK EXECUTOR
// ============================================================================

type UnblockNetworkExecutor struct {
	agentComm AgentCommunicator
}

func NewUnblockNetworkExecutor(agentComm AgentCommunicator) *UnblockNetworkExecutor {
	return &UnblockNetworkExecutor{agentComm: agentComm}
}

func (une *UnblockNetworkExecutor) GetType() string {
	return rules.ActionUnblockNetwork
}

func (une *UnblockNetworkExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	// Получаем host из параметров или из alert
	host := ""
	if h, ok := params["host"]; ok {
		host = fmt.Sprintf("%v", h)
	} else {
		host = getStringFromEventData(alert.EventData, "pc_name")
	}

	if host == "unknown" || host == "" {
		return fmt.Errorf("host not specified")
	}

	command := AgentCommand{
		Type:       "unblock_network",
		Parameters: map[string]interface{}{},
		AlertID:    alert.ID,
	}

	if err := une.agentComm.SendCommand(ctx, host, command); err != nil {
		return fmt.Errorf("failed to send unblock network command: %v", err)
	}

	log.Printf("✓ Unblock network command sent to host %s", host)
	return nil
}

// ============================================================================
// KILL PROCESS EXECUTOR
// ============================================================================

type KillProcessExecutor struct {
	agentComm AgentCommunicator
}

func NewKillProcessExecutor(agentComm AgentCommunicator) *KillProcessExecutor {
	return &KillProcessExecutor{agentComm: agentComm}
}

func (kpe *KillProcessExecutor) GetType() string {
	return rules.ActionKillProcess
}

func (kpe *KillProcessExecutor) Execute(ctx context.Context, alert *alerts.Alert, params map[string]interface{}) error {
	processName := getStringFromEventData(alert.EventData, "process_name")
	host := getStringFromEventData(alert.EventData, "pc_name")

	if processName == "unknown" || host == "unknown" {
		return fmt.Errorf("process_name or host not found in alert data")
	}

	command := AgentCommand{
		Type: "kill_process",
		Parameters: map[string]interface{}{
			"process_name": processName,
		},
		AlertID: alert.ID,
	}

	if err := kpe.agentComm.SendCommand(ctx, host, command); err != nil {
		return fmt.Errorf("failed to send kill process command: %v", err)
	}

	log.Printf("Kill process command sent for process %s on host %s", processName, host)
	return nil
}
