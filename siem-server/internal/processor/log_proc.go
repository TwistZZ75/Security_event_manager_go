package processor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"siem-server/internal/logsstructure"
	"siem-server/internal/parsers"
	"siem-server/internal/storage/postgres"
	"siem-server/rules"
)

type LogProc struct {
	parser     *parsers.Parser
	storage    *postgres.LogStorage
	ruleEngine *rules.Engine
}

// NewLogProc создает новый процессор логов
func NewLogProc(
	parser *parsers.Parser,
	storage *postgres.LogStorage,
	ruleEngine *rules.Engine,
) *LogProc {
	return &LogProc{
		parser:     parser,
		storage:    storage,
		ruleEngine: ruleEngine,
	}
}

// ProcessLog обрабатывает сырой лог
// code review: ошибка с контекстом (зачем-то создавался новый background, вместо передачи контекста из функции вызова)
// Не возвращалась и не логировалась ошибка при неудавшемся сохранении в базу
// TODO: использовать Kafka или RabbitMQ для реализации DLQ (dead letter queue), потому что сейчас принимаемые события могут
// теряться при ошибке сохранения в БД
// убрать проверку существования ruleEngine, так как она по сути проходится при создании экземпляра LogProc в конструкторе
func (lp *LogProc) ProcessLog(ctx context.Context, rawLog *logsstructure.RawLog) error {
	// парсинг логов
	normLog, err := lp.parser.Parse(rawLog)
	if err != nil {
		return fmt.Errorf("failed to parse log: %v", err)
	}

	// сохранение в БД
	ctxDb, cancel := context.WithTimeout(ctx, 3*time.Second)
	if err := lp.storage.Store(ctxDb, normLog); err != nil {
		// Логируем ошибку, но продолжаем обработку
		slog.Error("Failed to save log to DB", "error", err, "log ID", normLog.ID)
		cancel()
		return fmt.Errorf("Event wasn't save, error: %w; log ID %v", err, normLog.ID)
	}
	cancel()

	// проверка правил
	if err := lp.ruleEngine.Evaluate(ctx, normLog); err != nil {
		// Логируем ошибку, но не прерываем обработку
		slog.Error("Rule evaluation failed", "error", err)
		return fmt.Errorf("Rule evaluation error %w", err)
	}

	return nil
}

// ProcessBatch обрабатывает пакет логов
// code review: исправить логирование ошибки на slog и возвращать ошибку при неудачной обработке
// отсутствует обработка паники
func (lp *LogProc) ProcessBatch(ctx context.Context, rawLogs []*logsstructure.RawLog) error {
	var errs []error
	defer func() {
		if r := recover(); r != nil {
			errs = append(errs, fmt.Errorf("Panic while processing bath %v", r))
		}
	}()

	for i, rawLog := range rawLogs {
		if err := lp.ProcessLog(ctx, rawLog); err != nil {
			slog.Error("Failed to process log", "error", err, "event", i)
			errs = append(errs, fmt.Errorf("log %d: %w", i, err))
		}
	}
	return errors.Join(errs...) //возвращаем все ошибки (их срез), если они были и возвращаем nil, если ошибок не было(errs пуст)
}
