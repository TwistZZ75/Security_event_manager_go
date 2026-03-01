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
			last_seen, last_heartbeat
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
			last_seen, last_heartbeat
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
			last_seen, last_heartbeat
		FROM agents
		WHERE status = 'online'
		  AND last_heartbeat > $1
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

// записывает heartbeat от агента
func (as *AgentStorage) RecordHeartbeat(ctx context.Context, hostname, agentVersion, status string, metrics map[string]string) error {
	// Записываем heartbeat
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	query := `
		INSERT INTO agent_heartbeats (
			hostname,
			agent_version,
			status,
			metrics,
			received_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err = as.pool.Exec(ctx, query, hostname, agentVersion, status, metricsJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record heartbeat: %v", err)
	}

	return nil
}

// сканирует строку из БД в структуру AgentInfo
func (as *AgentStorage) ScanAgent(scanner interface {
	Scan(dest ...interface{}) error
}) (*AgentInfo, error) {
	var (
		id            int64
		agentID       string
		hostname      string
		os            string
		osVersion     string
		agentVersion  string
		ipAddress     string
		metadataJSON  []byte
		status        string
		registeredAt  time.Time
		lastSeen      time.Time
		lastHeartbeat *time.Time
	)

	err := scanner.Scan(
		&id, &agentID, &hostname, &os, &osVersion, &agentVersion,
		&ipAddress, &metadataJSON, &status, &registeredAt,
		&lastSeen, &lastHeartbeat,
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
		ID:            id,
		AgentID:       agentID,
		Hostname:      hostname,
		OS:            os,
		OSVersion:     osVersion,
		AgentVersion:  agentVersion,
		IPAddress:     ipAddress,
		Metadata:      metadata,
		Status:        status,
		RegisteredAt:  registeredAt,
		LastSeen:      lastSeen,
		LastHeartbeat: lastHeartbeat,
	}, nil
}
