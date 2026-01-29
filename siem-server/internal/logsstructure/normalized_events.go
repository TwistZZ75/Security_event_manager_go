package logsstructure

import (
	"time"
)

//Создаём структуру обработанного(нормализованного) лога

type NormalizedLog struct {
	ID                string
	Raw_log_id        int
	PC_name           string
	Username          string
	Event_description string
	Event_category    string
	Process_name      string
	Process_id        int
	Severity          string
	Timestamp         time.Time
}
