package main

import (
	"log"
	logsstructure "siem-server/internal/logsstructure"
	postgres "siem-server/internal/storage/postgres"
	"time"
)

func main() {
	var repo *postgres.LogStorage

	log.Println("Starting SIEM server...")

	entry := &logsstructure.NormalizedLog{
		ID:                "sha256",
		Raw_log_id:        2,
		PC_name:           "kazuma",
		Username:          "kazuma",
		Event_description: "aboba",
		Event_category:    "pizdec",
		Process_name:      "fortinate",
		Process_id:        1,
		Severity:          "INFO",
		Timestamp:         time.Now(),
	}
	repo.Store(entry)

}
