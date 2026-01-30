package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

// структура парсера
type ParserStruct struct {
	//В дальнейшем будет расширяться, как минимум за счёт поля в котором будет указан формат лога
}

// создаём конструктор для парсера
func NewParser() *ParserStruct {
	return &ParserStruct{} //возвращаем адрес структуры, в {} указываются поля структуры
	//при создании элемента структуры будет указываться {поле структуры: значение}
}

// функция парсинга исходного лога в нормализованный лог
// принимает элемент структуры RawLog
// возвращает нормализованный лог
func (p *ParserStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	NormalizedLog := &logstructure.NormalizedLog{
		ID:                p.generateID(raw_log),
		PC_name:           raw_log.PC_name,
		Username:          raw_log.Username,
		Event_description: raw_log.Raw_data,
		Event_category:    raw_log.Raw_data,
		Process_name:      raw_log.Raw_data,
		Severity:          p.Define_Severity(raw_log),
		Timestamp:         raw_log.Event_timestamp,
		OS:                raw_log.OS,
		Source:            raw_log.Log_source,
	}

	return NormalizedLog, nil
}

// функция генерации ID по содержанию лога для дедупликации логов (чтобы 1 и тот же лог дважды не записывался)
// принимает сырой лог
// возвращает хеш строку на основе данных из сырого лога
func (p *ParserStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Event_timestamp.String() + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}

// функция определения критичности события
// принимает экземпляр структуры RawLog
// возвращает строку
func (p *ParserStruct) Define_Severity(raw_log *logstructure.RawLog) string {

	severity := "INFO"
	lower_RawData := strings.ToLower(raw_log.Raw_data)
	if strings.Contains(lower_RawData, "error") {
		severity = "ERROR"
	} else if strings.Contains(lower_RawData, "warn") {
		severity = "WARNING"
	}

	return severity
}

// функция определения категории события
// принимает экземпляр структуры RawLog
// возвращает строку
func (p *ParserStruct) Define_EventCategory(raw_log *logstructure.RawLog) string {
	event_category := "qwe"
	return event_category
}

// функция определения описания события
// принимает экземпляр структуры RawLog
// возвращает строку
func (p *ParserStruct) Define_EventDescription(raw_log *logstructure.RawLog) string {
	event_description := "qwe"
	return event_description
}

// функция определения имени процесса
// принимает экземпляр структуры RawLog
// возвращает строку
func (p *ParserStruct) Define_ProcessName(raw_log *logstructure.RawLog) string {
	process_name := "qwe"
	return process_name
}
