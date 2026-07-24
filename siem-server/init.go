package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"siem-server/actions"
	"siem-server/agent"
	"siem-server/alerts"
	"siem-server/internal/delivery"
	"siem-server/internal/parsers"
	processor "siem-server/internal/processor"
	postgres "siem-server/internal/storage/postgres"
	"siem-server/rules"
	"siem-server/state"
	"siem-server/users"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// функция инициализации базы данных
// создаёт пул соединений с БД и проверяет соединение через Ping
// принимает контекст
// возвращает пул соединений и ошибку
func InitDb(ctx context.Context) (*pgxpool.Pool, error) {
	ctxDb, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctxDb, os.Getenv("DB_URL"))
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctxDb); err != nil {
		return nil, fmt.Errorf("Failed to ping database: %w", err)
	}
	log.Println("Connected to PostgreSQL")
	return pool, nil
}

type Storages struct {
	UserStorage         *users.UserStorage
	LogStorage          *postgres.LogStorage
	RuleStorage         *rules.RuleStorage
	AlertStorage        *alerts.AlertStorage
	ActionStorage       *actions.ActionStorage
	StateStorage        *state.StateStorage
	AgentStorage        *agent.AgentStorage
	CommandQueueStorage *agent.CommandQueue
}

// функция инициализации хранилищ
// принимает пул соединений
// возвращает объект хранилища
func InitStorages(pool *pgxpool.Pool) *Storages {
	return &Storages{
		UserStorage:         users.NewUserStorage(pool),
		LogStorage:          postgres.NewLogStorage(pool),
		RuleStorage:         rules.NewRuleStorage(pool),
		AlertStorage:        alerts.NewAlertStorage(pool),
		ActionStorage:       actions.NewActionStorage(pool),
		StateStorage:        state.NewStateStorage(pool),
		AgentStorage:        agent.NewAgentStorage(pool),
		CommandQueueStorage: agent.NewCommandQueue(pool),
	}

}

type Services struct {
	JwtService    *users.JWTService
	UserHandler   *users.Handler
	Notifier      *actions.MultiNotifier
	AgentCommands *actions.GRPCAgentComm
	AlertMgr      *alerts.AlertManager
	ActionDisp    *actions.Dispatcher
	RuleEngine    *rules.Engine
	LogParser     *parsers.Parser
	LogProc       *processor.LogProc
	LogHandler    *delivery.LogHandler
	AgentHandler  *agent.AgentServiceHandler
}

// функция инициализации сервисов и хендлеров
// принимает объект хранилища
// возвращает объект сервиса
func InitServices(storages *Storages) *Services {
	jwtService := users.NewJWTService()
	emailCfg := actions.EmailNotifier{
		SmtpHost:     os.Getenv("SMTP_HOST"),
		SmtpPort:     os.Getenv("SMTP_PORT"),
		SmtpUser:     os.Getenv("SMTP_USER"),
		SmtpPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("SMTP_FROM"),
		ToEmails:     []string{os.Getenv("ALERT_EMAIL")},
	}
	telegramCfg := actions.TelegramNotifier{
		BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		ChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	userHandler := users.NewHandler(storages.UserStorage, jwtService)
	multiNotifier := actions.NewMultiNotifier(&emailCfg, &telegramCfg)
	agentCommunicator := actions.NewGRPCAgentComm(storages.CommandQueueStorage)
	alertMgr := alerts.NewAlertManager(storages.AlertStorage, multiNotifier)
	actionDsp := actions.NewDispatcher(storages.ActionStorage, agentCommunicator, multiNotifier)
	ruleEngine := rules.NewEngine(
		storages.RuleStorage,
		alertMgr,
		actionDsp,
		storages.StateStorage,
	)
	logParser := parsers.NewParser()
	logProc := processor.NewLogProc(logParser, storages.LogStorage, ruleEngine)
	logHandler := delivery.NewLogHandler(logProc)
	agentHandler := agent.NewAgentServiceHandler(storages.AgentStorage, storages.CommandQueueStorage)

	return &Services{
		JwtService:    jwtService,
		UserHandler:   userHandler,
		Notifier:      multiNotifier,
		AgentCommands: agentCommunicator,
		AlertMgr:      alertMgr,
		ActionDisp:    actionDsp,
		RuleEngine:    ruleEngine,
		LogParser:     logParser,
		LogProc:       logProc,
		LogHandler:    logHandler,
		AgentHandler:  agentHandler,
	}
}
