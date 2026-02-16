package parsers

import (
	"fmt"
	logstructure "siem-server/internal/logsstructure"
)

// создаём интерфейс парсера логов с функцией Parser
// принимает "сырой" лог
// возвращает нормализованный лог или ошибку
type LogParser interface {
	Parser(raw *logstructure.RawLog) (*logstructure.NormalizedLog, error)
	GenerateID(raw_log *logstructure.RawLog) string
}

type Parser struct {
	parsers map[string]LogParser
}

func NewParser() *Parser {
	mp := &Parser{
		parsers: make(map[string]LogParser),
	}

	// Регистрируем все доступные парсеры
	mp.parsers["xml"] = NewXmlParse()
	mp.parsers["json"] = NewSuricataParse()
	mp.parsers["squid"] = NewSquidParse()
	mp.parsers["syslog"] = NewSyslogParse()

	return mp
}

// Parser выбирает подходящий парсер на основе формата лога
func (mp *Parser) Parser(raw *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	parser, exists := mp.parsers[raw.Format]
	if !exists {
		return nil, fmt.Errorf("unsupported log format: %s", raw.Format)
	}

	return parser.Parser(raw)
}

// GenerateID делегирует генерацию ID соответствующему парсеру
func (mp *Parser) GenerateID(raw_log *logstructure.RawLog) string {
	parser, exists := mp.parsers[raw_log.Format]
	if !exists {
		return ""
	}

	return parser.GenerateID(raw_log)
}
