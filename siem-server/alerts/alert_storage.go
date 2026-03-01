package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertStorage struct {
	pool *pgxpool.Pool
}

func NewAlertStorage(pool *pgxpool.Pool) *AlertStorage {
	return &AlertStorage{pool: pool}
}

func (ast *AlertStorage) CreateAlert(ctx context.Context, alert *Alert) error {
	// Сериализуем event_data в JSONB
	eventDataJSON, err := json.Marshal(alert.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %v", err)
	}

	query := `
		INSERT INTO alerts (
			rule_id,
			rule_name,
			severity,
			title,
			description,
			event_data,
			status,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err = ast.pool.QueryRow(ctx, query,
		alert.RuleID,
		alert.RuleName,
		alert.Severity,
		alert.Title,
		alert.Description,
		eventDataJSON,
		alert.Status,
		alert.CreatedAt,
	).Scan(&alert.ID)

	if err != nil {
		return fmt.Errorf("failed to create alert: %v", err)
	}

	return nil
}

func (ast *AlertStorage) GetAlert(ctx context.Context, id int64) (*Alert, error) {
	query := `
		SELECT 
			id,
			rule_id,
			rule_name,
			severity,
			title,
			description,
			event_data,
			status,
			created_at,
			acknowledged_at,
			acknowledged_by,
			resolved_at,
			resolved_by,
			notes
		FROM alerts
		WHERE id = $1
	`

	row := ast.pool.QueryRow(ctx, query, id)
	alert, err := ast.ScanAlert(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("alert not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get alert: %v", err)
	}

	return alert, nil
}

func (ast *AlertStorage) ScanAlert(scanner interface {
	Scan(dest ...interface{}) error
}) (*Alert, error) {
	var (
		id             int64
		ruleID         string
		ruleName       string
		severity       string
		title          string
		description    string
		eventDataJSON  []byte
		status         string
		createdAt      time.Time
		acknowledgedAt *time.Time
		acknowledgedBy string
		resolvedAt     *time.Time
		resolvedBy     string
		notes          string
	)

	err := scanner.Scan(
		&id,
		&ruleID,
		&ruleName,
		&severity,
		&title,
		&description,
		&eventDataJSON,
		&status,
		&createdAt,
		&acknowledgedAt,
		&acknowledgedBy,
		&resolvedAt,
		&resolvedBy,
		&notes,
	)

	if err != nil {
		return nil, err
	}

	// Десериализуем event_data
	var eventData map[string]interface{}
	if len(eventDataJSON) > 0 {
		if err := json.Unmarshal(eventDataJSON, &eventData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %v", err)
		}
	}

	alert := &Alert{
		ID:             id,
		RuleID:         ruleID,
		RuleName:       ruleName,
		Severity:       severity,
		Title:          title,
		Description:    description,
		EventData:      eventData,
		Status:         status,
		CreatedAt:      createdAt,
		AcknowledgedAt: acknowledgedAt,
		AcknowledgedBy: acknowledgedBy,
		ResolvedAt:     resolvedAt,
		ResolvedBy:     resolvedBy,
		Notes:          notes,
	}

	return alert, nil
}

func (ast *AlertStorage) GetAlerts(ctx context.Context, filter AlertFilter) ([]*Alert, error) {
	// Строим динамический запрос
	query := `
		SELECT 
			id, rule_id, rule_name, severity, title, description,
			event_data, status, created_at, acknowledged_at,
			acknowledged_by, resolved_at, resolved_by, notes
		FROM alerts
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Добавляем фильтры
	if filter.RuleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argPos)
		args = append(args, filter.RuleID)
		argPos++
	}

	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argPos)
		args = append(args, filter.Severity)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	if !filter.From.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, filter.From)
		argPos++
	}

	if !filter.To.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, filter.To)
		argPos++
	}

	// Сортировка
	query += " ORDER BY created_at DESC"

	// Пагинация - разделение большого объёма на меньшие (страницы)
	// выводить не весь объём алертов, а по 100/50/25 на странице
	if filter.Limit > 0 { //по сколько выводить
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 { //с какого начинать на данной странице
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	// Выполняем запрос
	rows, err := ast.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %v", err)
	}
	defer rows.Close()

	var alertsList []*Alert

	for rows.Next() {
		alert, err := ast.ScanAlert(rows)
		if err != nil {
			fmt.Printf("Warning: failed to scan alert: %v\n", err)
			continue
		}
		alertsList = append(alertsList, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %v", err)
	}

	return alertsList, nil
}

func (ast *AlertStorage) UpdateAlert(ctx context.Context, id int64, status string, updatedBy string, notes string) error {
	now := time.Now()

	var query string
	var args []interface{}

	switch status {
	case StatusAcknowledged:
		query = `
			UPDATE alerts
			SET status = $1,
			    acknowledged_at = $2,
			    acknowledged_by = $3,
			    notes = $4
			WHERE id = $5
		`
		args = []interface{}{status, now, updatedBy, notes, id}

	case StatusResolved:
		query = `
			UPDATE alerts
			SET status = $1,
			    resolved_at = $2,
			    resolved_by = $3,
			    notes = $4
			WHERE id = $5
		`
		args = []interface{}{status, now, updatedBy, notes, id}

	case StatusFalsePositive:
		query = `
			UPDATE alerts
			SET status = $1,
			    resolved_at = $2,
			    resolved_by = $3,
			    notes = $4
			WHERE id = $5
		`
		args = []interface{}{status, now, updatedBy, notes, id}

	default:
		return fmt.Errorf("invalid alert status: %s", status)
	}

	result, err := ast.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alert status: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("alert not found: %d", id)
	}

	return nil
}

func (ast *AlertStorage) GetAlertStats(ctx context.Context, from, to time.Time) (*AlertStats, error) {
	query := `
		SELECT
			COUNT(*)::BIGINT as total,
			COUNT(*) FILTER (WHERE severity = 'critical')::BIGINT as critical,
			COUNT(*) FILTER (WHERE severity = 'high')::BIGINT as high,
			COUNT(*) FILTER (WHERE severity = 'medium')::BIGINT as medium,
			COUNT(*) FILTER (WHERE severity = 'low')::BIGINT as low,
			COUNT(*) FILTER (WHERE status = 'open')::BIGINT as open,
			COUNT(*) FILTER (WHERE status = 'acknowledged')::BIGINT as acknowledged,
			COUNT(*) FILTER (WHERE status = 'resolved')::BIGINT as resolved,
			COUNT(*) FILTER (WHERE status = 'false_positive')::BIGINT as false_positive,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as avg_resolution_hours
		FROM alerts
		WHERE created_at >= $1 AND created_at <= $2
	` //BIGINT потому что функция COUNT возвращает BIGINT

	var (
		total         int64
		critical      int64
		high          int64
		medium        int64
		low           int64
		open          int64
		acknowledged  int64
		resolved      int64
		falsePositive int64
		avgResolution *float64
	)

	err := ast.pool.QueryRow(ctx, query, from, to).Scan(
		&total,
		&critical,
		&high,
		&medium,
		&low,
		&open,
		&acknowledged,
		&resolved,
		&falsePositive,
		&avgResolution,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats: %v", err)
	}

	stats := &AlertStats{
		Total: total,
		BySeverity: map[string]int64{
			"critical": critical,
			"high":     high,
			"medium":   medium,
			"low":      low,
		},
		ByStatus: map[string]int64{
			"open":           open,
			"acknowledged":   acknowledged,
			"resolved":       resolved,
			"false_positive": falsePositive,
		},
	}

	if avgResolution != nil {
		stats.AverageResTime = *avgResolution
	}

	return stats, nil
}

// GetAlertsByRule возвращает все алерты для конкретного правила
func (ast *AlertStorage) GetAlertsByRule(ctx context.Context, ruleID string, limit int) ([]*Alert, error) {
	filter := AlertFilter{
		RuleID: ruleID,
		Limit:  limit,
	}
	return ast.GetAlerts(ctx, filter)
}

// GetOpenAlerts возвращает все открытые алерты
func (ast *AlertStorage) GetOpenAlerts(ctx context.Context) ([]*Alert, error) {
	filter := AlertFilter{
		Status: StatusOpen,
	}
	return ast.GetAlerts(ctx, filter)
}

// GetCriticalAlerts возвращает критичные алерты за последние 24 часа
func (ast *AlertStorage) GetCriticalAlerts(ctx context.Context) ([]*Alert, error) {
	filter := AlertFilter{
		Severity: "critical",
		Status:   StatusOpen,
		From:     time.Now().Add(-24 * time.Hour),
		Limit:    100,
	}
	return ast.GetAlerts(ctx, filter)
}

// GetRecentAlerts возвращает последние N алертов
func (ast *AlertStorage) GetRecentAlerts(ctx context.Context, limit int) ([]*Alert, error) {
	filter := AlertFilter{
		Limit: limit,
	}
	return ast.GetAlerts(ctx, filter)
}

// CountAlerts возвращает общее количество алертов с учетом фильтра
func (ast *AlertStorage) CountAlerts(ctx context.Context, filter AlertFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM alerts WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if filter.RuleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argPos)
		args = append(args, filter.RuleID)
		argPos++
	}

	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argPos)
		args = append(args, filter.Severity)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	if !filter.From.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, filter.From)
		argPos++
	}

	if !filter.To.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, filter.To)
	}

	var count int64
	err := ast.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count alerts: %v", err)
	}

	return count, nil
}

// DeleteOldAlerts удаляет алерты старше указанного периода
func (ast *AlertStorage) DeleteOldAlerts(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM alerts
		WHERE created_at < $1
		  AND status IN ('resolved', 'false_positive')
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := ast.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old alerts: %v", err)
	}

	return result.RowsAffected(), nil
}

// GetAlertTimeline возвращает временную линию алертов (по часам)
func (ast *AlertStorage) GetAlertTimeline(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	query := `
		SELECT 
			date_trunc('hour', created_at) as hour,
			COUNT(*) as count
		FROM alerts
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY hour
		ORDER BY hour
	`

	rows, err := ast.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert timeline: %v", err)
	}
	defer rows.Close()

	timeline := make(map[string]int64)

	for rows.Next() {
		var hour time.Time
		var count int64

		if err := rows.Scan(&hour, &count); err != nil {
			return nil, fmt.Errorf("failed to scan timeline row: %v", err)
		}

		timeline[hour.Format(time.RFC3339)] = count
	}

	return timeline, nil
}

// GetTopTriggeredRules возвращает топ N правил по количеству алертов
func (ast *AlertStorage) GetTopTriggeredRules(ctx context.Context, limit int, from, to time.Time) ([]struct {
	RuleID   string
	RuleName string
	Count    int64
}, error) {
	query := `
		SELECT 
			rule_id,
			rule_name,
			COUNT(*) as count
		FROM alerts
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY rule_id, rule_name
		ORDER BY count DESC
		LIMIT $3
	`

	rows, err := ast.pool.Query(ctx, query, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top rules: %v", err)
	}
	defer rows.Close()

	var results []struct {
		RuleID   string
		RuleName string
		Count    int64
	}

	for rows.Next() {
		var result struct {
			RuleID   string
			RuleName string
			Count    int64
		}

		if err := rows.Scan(&result.RuleID, &result.RuleName, &result.Count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		results = append(results, result)
	}

	return results, nil
}
