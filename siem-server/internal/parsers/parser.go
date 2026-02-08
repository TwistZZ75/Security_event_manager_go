package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

type Event struct {
	XMLName   xml.Name  `xml:"Event"`
	System    System    `xml:"System"`
	EventData EventData `xml:"EventData"` // EventData находится на уровне System, а не внутри него
}

type System struct {
	Provider    Provider `xml:"Provider"`
	EventID     int      `xml:"EventID"`
	Version     int      `xml:"Version"`
	Level       int      `xml:"Level"`
	Task        int      `xml:"Task"`
	Opcode      int      `xml:"Opcode"`
	Keywords    string   `xml:"Keywords"`
	TimeCreated struct {
		SystemTime string `xml:"SystemTime,attr"`
	} `xml:"TimeCreated"`
	EventRecordID int64  `xml:"EventRecordID"`
	Correlation   string `xml:"Correlation"`
	Execution     struct {
		ProcessID int `xml:"ProcessID,attr"`
		ThreadID  int `xml:"ThreadID,attr"`
	} `xml:"Execution"`
	Channel  string `xml:"Channel"`
	Computer string `xml:"Computer"`
	Security string `xml:"Security"`
}

type EventData struct {
	Data []DataField `xml:"Data"` // Здесь хранятся данные
}

type DataField struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

type Provider struct {
	Name string `xml:"Name,attr"`
	Guid string `xml:"Guid,attr"`
}

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
		Event_description: p.Define_EventDescription(raw_log.Raw_data),
		Event_category:    p.Define_EventCategory(raw_log.Raw_data),
		Process_name:      p.Define_ProcessName(raw_log.Raw_data),
		Severity:          p.Define_Severity(raw_log.Raw_data),
		Timestamp:         raw_log.Event_timestamp,
		OS:                raw_log.OS,
		Source:            raw_log.Log_source,
		Raw_log:           raw_log.Raw_data,
	}
	//fmt.Println(NormalizedLog)
	return NormalizedLog, nil
}

// функция генерации ID по содержанию лога для дедупликации логов (чтобы 1 и тот же лог дважды не записывался)
// принимает сырой лог
// возвращает хеш строку на основе данных из сырого лога
func (p *ParserStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}

// функция определения критичности события
// принимает строку
// возвращает строку
func (p *ParserStruct) Define_Severity(raw_log string) string {

	var event Event
	err := xml.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}

	level := event.System.Level
	switch level {
	case 0:
		return "Info"
	case 1:
		return "Critical"
	case 2:
		return "Error"
	case 3:
		return "Warning"
	case 4:
		return "Info"
	case 5:
		return "Verbose"
	default:
		return "Undefined"
	}
}

// функция определения категории события
// принимает строку
// возвращает строку
func (p *ParserStruct) Define_EventCategory(raw_log string) string {
	var event Event

	err := xml.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	// Определяем категорию на основе EventID и Channel
	switch event.System.Channel {
	case "Security":
		return p.categorizeSecurityEvent(event.System.EventID)
	case "System":
		return "System Event"
	case "Application":
		return "Application Event"
	default:
		return "Unknown"
	}
}

// Категоризация событий безопасности
// принимает eventID
// возвращает строку
func (p *ParserStruct) categorizeSecurityEvent(eventID int) string {
	// Основные категории Windows Security Events
	switch {
	case eventID >= 4624 && eventID <= 4625:
		return "Authentication"
	case eventID >= 4720 && eventID <= 4726:
		return "Account Management"
	case eventID >= 4740 && eventID <= 4767:
		return "Account Lockout"
	case eventID >= 5140 && eventID <= 5145:
		return "File Share Access"
	case eventID >= 5156 && eventID <= 5159:
		return "Network Connection"
	case eventID == 4688:
		return "Process Creation"
	case eventID == 4689:
		return "Process Termination"
	default:
		return "Security Event"
	}
}

// функция определения описания события
// принимает строку
// возвращает строку
func (p *ParserStruct) Define_EventDescription(raw_log string) string {
	var event Event
	err := xml.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	switch event.System.EventID {
	case 4624:
		return "Учётная запись успешно вошла в систему"
	case 4625:
		return "Отказ во входе в систему для учётной записи"
	case 5158:
		return "Разрешён прослушивающий сокет"
	default:
		return "Описание"
	}
}

// функция определения имени процесса
// принимает строку
// возвращает строку
func (p *ParserStruct) Define_ProcessName(raw_log string) string {
	var event Event
	err := xml.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	// Ищем Application в EventData
	for _, data := range event.EventData.Data {
		if data.Name == "Application" {
			// Извлекаем имя файла из полного пути
			parts := strings.Split(data.Value, "\\")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
			return data.Value
		}
	}

	// Если не нашли Application, возвращаем Provider Name
	return event.System.Provider.Name
}
