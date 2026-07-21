package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"siem-server/internal/logsstructure"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// описание структуры хранимого лога
type LogStorage struct {
	pool *pgxpool.Pool
}

// функция создания нового хранилища логов
// принимает ссылку на пул соединений
// возвращает структуру LogStorage
func NewLogStorage(pool *pgxpool.Pool) *LogStorage {
	return &LogStorage{pool: pool}
}

// пишем реализацию метода Store из интерфейса LogStorage /logstructure/storage.go
// r *LogStorage - это тип данных функции(как int или void, только наш, кастомный), передаётся ссылкой,
// чтобы каждый раз не копировать структуру
// получает ссылку на нормализованный лог
// возвращает ошибку в случае неудачи
func (r *LogStorage) Store(ctx context.Context, entry *logsstructure.NormalizedLog) error {

	//проверка существования пула соединений с БД
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	//запрос к БД
	query := `
	INSERT INTO normalized_events (id, pc_name, username, event_description, 
	event_category, process_name, severity, timestamp, os, source, raw_log)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (id) DO NOTHING
	`
	// выполняем операцию Exec, игнорируем результат её выполнения, если он не ошибка
	// если получаем ошибку, метод её вернёт, в противном случае он вернёт nil, если ошибок не было
	tag, err := r.pool.Exec(ctx, query,
		entry.ID,
		entry.PC_name,
		entry.Username,
		entry.Event_description,
		entry.Event_category,
		entry.Process_name,
		entry.Severity,
		entry.Timestamp,
		entry.OS,
		entry.Source,
		entry.Raw_log,
	)
	if err != nil {
		return fmt.Errorf("insert normalized event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		slog.Warn("Duplicate event ignored", "event_id", entry.ID)
	}
	return err //возвращаем ошибку или nil
}

// scanLog сканирует одну строку результата в структуру NormalizedLog.
// Используется всеми функциями чтения.
func (r *LogStorage) scanLog(scanner interface {
	Scan(dest ...interface{}) error
}) (*logsstructure.NormalizedLog, error) {
	var (
		id               string
		pcName           string
		username         string
		eventDescription string
		eventCategory    string
		processName      string
		severity         string
		timestamp        time.Time
		os               string
		source           string
		rawLog           string
	)

	err := scanner.Scan(
		&id,
		&pcName,
		&username,
		&eventDescription,
		&eventCategory,
		&processName,
		&severity,
		&timestamp,
		&os,
		&source,
		&rawLog,
	)
	if err != nil {
		return nil, err
	}

	return &logsstructure.NormalizedLog{
		ID:                id,
		PC_name:           pcName,
		Username:          username,
		Event_description: eventDescription,
		Event_category:    eventCategory,
		Process_name:      processName,
		Severity:          severity,
		Timestamp:         timestamp,
		OS:                os,
		Source:            source,
		Raw_log:           rawLog,
	}, nil
}

// базовый SELECT без WHERE — все остальные функции добавляют условия к нему
const baseSelect = `
	SELECT id, pc_name, username, event_description, event_category,
	       process_name, severity, timestamp, os, source, raw_log
	FROM normalized_events
`

// GetByID возвращает один лог по его id (sha256-хеш).
// Если лог не найден — возвращает nil, nil.
func (r *LogStorage) GetByID(ctx context.Context, id string) (*logsstructure.NormalizedLog, error) {
	query := baseSelect + `WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	log, err := r.scanLog(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get log by id: %w", err)
	}
	return log, nil
}

// GetAll возвращает все логи, отсортированные по времени (новые первыми).
// Внимание: при большом объёме данных используй GetRecent или GetWithFilter.
func (r *LogStorage) GetAll(ctx context.Context) ([]*logsstructure.NormalizedLog, error) {
	query := baseSelect + `ORDER BY timestamp DESC`
	return r.queryMany(ctx, query)
}

// GetRecent возвращает последние N логов (новые первыми).
func (r *LogStorage) GetRecent(ctx context.Context, limit int) ([]*logsstructure.NormalizedLog, error) {
	query := baseSelect + `ORDER BY timestamp DESC LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent logs: %v", err)
	}
	defer rows.Close()

	return r.collectRows(rows)
}

// GetWithFilter возвращает логи с учётом фильтра и пагинации.
func (r *LogStorage) GetWithFilter(ctx context.Context, f logsstructure.LogFilter) ([]*logsstructure.NormalizedLog, error) {
	query := baseSelect + `WHERE 1=1`
	args := []interface{}{}
	pos := 1

	if f.PCName != "" {
		query += fmt.Sprintf(" AND pc_name = $%d", pos)
		args = append(args, f.PCName)
		pos++
	}
	if f.Username != "" {
		query += fmt.Sprintf(" AND username = $%d", pos)
		args = append(args, f.Username)
		pos++
	}
	if f.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", pos)
		args = append(args, f.Severity)
		pos++
	}
	if f.Category != "" {
		query += fmt.Sprintf(" AND event_category = $%d", pos)
		args = append(args, f.Category)
		pos++
	}
	if f.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", pos)
		args = append(args, f.Source)
		pos++
	}
	if f.OS != "" {
		query += fmt.Sprintf(" AND os = $%d", pos)
		args = append(args, f.OS)
		pos++
	}
	if !f.From.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", pos)
		args = append(args, f.From)
		pos++
	}
	if !f.To.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", pos)
		args = append(args, f.To)
		pos++
	}

	query += " ORDER BY timestamp DESC"

	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", pos)
		args = append(args, f.Limit)
		pos++
	}
	if f.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", pos)
		args = append(args, f.Offset)
		pos++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs with filter: %w", err)
	}
	defer rows.Close()

	return r.collectRows(rows)
}

// Count возвращает общее количество логов в таблице.
func (r *LogStorage) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM normalized_events`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %v", err)
	}
	return count, nil
}

// CountBySeverity возвращает карту severity → количество записей.
func (r *LogStorage) CountBySeverity(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT severity, COUNT(*) 
		FROM normalized_events 
		GROUP BY severity
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count by severity: %v", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var sev string
		var cnt int64
		if err := rows.Scan(&sev, &cnt); err != nil {
			continue
		}
		result[sev] = cnt
	}
	return result, rows.Err()
}

// CountByCategory возвращает карту event_category → количество записей.
func (r *LogStorage) CountByCategory(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT event_category, COUNT(*) 
		FROM normalized_events 
		GROUP BY event_category 
		ORDER BY COUNT(*) DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count by category: %v", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var cat string
		var cnt int64
		if err := rows.Scan(&cat, &cnt); err != nil {
			continue
		}
		result[cat] = cnt
	}
	return result, rows.Err()
}

// CountBySource возвращает карту source → количество записей.
func (r *LogStorage) CountBySource(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT source, COUNT(*) 
		FROM normalized_events 
		GROUP BY source 
		ORDER BY COUNT(*) DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count by source: %v", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var src string
		var cnt int64
		if err := rows.Scan(&src, &cnt); err != nil {
			continue
		}
		result[src] = cnt
	}
	return result, rows.Err()
}

// GetByPC возвращает все логи с конкретного компьютера (новые первыми).
func (r *LogStorage) GetByPC(ctx context.Context, pcName string, limit int) ([]*logsstructure.NormalizedLog, error) {
	return r.GetWithFilter(ctx, logsstructure.LogFilter{PCName: pcName, Limit: limit})
}

// GetByUser возвращает все логи конкретного пользователя (новые первыми).
func (r *LogStorage) GetByUser(ctx context.Context, username string, limit int) ([]*logsstructure.NormalizedLog, error) {
	return r.GetWithFilter(ctx, logsstructure.LogFilter{Username: username, Limit: limit})
}

// GetBySeverity возвращает логи с указанной важностью (новые первыми).
func (r *LogStorage) GetBySeverity(ctx context.Context, severity string, limit int) ([]*logsstructure.NormalizedLog, error) {
	return r.GetWithFilter(ctx, logsstructure.LogFilter{Severity: severity, Limit: limit})
}

// GetInTimeRange возвращает логи за указанный временной диапазон.
func (r *LogStorage) GetInTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*logsstructure.NormalizedLog, error) {
	return r.GetWithFilter(ctx, logsstructure.LogFilter{From: from, To: to, Limit: limit})
}

// DeleteOlderThan удаляет логи старше указанного срока.
// Возвращает количество удалённых строк.
func (r *LogStorage) DeleteOlderThan(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := r.pool.Exec(ctx,
		`DELETE FROM normalized_events WHERE timestamp < $1`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %v", err)
	}
	return result.RowsAffected(), nil
}

// queryMany выполняет произвольный SELECT и возвращает срез логов.
func (r *LogStorage) queryMany(ctx context.Context, query string, args ...interface{}) ([]*logsstructure.NormalizedLog, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()
	return r.collectRows(rows)
}

// collectRows итерирует по rows и собирает срез логов.
func (r *LogStorage) collectRows(rows pgx.Rows) ([]*logsstructure.NormalizedLog, error) {
	var logs []*logsstructure.NormalizedLog

	for rows.Next() {
		log, err := r.scanLog(rows)
		if err != nil {
			slog.Warn("Failed to scan log row", "error", err)
			continue
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log rows: %v", err)
	}

	return logs, nil
}
