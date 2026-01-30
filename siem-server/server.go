package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"siem-server/internal/delivery"
	"siem-server/internal/parsers"
	processor "siem-server/internal/processor"
	postgres "siem-server/internal/storage/postgres"
	"siem-server/proto/server/pkg/pb"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	log.Println("Starting SIEM server...")
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	connString := os.Getenv("DB_URL")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping to database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	log.Println("Initializing components")

	logStorage := postgres.NewLogStorage(pool)
	log.Println("Storage initialized")

	logParser := parsers.NewParser()
	log.Println("Parser initialized")

	logProc := processor.NewLogProc(logParser, logStorage)
	log.Println("Processor initialized")

	logHandler := delivery.NewLogHandler(logProc)
	log.Println("Handler initialized")

	log.Println("Starting server *_*")

	portStr := os.Getenv("PORT")
	listener, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("Failed to listen port %v", err)
	}

	grpcServ := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	pb.RegisterLogServiceServer(grpcServ, logHandler)

	reflection.Register(grpcServ)

	log.Println("Server is ready")
	log.Println("Ctrl + C to stop")

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		grpcServ.GracefulStop()
		log.Println("Server stopped")
	}()

	if err := grpcServ.Serve(listener); err != nil {
		log.Fatalf("Failed to serve %v", err)
	}

}
