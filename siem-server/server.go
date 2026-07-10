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
	"siem-server/users"
	webserver "siem-server/web_server"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

//Концептуальные ошибки:
//1. Отсутствие graceful shutdown для веб-сервера и фоновых горутин (сейчас ожидается только завершение grpc сервера)
//TODO: прокинуть контекст, чтобы при завершении работы сервера остальные элементы тоже знали о том, что пора завершаться
//2. Нагруженный main
//TODO: вынести инициализации в отдельные функции
//3. Захардкоженные значения порта, интервалы тикеров, ограничений grpc
//TODO: вынести это всё в config файл, будь то env или yaml
//4. Использование log для логирования
//TODO: использовать slog/zap/zerolog
//5. Использование gRPC reflection
//TODO: изменить подход

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

	//Ошибка: пинг использует Context.Background, это может привести к зависанию, если БД недоступна
	//TODO: использовать контекст с таймаутом
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Инициализация системы пользователей
	userStorage := users.NewUserStorage(pool)
	jwtService := users.NewJWTService()
	userHandler := users.NewHandler(userStorage, jwtService)

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
	//Ошибка: использование Context.Background
	//TODO: использовать контест с таймаутом
	if err := ruleEngine.LoadRules(ctx); err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}

	// Выводим информацию о загруженных правилах
	//Ошибка: игнорирование ошибки возвращаемой функцией
	//rulesCount, _ := ruleStorage.GetEnabledRulesCount(ctx)
	ruleCount, err := ruleStorage.GetEnabledRulesCount(ctx)
	if err != nil {
		//TODO: добавить логирование ошибки
	}
	log.Printf("Rule Engine initialized with %d enabled rules", ruleCount)

	// ============================================================================
	// PROCESSORS
	// ============================================================================
	log.Println("Initializing processors...")

	logParser := parsers.NewParser()

	logProc := processor.NewLogProc(logParser, logStorage, ruleEngine)

	log.Println("Processors initialized")

	// ============================================================================
	// HANDLERS
	// ============================================================================
	log.Println("Initializing handlers...")

	logHandler := delivery.NewLogHandler(logProc)

	webServer := webserver.NewWebServer(
		agentStorage, alertStorage, actionStorage, ruleStorage, logStorage,
		userStorage, jwtService, userHandler,
	)

	// Запускаем веб-сервер в отдельной горутине
	go func() {
		if err := webServer.Start(":8080"); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	}()

	// удаление истёкших refresh токенов (каждый час)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_, err := pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
			if err != nil {
				log.Printf("Failed to cleanup refresh tokens: %v", err)
			}
		}
	}()

	log.Println("Web interface available at http://localhost:8080")

	agentHandler := agent.NewAgentServiceHandler(agentStorage, commandQueue)

	log.Println("Handlers initialized")

	// ============================================================================
	// BACKGROUND TASKS
	// ============================================================================
	log.Println("Starting background tasks...")

	// Очистка истекших состояний (каждые 5 минут)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := stateStorage.DeleteExpiredStates(ctx); err != nil {
				log.Printf("Failed to cleanup expired states: %v", err)
			}
		}
	}()

	// Перезагрузка правил (каждые 5 минут)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := ruleEngine.ReloadRules(ctx); err != nil {
				log.Printf("Failed to reload rules: %v", err)
			} else {
				//Ошибка: игнорирование ошибки возвращаемой функцией
				//исправить название переменной
				//count, _ := ruleStorage.GetEnabledRulesCount(ctx)
				ruleCount, err := ruleStorage.GetEnabledRulesCount(ctx)
				if err != nil {
					//TODO: логирование ошибки
				}
				log.Printf("Rules reloaded: %d enabled", ruleCount)
			}
		}
	}()

	go func() {
		//проверка статуса агентов при запуске
		if err := agentStorage.MarkOfflineAgents(ctx); err != nil {
			log.Printf("Failed to mark agent offline: %v", err)
		}
		ticker := time.NewTicker(1 * time.Minute) //все дальнейшие проверки идут по тикеру
		defer ticker.Stop()

		for range ticker.C {
			if err := agentStorage.MarkOfflineAgents(ctx); err != nil {
				log.Printf("Failed to mark agent offline: %v", err)
			} else { //здесь ошибка: второй вызов MarkOfflineAgents, он вызывается уже при при проверке условия
				//agentStorage.MarkOfflineAgents(ctx)
				log.Printf("Agent marked offline")
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
	//bad practice: прокидывать fatal
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

	//Ошибка: использование reflection
	//TODO: использовать другой способ регистрации
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
	//bad practice: прокидывать fatal
	//TODO: получить ошибку, обернуть её через %w и записать в логи
	if err := grpcServ.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
