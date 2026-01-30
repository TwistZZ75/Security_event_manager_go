package parsers

import logstructure "siem-server/internal/logsstructure"

//создаём интерфейс парсера логов с функцией Parser
//принимает "сырой" лог
//возвращает нормализованный лог или ошибку
type LogParser interface {
	Parser(raw *logstructure.RawLog) (*logstructure.NormalizedLog, error)
}
