package main

import (
	"context"
	"log"
	"os"
	logsstructure "siem-server/internal/logsstructure"
	postgres "siem-server/internal/storage/postgres"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
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

	storage := postgres.NewLogStorage(pool)

	entry := &logsstructure.NormalizedLog{
		ID:                "sha256",
		Raw_log_id:        1,
		PC_name:           "kazuma",
		Username:          "kazuma",
		Event_description: "aboba",
		Event_category:    "pizdec",
		Process_name:      "fortinate",
		Process_id:        1,
		Severity:          "INFO",
		Timestamp:         time.Now(),
	}

	if err := storage.Store(entry); err != nil {
		log.Fatalf("Failed to store log: %v", err)
	}

	log.Println("Success store")
}
