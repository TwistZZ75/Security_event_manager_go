package logsstructure

import (
	"time"
)

//Создаём структуру входящего необработанного лога

type RawLog struct {
	Username        string    // имя пользователя
	PC_name         string    // имя компьютера
	OS              string    // ОС
	Log_source      string    // источник лога (suricata, syslog, winEvents...)
	Event_timestamp time.Time // время создания лога
	Format          string    // формат лога (xml, json и т.д.)
	Raw_data        string    // сам лог целиком
}
