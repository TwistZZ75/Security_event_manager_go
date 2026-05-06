package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ActionStorage struct {
	pool *pgxpool.Pool
}

func NewActionStorage(pool *pgxpool.Pool) *ActionStorage {
	return &ActionStorage{pool: pool}
}

// Если ID == 0, создает новую запись (INSERT)
// Если ID != 0, обновляет существующую (UPDATE)
func (as *ActionStorage) LogAction(ctx context.Context, actionLog *ActionLog) error {
	if actionLog.ID == 0 {
		return as.StoreAction(ctx, actionLog)
	}
	return as.UpdateAction(ctx, actionLog)
}

func (as *ActionStorage) StoreAction(ctx context.Context, actionLog *ActionLog) error {
	// Сериализуем parameters в JSONB
	parametersJSON, err := json.Marshal(actionLog.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %v", err)
	}

	// Устанавливаем время выполнения если не установлено
	if actionLog.ExecutedAt.IsZero() {
		actionLog.ExecutedAt = time.Now()
	}

	query := `
		INSERT INTO actions_log (
			alert_id,
			action_type,
			target,
			parameters,
			status,
			executed_at,
			result,
			error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err = as.pool.QueryRow(ctx, query,
		actionLog.AlertID,
		actionLog.ActionType,
		actionLog.Target,
		parametersJSON,
		actionLog.Status,
		actionLog.ExecutedAt,
		actionLog.Result,
		actionLog.Error,
	).Scan(&actionLog.ID)

	if err != nil {
		return fmt.Errorf("failed to insert action log: %v", err)
	}

	return nil
}

func (as *ActionStorage) UpdateAction(ctx context.Context, actionLog *ActionLog) error {
	query := `
		UPDATE actions_log
		SET status = $1,
		    result = $2,
		    error = $3,
		    executed_at = $4
		WHERE id = $5
	`

	_, err := as.pool.Exec(ctx, query,
		actionLog.Status,
		actionLog.Result,
		actionLog.Error,
		time.Now(),
		actionLog.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update action log: %v", err)
	}

	return nil
}

// ScanActionLog сканирует строку из БД в структуру ActionLog
func (as *ActionStorage) ScanActionLog(scanner interface {
	Scan(dest ...interface{}) error
}) (*ActionLog, error) {
	var (
		id             int64
		alertID        int64
		actionType     string
		target         string
		parametersJSON []byte
		status         string
		executedAt     time.Time
		result         string
		errorMsg       string
	)

	err := scanner.Scan(
		&id,
		&alertID,
		&actionType,
		&target,
		&parametersJSON,
		&status,
		&executedAt,
		&result,
		&errorMsg,
	)

	if err != nil {
		return nil, err
	}

	// Десериализуем parameters
	var parameters map[string]interface{}
	if len(parametersJSON) > 0 {
		if err := json.Unmarshal(parametersJSON, &parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
		}
	}

	actionLog := &ActionLog{
		ID:         id,
		AlertID:    alertID,
		ActionType: actionType,
		Target:     target,
		Parameters: parameters,
		Status:     status,
		ExecutedAt: executedAt,
		Result:     result,
		Error:      errorMsg,
	}

	return actionLog, nil
}

// GetActionLog получает лог действия по ID
func (as *ActionStorage) GetActionLog(ctx context.Context, id string) (*ActionLog, error) {
	query := `
		SELECT 
			id,
			alert_id,
			action_type,
			target,
			parameters,
			status,
			executed_at,
			result,
			error
		FROM actions_log
		WHERE id = $1
	`

	row := as.pool.QueryRow(ctx, query, id)
	actionLog, err := as.ScanActionLog(row)
	if err != nil {
		return nil, fmt.Errorf("failed to get action log: %v", err)
	}

	return actionLog, nil
}

func (as *ActionStorage) GetActionsByAlert(ctx context.Context, alertID int64) ([]*ActionLog, error) {
	query := `
		SELECT 
			id, alert_id, action_type, target, parameters,
			status, executed_at, result, error
		FROM actions_log
		WHERE alert_id = $1
		ORDER BY executed_at DESC
	`

	rows, err := as.pool.Query(ctx, query, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to query actions: %v", err)
	}
	defer rows.Close()

	var actions []*ActionLog

	for rows.Next() {
		action, err := as.ScanActionLog(rows)
		if err != nil {
			fmt.Printf("Warning: failed to scan action log: %v\n", err)
			continue
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func (as *ActionStorage) ListActions(ctx context.Context, filter ActionFilter) ([]*ActionLog, error) {
	query := `
		SELECT 
			id, alert_id, action_type, target, parameters,
			status, executed_at, result, error
		FROM actions_log
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Добавляем фильтры
	if filter.AlertID > 0 {
		query += fmt.Sprintf(" AND alert_id = $%d", argPos)
		args = append(args, filter.AlertID)
		argPos++
	}

	if filter.ActionType != "" {
		query += fmt.Sprintf(" AND action_type = $%d", argPos)
		args = append(args, filter.ActionType)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	if filter.Target != "" {
		query += fmt.Sprintf(" AND target = $%d", argPos)
		args = append(args, filter.Target)
		argPos++
	}

	// Сортировка
	query += " ORDER BY executed_at DESC"

	// Пагинация
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := as.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query actions: %v", err)
	}
	defer rows.Close()

	var actions []*ActionLog

	for rows.Next() {
		action, err := as.ScanActionLog(rows)
		if err != nil {
			fmt.Printf("Warning: failed to scan action log: %v\n", err)
			continue
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func (as *ActionStorage) GetActionStats(ctx context.Context, from, to time.Time) (*ActionStats, error) {
	query := `
		SELECT
			COUNT(*)::BIGINT as total,
			COUNT(*) FILTER (WHERE status = 'success')::BIGINT as success,
			COUNT(*) FILTER (WHERE status = 'failed')::BIGINT as failed,
			COUNT(*) FILTER (WHERE status = 'pending')::BIGINT as pending,
			COUNT(*) FILTER (WHERE action_type = 'block_account')::BIGINT as block_account,
			COUNT(*) FILTER (WHERE action_type = 'block_network')::BIGINT as block_network,
			COUNT(*) FILTER (WHERE action_type = 'kill_process')::BIGINT as kill_process,
			COUNT(*) FILTER (WHERE action_type = 'notify')::BIGINT as notify
		FROM actions_log
		WHERE executed_at >= $1 AND executed_at <= $2
	`

	var stats ActionStats
	err := as.pool.QueryRow(ctx, query, from, to).Scan(
		&stats.Total,
		&stats.Success,
		&stats.Failed,
		&stats.Pending,
		&stats.BlockAccount,
		&stats.BlockNetwork,
		&stats.KillProcess,
		&stats.Notify,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get action stats: %v", err)
	}

	return &stats, nil
}

func (as *ActionStorage) GetFailedActions(ctx context.Context, limit int) ([]*ActionLog, error) {
	query := `
		SELECT 
			id, alert_id, action_type, target, parameters,
			status, executed_at, result, error
		FROM actions_log
		WHERE status = 'failed'
		ORDER BY executed_at DESC
		LIMIT $1
	`

	rows, err := as.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed actions: %v", err)
	}
	defer rows.Close()

	var actions []*ActionLog

	for rows.Next() {
		action, err := as.ScanActionLog(rows)
		if err != nil {
			continue
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func (as *ActionStorage) GetPendingActions(ctx context.Context) ([]*ActionLog, error) {
	query := `
		SELECT 
			id, alert_id, action_type, target, parameters,
			status, executed_at, result, error
		FROM actions_log
		WHERE status = 'pending'
		ORDER BY executed_at ASC
	`

	rows, err := as.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending actions: %v", err)
	}
	defer rows.Close()

	var actions []*ActionLog

	for rows.Next() {
		action, err := as.ScanActionLog(rows)
		if err != nil {
			continue
		}
		actions = append(actions, action)
	}

	return actions, nil
}

func (as *ActionStorage) DeleteOldActions(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM actions_log
		WHERE executed_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := as.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old actions: %v", err)
	}

	return result.RowsAffected(), nil
}

func (acst *ActionStorage) GetRecentActions(ctx context.Context, limit int) ([]*ActionLog, error) {
	filter := ActionFilter{
		Limit: limit,
	}
	return acst.ListActions(ctx, filter)
}
