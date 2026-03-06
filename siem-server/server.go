package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"siem-server/actions"
	"siem-server/agent"
	"siem-server/alerts"
	"siem-server/internal/delivery"
	"siem-server/internal/parsers"
	processor "siem-server/internal/processor"
	postgres "siem-server/internal/storage/postgres"
	"siem-server/proto/server/pkg/pb"
	"siem-server/rules"
	"siem-server/state"
	webserver "siem-server/web_server"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	log.Println("Starting SIEM server...")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	// Подключение к БД
	connString := os.Getenv("DB_URL")
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	log.Println("Initializing storage layers...")

	logStorage := postgres.NewLogStorage(pool)
	ruleStorage := rules.NewRuleStorage(pool)
	alertStorage := alerts.NewAlertStorage(pool)
	actionStorage := actions.NewActionStorage(pool)
	stateStorage := state.NewStateStorage(pool)
	commandQueue := agent.NewCommandQueue(pool)
	agentStorage := agent.NewAgentStorage(pool)

	notifier := actions.NewMultiNotifier()
	agentComm := actions.NewGRPCAgentComm(commandQueue)

	alertMgr := alerts.NewAlertManager(alertStorage, notifier)
	actionDsp := actions.NewDispatcher(actionStorage, agentComm, notifier)

	log.Println("Storage layers initialized")

	log.Println("Initializing Rule Engine...")

	ruleEngine := rules.NewEngine(
		ruleStorage,
		alertMgr,
		actionDsp,
		stateStorage,
	)

	// Загружаем правила из БД
	if err := ruleEngine.LoadRules(ctx); err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}

	// Выводим информацию о загруженных правилах
	rulesCount, _ := ruleStorage.GetEnabledRulesCount(ctx)
	log.Printf("Rule Engine initialized with %d enabled rules", rulesCount)

	// ============================================================================
	// PROCESSORS
	// ============================================================================
	log.Println("Initializing processors...")

	logParser := parsers.NewParser()

	// ВАЖНО: Передаем ruleEngine в LogProc!
	logProc := processor.NewLogProc(logParser, logStorage, ruleEngine)

	log.Println("Processors initialized")

	// ============================================================================
	// HANDLERS
	// ============================================================================
	log.Println("Initializing handlers...")

	logHandler := delivery.NewLogHandler(logProc)

	webServer := webserver.NewWebServer(
		agentStorage,
		alertStorage,
		actionStorage,
		ruleStorage,
	)

	// Запускаем веб-сервер в отдельной горутине
	go func() {
		if err := webServer.Start(":8080"); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	}()

	log.Println("Web interface available at http://localhost:8080")

	// TODO: Добавить другие handlers когда будут готовы
	// ruleHandler := delivery.NewRuleHandler(ruleStorage, alertStorage)
	agentHandler := agent.NewAgentServiceHandler(agentStorage, commandQueue)

	log.Println("Handlers initialized")

	// ============================================================================
	// BACKGROUND TASKS
	// ============================================================================
	log.Println("Starting background tasks...")

	// Задача 1: Очистка истекших состояний (каждые 5 минут)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := stateStorage.DeleteExpiredStates(ctx); err != nil {
				log.Printf("Failed to cleanup expired states: %v", err)
			}
		}
	}()

	// Задача 2: Перезагрузка правил (каждые 5 минут)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := ruleEngine.ReloadRules(ctx); err != nil {
				log.Printf("Failed to reload rules: %v", err)
			} else {
				count, _ := ruleStorage.GetEnabledRulesCount(ctx)
				log.Printf("✓ Rules reloaded: %d enabled", count)
			}
		}
	}()

	log.Println("Background tasks started")

	// ============================================================================
	// gRPC SERVER
	// ============================================================================
	log.Println("Starting gRPC server...")

	portStr := os.Getenv("PORT")
	listener, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("Failed to listen port: %v", err)
	}

	grpcServ := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	// Регистрируем сервисы
	pb.RegisterLogServiceServer(grpcServ, logHandler)
	pb.RegisterAgentServiceServer(grpcServ, agentHandler)

	reflection.Register(grpcServ)

	log.Printf("Server is ready on %s", portStr)
	log.Println("Press Ctrl+C to stop")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		grpcServ.GracefulStop()
		log.Println("Server stopped")
	}()

	// Запускаем сервер
	if err := grpcServ.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
