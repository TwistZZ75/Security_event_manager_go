package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"siem-server/proto/server/pkg/pb"
	webserver "siem-server/web_server"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {

	log.Println("Starting SIEM server...")

	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM) //создаём корневой контекст с отменой
	defer cancel()                                                                             //всегда отменяем при завершении main
	var wg sync.WaitGroup
	// Подключение к БД
	pool, err := InitDb(ctx)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer pool.Close()

	storages := InitStorages(pool)     //инициализация всех хранилищ
	services := InitServices(storages) //инициализация всех сервисов
	webServer := webserver.NewWebServer(
		storages.AgentStorage, storages.AlertStorage, storages.ActionStorage, storages.RuleStorage, storages.LogStorage,
		storages.UserStorage, services.JwtService, services.UserHandler, services.EventBus,
	)
	http_port := os.Getenv("HTTP_PORT")
	//запуск всех фоновых процессов
	wg.Add(6)
	go StartRuleEngine(ctx, cancel, services, storages, &wg)
	go StartWebServer(ctx, cancel, webServer, http_port, &wg)
	log.Println("Web interface available at http://localhost:8080")
	go DeleteExpiredTokens(ctx, pool, &wg)
	go DeleteExpiredStates(ctx, storages, &wg)
	go ReloadRules(ctx, storages, services, &wg)
	go CheckAgentStatus(ctx, storages, services, &wg)
	log.Println("Background tasks started")

	log.Println("Starting gRPC server...")
	portStr := os.Getenv("GRPC_PORT")
	listener, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("Failed to listen port: %v", err)
	}
	grpcServ := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)
	// Регистрируем сервисы
	pb.RegisterLogServiceServer(grpcServ, services.LogHandler)
	pb.RegisterAgentServiceServer(grpcServ, services.AgentHandler)

	//reflection.Register(grpcServ)

	log.Printf("Server is ready on %s", portStr)
	log.Println("Press Ctrl+C to stop")

	// Запускаем сервер
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcServ.Serve(listener); err != nil {
			log.Printf("Failed to serve: %v", err)
		}
	}()

	// Graceful Shutdown
	<-ctx.Done()
	grpcServ.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := services.EventBus.Shutdown(shutdownCtx); err != nil {
		slog.Error("Event bus shutdown error", "error", err)
	}
	webServer.Shutdown(shutdownCtx)

	log.Println("Shutting down")
	wg.Wait()
}
