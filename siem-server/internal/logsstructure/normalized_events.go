package logsstructure

import (
	"time"
)

//Создаём структуру обработанного(нормализованного) лога

type NormalizedLog struct {
	ID                string    //идентификатор лога
	Raw_log_id        int       //идентификатор "сырого" лога, если администратору необходимо будет просмотреть изначальный лог
	PC_name           string    //имя компьютера с которого пришёл лог
	Username          string    //имя пользователя у которого лог был сгенерирован
	Event_description string    //описание события
	Event_category    string    //категория события "событие аутентификации", "событие файловой системы" и т.п.
	Process_name      string    //имя процесса создавшего событие
	Severity          string    //важность события "INFO", "WARNING", "DANGER"
	Timestamp         time.Time //время создания события
	OS                string    //ОС
	Source            string    //источник лога
}
