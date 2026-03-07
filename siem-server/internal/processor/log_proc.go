package processor

import (
	"context"
	"fmt"
	"log"

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
func (lp *LogProc) ProcessLog(ctx context.Context, rawLog *logsstructure.RawLog) error {
	// парсинг логов
	normLog, err := lp.parser.Parser(rawLog)
	if err != nil {
		return fmt.Errorf("failed to parse log: %v", err)
	}

	// сохранение в БД
	if err := lp.storage.Store(normLog); err != nil {
		// Логируем ошибку, но продолжаем обработку
		log.Printf("Failed to save log to DB: %v", err)
	}

	// проверка правил
	if lp.ruleEngine != nil {
		if err := lp.ruleEngine.Evaluate(ctx, normLog); err != nil {
			// Логируем ошибку, но не прерываем обработку
			log.Printf("Rule evaluation failed: %v", err)
		}
	} else {
		log.Printf("Warning: Rule Engine is nil, rules not evaluated!")
	}

	return nil
}

// ProcessBatch обрабатывает пакет логов
func (lp *LogProc) ProcessBatch(ctx context.Context, rawLogs []*logsstructure.RawLog) error {
	for _, rawLog := range rawLogs {
		if err := lp.ProcessLog(ctx, rawLog); err != nil {
			log.Printf("Failed to process log: %v", err)
			// Продолжаем обработку остальных логов
		}
	}
	return nil
}
