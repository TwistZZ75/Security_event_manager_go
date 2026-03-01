package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandQueue struct {
	pool *pgxpool.Pool
}

func NewCommandQueue(pool *pgxpool.Pool) *CommandQueue {
	return &CommandQueue{pool: pool}
}

// создает новую команду для агента
func (cq *CommandQueue) CreateCommand(ctx context.Context, cmd *AgentCommand) error {
	// Сериализуем parameters
	parametersJSON, err := json.Marshal(cmd.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %v", err)
	}

	query := `
		INSERT INTO agent_commands (
			hostname,
			command_type,
			parameters,
			alert_id,
			priority,
			status,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err = cq.pool.QueryRow(ctx, query,
		cmd.Hostname,
		cmd.CommandType,
		parametersJSON,
		cmd.AlertID,
		cmd.Priority,
		"pending",
		time.Now(),
	).Scan(&cmd.ID)

	if err != nil {
		return fmt.Errorf("failed to create command: %v", err)
	}

	return nil
}

// получает список ожидающих команд для агента
func (cq *CommandQueue) GetPendingCommands(ctx context.Context, hostname string) ([]*AgentCommand, error) {
	query := `
		WITH updated AS (
			UPDATE agent_commands
			SET status = 'sent'
			WHERE hostname = $1
			AND status = 'pending'
			RETURNING id, command_type, parameters, alert_id, priority, created_at
		)
		SELECT * FROM updated;
	`

	rows, err := cq.pool.Query(ctx, query, hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending commands: %v", err)
	}
	defer rows.Close()

	var commands []*AgentCommand

	for rows.Next() {
		var (
			id             int64
			commandType    string
			parametersJSON []byte
			alertID        int64
			priority       string
			createdAt      time.Time
		)

		err := rows.Scan(&id, &commandType, &parametersJSON, &alertID, &priority, &createdAt)
		if err != nil {
			fmt.Printf("Warning: failed to scan command: %v\n", err)
			continue
		}

		// Десериализуем parameters
		var parameters map[string]string
		if len(parametersJSON) > 0 {
			if err := json.Unmarshal(parametersJSON, &parameters); err != nil {
				fmt.Printf("Warning: failed to unmarshal parameters: %v\n", err)
				continue
			}
		}

		cmd := &AgentCommand{
			ID:          id,
			Hostname:    hostname,
			CommandType: commandType,
			Parameters:  parameters,
			AlertID:     alertID,
			Priority:    priority,
			CreatedAt:   createdAt,
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

// обновляет статус команды
func (cq *CommandQueue) UpdateCommandStatus(ctx context.Context, commandID int64, status, result, errorMsg string) error {
	query := `
		UPDATE agent_commands
		SET status = $1,
		    result = $2,
		    error = $3,
		    completed_at = $4
		WHERE id = $5
	`

	_, err := cq.pool.Exec(ctx, query, status, result, errorMsg, time.Now(), commandID)
	if err != nil {
		return fmt.Errorf("failed to update command status: %v", err)
	}

	return nil
}

// получает команду по ID
func (cq *CommandQueue) GetCommand(ctx context.Context, commandID int64) (*AgentCommand, error) {
	query := `
		SELECT 
			id, hostname, command_type, parameters, alert_id,
			priority, status, created_at, sent_at, completed_at,
			result, error
		FROM agent_commands
		WHERE id = $1
	`

	row := cq.pool.QueryRow(ctx, query, commandID)
	cmd, err := cq.ScanCommand(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("command not found: %d", commandID)
		}
		return nil, fmt.Errorf("failed to get command: %v", err)
	}

	return cmd, nil
}

// получает все команды для конкретного алерта
func (cq *CommandQueue) GetCommandsByAlert(ctx context.Context, alertID int64) ([]*AgentCommand, error) {
	query := `
		SELECT 
			id, hostname, command_type, parameters, alert_id,
			priority, status, created_at, sent_at, completed_at,
			result, error
		FROM agent_commands
		WHERE alert_id = $1
		ORDER BY created_at DESC
	`

	rows, err := cq.pool.Query(ctx, query, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %v", err)
	}
	defer rows.Close()

	var commands []*AgentCommand
	for rows.Next() {
		cmd, err := cq.ScanCommand(rows)
		if err != nil {
			continue
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// получает команды для конкретного хоста
func (cq *CommandQueue) GetCommandsByHost(ctx context.Context, hostname string, limit int) ([]*AgentCommand, error) {
	query := `
		SELECT 
			id, hostname, command_type, parameters, alert_id,
			priority, status, created_at, sent_at, completed_at,
			result, error
		FROM agent_commands
		WHERE hostname = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := cq.pool.Query(ctx, query, hostname, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %v", err)
	}
	defer rows.Close()

	var commands []*AgentCommand
	for rows.Next() {
		cmd, err := cq.ScanCommand(rows)
		if err != nil {
			continue
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// отменяет команду
func (cq *CommandQueue) CancelCommand(ctx context.Context, commandID int64) error {
	query := `
		UPDATE agent_commands
		SET status = 'cancelled',
		    completed_at = $1
		WHERE id = $2 AND status IN ('pending', 'sent')
	`

	result, err := cq.pool.Exec(ctx, query, time.Now(), commandID)
	if err != nil {
		return fmt.Errorf("failed to cancel command: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("command cannot be cancelled or not found")
	}

	return nil
}

// удаляет старые команды
func (cq *CommandQueue) DeleteOldCommands(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM agent_commands
		WHERE created_at < $1
		  AND status IN ('success', 'failed', 'cancelled')
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := cq.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old commands: %v", err)
	}

	return result.RowsAffected(), nil
}

// сканирует строку из БД в структуру AgentCommand
func (cq *CommandQueue) ScanCommand(scanner interface {
	Scan(dest ...interface{}) error
}) (*AgentCommand, error) {
	var (
		id             int64
		hostname       string
		commandType    string
		parametersJSON []byte
		alertID        int64
		priority       string
		status         string
		createdAt      time.Time
		sentAt         *time.Time
		completedAt    *time.Time
		result         string
		errorMsg       string
	)

	err := scanner.Scan(
		&id, &hostname, &commandType, &parametersJSON, &alertID,
		&priority, &status, &createdAt, &sentAt, &completedAt,
		&result, &errorMsg,
	)

	if err != nil {
		return nil, err
	}

	var parameters map[string]string
	if len(parametersJSON) > 0 {
		if err := json.Unmarshal(parametersJSON, &parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
		}
	}

	return &AgentCommand{
		ID:          id,
		Hostname:    hostname,
		CommandType: commandType,
		Parameters:  parameters,
		AlertID:     alertID,
		Priority:    priority,
		Status:      status,
		CreatedAt:   createdAt,
		SentAt:      sentAt,
		CompletedAt: completedAt,
		Result:      result,
		Error:       errorMsg,
	}, nil
}
