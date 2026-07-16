package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

type ParseSquidStruct struct{}

func NewSquidParse() *ParseSquidStruct {
	return &ParseSquidStruct{}
}

func (squid_p *ParseSquidStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	NormalizedLog := &logstructure.NormalizedLog{
		ID:        squid_p.generateID(raw_log),
		PC_name:   raw_log.PC_name,
		Username:  raw_log.Username,
		Timestamp: raw_log.Event_timestamp,
		OS:        raw_log.OS,
		Source:    raw_log.Log_source,
		Raw_log:   raw_log.Raw_data,
	}

	// Формат: timestamp duration client_ip code/status bytes method URL user
	parts := strings.Fields(raw_log.Raw_data)

	NormalizedLog.Event_category = "Squid event"

	if len(parts) >= 7 {
		method := parts[5]
		url := parts[6]
		NormalizedLog.Event_description = "HTTP " + method + " request to " + url

		// Проверяем статус код для определения серьезности
		if len(parts) >= 4 {
			statusParts := strings.Split(parts[3], "/")
			if len(statusParts) > 0 {
				statusCode := statusParts[0]
				if strings.HasPrefix(statusCode, "4") || strings.HasPrefix(statusCode, "5") {
					NormalizedLog.Severity = "Warning"
				} else {
					NormalizedLog.Severity = "Info"
				}
			}
		}
	} else {
		NormalizedLog.Event_description = "Squid proxy event"
		NormalizedLog.Severity = "Info"
	}

	NormalizedLog.Process_name = "squid"

	//fmt.Println(NormalizedLog)
	return NormalizedLog, nil
}

// функция генерации ID по содержанию лога для дедупликации логов (чтобы 1 и тот же лог дважды не записывался)
// принимает сырой лог
// возвращает хеш строку на основе данных из сырого лога
func (squid_p *ParseSquidStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}
