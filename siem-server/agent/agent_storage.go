package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// реализует интерфейс для работы с агентами
type AgentStorage struct {
	pool *pgxpool.Pool
}

// создает новое хранилище агентов
func NewAgentStorage(pool *pgxpool.Pool) *AgentStorage {
	return &AgentStorage{pool: pool}
}

// регистрирует новый агент
func (as *AgentStorage) RegisterAgent(ctx context.Context, agent *AgentInfo) error {
	// Сериализуем metadata
	metadataJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	query := `
		INSERT INTO agents (
			agent_id,
			hostname,
			os,
			os_version,
			agent_version,
			ip_address,
			metadata,
			status,
			registered_at,
			last_seen
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (hostname) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			os = EXCLUDED.os,
			os_version = EXCLUDED.os_version,
			agent_version = EXCLUDED.agent_version,
			ip_address = EXCLUDED.ip_address,
			metadata = EXCLUDED.metadata,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen
		RETURNING id
	`

	now := time.Now()
	err = as.pool.QueryRow(ctx, query,
		agent.AgentID,
		agent.Hostname,
		agent.OS,
		agent.OSVersion,
		agent.AgentVersion,
		agent.IPAddress,
		metadataJSON,
		"online",
		now,
		now,
	).Scan(&agent.ID)

	if err != nil {
		return fmt.Errorf("failed to register agent: %v", err)
	}

	return nil
}

// обновляет статус агента
func (as *AgentStorage) UpdateAgentStatus(ctx context.Context, hostname string, status string, lastSeen time.Time) error {
	query := `
		UPDATE agents
		SET status = $1,
		    last_seen = $2
		WHERE hostname = $3
	`

	_, err := as.pool.Exec(ctx, query, status, lastSeen, hostname)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %v", err)
	}

	return nil
}

// получает информацию об агенте
func (as *AgentStorage) GetAgent(ctx context.Context, hostname string) (*AgentInfo, error) {
	query := `
		SELECT 
			id, agent_id, hostname, os, os_version, agent_version,
			ip_address, metadata, status, registered_at,
			last_seen
		FROM agents
		WHERE hostname = $1
	`

	row := as.pool.QueryRow(ctx, query, hostname)
	agent, err := as.ScanAgent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %s", hostname)
		}
		return nil, fmt.Errorf("failed to get agent: %v", err)
	}

	return agent, nil
}

// возвращает список всех агентов
func (as *AgentStorage) ListAgents(ctx context.Context) ([]*AgentInfo, error) {
	query := `
		SELECT 
			id, agent_id, hostname, os, os_version, agent_version,
			ip_address, metadata, status, registered_at,
			last_seen
		FROM agents
		ORDER BY last_seen DESC
	`

	rows, err := as.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %v", err)
	}
	defer rows.Close()

	var agents []*AgentInfo
	for rows.Next() {
		agent, err := as.ScanAgent(rows)
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// возвращает список онлайн агентов
func (as *AgentStorage) GetOnlineAgents(ctx context.Context) ([]*AgentInfo, error) {
	query := `
		SELECT 
			id, agent_id, hostname, os, os_version, agent_version,
			ip_address, metadata, status, registered_at,
			last_seen
		FROM agents
		WHERE status = 'online'
		  AND last_seen > $1
		ORDER BY last_seen DESC
	`

	cutoff := time.Now().Add(-2 * time.Minute)
	rows, err := as.pool.Query(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query online agents: %v", err)
	}
	defer rows.Close()

	var agents []*AgentInfo
	for rows.Next() {
		agent, err := as.ScanAgent(rows)
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// сканирует строку из БД в структуру AgentInfo
func (as *AgentStorage) ScanAgent(scanner interface {
	Scan(dest ...interface{}) error
}) (*AgentInfo, error) {
	var (
		id           int64
		agentID      string
		hostname     string
		os           string
		osVersion    string
		agentVersion string
		ipAddress    string
		metadataJSON []byte
		status       string
		registeredAt time.Time
		lastSeen     time.Time
	)

	err := scanner.Scan(
		&id, &agentID, &hostname, &os, &osVersion, &agentVersion,
		&ipAddress, &metadataJSON, &status, &registeredAt,
		&lastSeen,
	)

	if err != nil {
		return nil, err
	}

	var metadata map[string]string
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
		}
	}

	return &AgentInfo{
		ID:           id,
		AgentID:      agentID,
		Hostname:     hostname,
		OS:           os,
		OSVersion:    osVersion,
		AgentVersion: agentVersion,
		IPAddress:    ipAddress,
		Metadata:     metadata,
		Status:       status,
		RegisteredAt: registeredAt,
		LastSeen:     lastSeen,
	}, nil
}

// UpsertAgent создает или обновляет агента
func (as *AgentStorage) UpsertAgent(ctx context.Context, agent *AgentInfo) error {
	metadataJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	query := `
		INSERT INTO agents (
			agent_id, hostname, ip_address, os, os_version, 
			agent_version, status, last_seen, metadata, registered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (hostname) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			ip_address = EXCLUDED.ip_address,
			os = EXCLUDED.os,
			os_version = EXCLUDED.os_version,
			agent_version = EXCLUDED.agent_version,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen,
			metadata = EXCLUDED.metadata
	`

	_, err = as.pool.Exec(ctx, query,
		agent.AgentID,
		agent.Hostname,
		agent.IPAddress,
		agent.OS,
		agent.OSVersion,
		agent.AgentVersion,
		agent.Status,
		agent.LastSeen,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert agent: %v", err)
	}

	return nil
}

// UpdateAgentLastSeen обновляет время последней активности агента
func (as *AgentStorage) UpdateAgentLastSeen(ctx context.Context, hostname string) error {
	query := `
		UPDATE agents
		SET last_seen = NOW(),
		    status = 'online'
		WHERE hostname = $1
	`

	_, err := as.pool.Exec(ctx, query, hostname)
	if err != nil {
		return fmt.Errorf("failed to update last_seen: %v", err)
	}

	return nil
}

// GetAgentByHostname получает агента по hostname
func (as *AgentStorage) GetAgentByHostname(ctx context.Context, hostname string) (*AgentInfo, error) {
	query := `
		SELECT 
			id, agent_id, hostname, ip_address, os, os_version,
			agent_version, status, last_seen, registered_at, metadata
		FROM agents
		WHERE hostname = $1
	`

	var agent AgentInfo
	var metadataJSON []byte

	err := as.pool.QueryRow(ctx, query, hostname).Scan(
		&agent.ID,
		&agent.AgentID,
		&agent.Hostname,
		&agent.IPAddress,
		&agent.OS,
		&agent.OSVersion,
		&agent.AgentVersion,
		&agent.Status,
		&agent.LastSeen,
		&agent.RegisteredAt,
		&metadataJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %v", err)
	}

	// Десериализуем metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &agent.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
		}
	}

	return &agent, nil
}

// GetPendingCommandsForHost получает pending команды для хоста
func (cq *CommandQueue) GetPendingCommandsForHost(ctx context.Context, hostname string) ([]*AgentCommand, error) {
	query := `
		SELECT 
			id, hostname, command_type, parameters, alert_id, 
			priority, status, created_at
		FROM agent_commands
		WHERE hostname = $1 
		  AND status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT 10
	`

	rows, err := cq.pool.Query(ctx, query, hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commands: %v", err)
	}
	defer rows.Close()

	var commands []*AgentCommand

	for rows.Next() {
		var cmd AgentCommand
		var parametersJSON []byte

		err := rows.Scan(
			&cmd.ID,
			&cmd.Hostname,
			&cmd.CommandType,
			&parametersJSON,
			&cmd.AlertID,
			&cmd.Priority,
			&cmd.Status,
			&cmd.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %v", err)
		}

		// Десериализуем parameters
		if len(parametersJSON) > 0 {
			var params map[string]interface{}
			if err := json.Unmarshal(parametersJSON, &params); err != nil {
				return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
			}

			// Конвертируем в map[string]string
			cmd.Parameters = make(map[string]string)
			for k, v := range params {
				cmd.Parameters[k] = fmt.Sprintf("%v", v)
			}
		}

		commands = append(commands, &cmd)
	}

	// После получения команд, обновляем их статус на "sent"
	if len(commands) > 0 {
		ids := make([]int64, len(commands))
		for i, cmd := range commands {
			ids[i] = cmd.ID
		}

		updateQuery := `
			UPDATE agent_commands
			SET status = 'sent',
			    updated_at = NOW()
			WHERE id = ANY($1)
		`
		_, err = cq.pool.Exec(ctx, updateQuery, ids)
		if err != nil {
			// Не критично, логируем и продолжаем
			fmt.Printf("Warning: failed to update command status to 'sent': %v\n", err)
		}
	}

	return commands, nil
}

// UpdateCommandStatus обновляет статус выполнения команды
func (cq *CommandQueue) UpdateCommandStatus(ctx context.Context, commandID int64, status, result, errorMsg string) error {
	query := `
		UPDATE agent_commands
		SET status = $1,
		    result = $2,
		    error = $3,
		    executed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $4
	`

	_, err := cq.pool.Exec(ctx, query, status, result, errorMsg, commandID)
	if err != nil {
		return fmt.Errorf("failed to update command status: %v", err)
	}

	return nil
}

// MarkOfflineAgents помечает агентов как offline если last_seen > 5 минут
func (as *AgentStorage) MarkOfflineAgents(ctx context.Context) error {
	query := `
		UPDATE agents
		SET status = 'offline'
		WHERE status = 'online'
		  AND last_seen < NOW() - INTERVAL '5 minutes'
	`

	result, err := as.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to mark offline agents: %v", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Marked %d agents as offline\n", rowsAffected)
	}

	return nil
}
