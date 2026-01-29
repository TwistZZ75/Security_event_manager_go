package logsstructure

//создаём интерфейс хранения логов с функцией хранения Store
//принимает указатель на входящий лог
//возвращает ошибку в случае неудачи
type LogStorage interface {
	Store(entry *NormalizedLog) error
}
