package logsstructure

import (
	"time"
)

//Создаём структуру входящего необработанного лога

type RawLog struct {
	Username        string
	PC_name         string
	Log_source      string
	Event_timestamp time.Time
	Format          string
	Raw_data        string
}
