package logsstructure

import (
	"time"
)

//Создаём структуру обработанного(нормализованного) лога

type NormalizedLog struct {
	ID                string                 //идентификатор лога
	PC_name           string                 //имя компьютера с которого пришёл лог
	Username          string                 //имя пользователя у которого лог был сгенерирован
	Event_description string                 //описание события
	Event_category    string                 //категория события "событие аутентификации", "событие файловой системы" и т.п.
	Process_name      string                 //имя процесса создавшего событие
	Severity          string                 //важность события "INFO", "WARNING", "DANGER"
	Timestamp         time.Time              //время создания события
	OS                string                 //ОС
	Source            string                 //источник лога
	Raw_log           string                 //исходный лог
	RawLogCached      map[string]interface{} //кеш распаршенного JSON
}

// LogFilter описывает параметры фильтрации и пагинации логов.
type LogFilter struct {
	PCName   string    // фильтр по имени компьютера
	Username string    // фильтр по пользователю
	Severity string    // фильтр по важности (INFO, WARNING, DANGER)
	Category string    // фильтр по категории события
	Source   string    // фильтр по источнику лога
	OS       string    // фильтр по ОС
	From     time.Time // начало временного диапазона
	To       time.Time // конец временного диапазона
	Limit    int       // кол-во записей на страницу (0 = без ограничений)
	Offset   int       // смещение для пагинации
}
