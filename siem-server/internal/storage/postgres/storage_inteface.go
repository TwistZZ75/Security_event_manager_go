package postgres

import "siem-server/internal/logsstructure"

//создаём интерфейс хранения логов с функцией хранения Store
//принимает указатель на входящий лог
//возвращает ошибку в случае неудачи
type LogStorageInterface interface {
	Store(entry *logsstructure.NormalizedLog) error
}
