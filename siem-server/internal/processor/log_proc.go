package processor

import (
	logstructure "siem-server/internal/logsstructure"
	parsers "siem-server/internal/parsers"
)

// обращаемся к интерфейсам, а не к конкретным структурам для того, чтобы не зависеть от конкретных реализаций
// нужных нам структур, т.е. у нас здесь ничего не сломается, если мы перейдём с Postgres на MySQL или Redis
type LogProc struct {
	parser  parsers.LogParser       //интерфейс парсера
	storage logstructure.LogStorage //интерфейс хранилища (работа с БД)
}

// создаём конструктор структуры LogProc
func NewLogProc(pars parsers.LogParser, stor logstructure.LogStorage) *LogProc {
	return &LogProc{parser: pars, storage: stor}
}

// функция описывающая процесс обработки лога (пока парсинг и сохранение)
// принимает сырой лог
// возвращает ошибку или nil
func (Lp *LogProc) ProcessRawLog(raw *logstructure.RawLog) error {
	//парсим сырой лог
	normLog, err := Lp.parser.Parser(raw) //вызов парсера через структуру LogProc в поле которой указан интерфейс parser
	if err != nil {                       //проверяем произошла ли при парсинге ошибка
		return err
	}

	return Lp.storage.Store(normLog) //сохраняем лог в БД и возвращаем результат (ошибка сохранения или nil)
}
