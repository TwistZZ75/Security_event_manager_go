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
	//загружаем переменные окружения из файла .env
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	//получаем переменную окружения DataBaseURL
	connString := os.Getenv("DB_URL")
	ctx := context.Background()
	//создаём пул соединений с БД
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close() //определяем закрытие пула соединений с БД после завершения main()

	//проверяем соединение с БД при помощи Ping()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping to database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	log.Println("Initializing components")

	//Инициализация компонентов
	//1. создаём новый экземпляр хранилища
	logStorage := postgres.NewLogStorage(pool)
	log.Println("Storage initialized")

	//2. создаём новый экземпляр парсера
	logParser := parsers.NewParser()
	log.Println("Parser initialized")

	//3. создаём новый процессор обработки лога
	logProc := processor.NewLogProc(logParser, logStorage)
	log.Println("Processor initialized")

	//4. создаём новый обработчик
	logHandler := delivery.NewLogHandler(logProc)
	log.Println("Handler initialized")

	log.Println("Starting server *_*")

	//получаем из .env переменную порта
	portStr := os.Getenv("PORT")
	//начинаем прослушивание порта
	listener, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("Failed to listen port %v", err)
	}

	//определяем максимальный размер принимаемого и отправляемого сообщения
	grpcServ := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	//регистрируем logHandler как обработчик logService
	//для того чтобы все обращения клиентов к SendRawLog() направлялись в logHandler.SendRawLog()
	pb.RegisterLogServiceServer(grpcServ, logHandler)

	reflection.Register(grpcServ) //для отладки

	log.Println("Server is ready")
	log.Println("Ctrl + C to stop")

	//функция остановки сервера
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
