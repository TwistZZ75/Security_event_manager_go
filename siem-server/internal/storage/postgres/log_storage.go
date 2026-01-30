package postgres

import (
	"context"
	"fmt"
	logsstructure "siem-server/internal/logsstructure"

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
func (r *LogStorage) Store(entry *logsstructure.NormalizedLog) error {

	//создаём корневой контекст
	ctx := context.Background()

	//проверка существования пула соединений с БД
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	//запрос к БД
	query := `
	INSERT INTO normalized_events (id, raw_event_id, pc_name, username, event_description, 
	event_category, process_name, severity, timestamp, os, source)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (id) DO NOTHING
	`
	// выполняем операцию Exec, игнорируем результат её выполнения, если он не ошибка
	// если получаем ошибку, метод её вернёт, в противном случае он вернёт nil, если ошибок не было
	_, error := r.pool.Exec(ctx, query,
		entry.ID,
		entry.Raw_log_id,
		entry.PC_name,
		entry.Username,
		entry.Event_description,
		entry.Event_category,
		entry.Process_name,
		entry.Severity,
		entry.Timestamp,
		entry.OS,
		entry.Source,
	)
	return error //возвращаем ошибку или nil
}
