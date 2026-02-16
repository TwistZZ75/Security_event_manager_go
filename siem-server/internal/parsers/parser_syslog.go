package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	logstructure "siem-server/internal/logsstructure"
)

type ParseSyslogStruct struct{}

func NewSyslogParse() *ParseSyslogStruct {
	return &ParseSyslogStruct{}
}

func (syslog_p *ParseSyslogStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	NormalizedLog := &logstructure.NormalizedLog{
		ID:        syslog_p.GenerateID(raw_log),
		PC_name:   raw_log.PC_name,
		Username:  raw_log.Username,
		Timestamp: raw_log.Event_timestamp,
		OS:        raw_log.OS,
		Source:    raw_log.Log_source,
		Raw_log:   raw_log.Raw_data,
	}

	syslogPattern := regexp.MustCompile(`^([^ ]+)\s+(\S+)\s+(\S+)(?:\[(\d+)\])?:\s+(.*)$`)
	matches := syslogPattern.FindStringSubmatch(raw_log.Raw_data)
	//пример лога: 2026-02-16T20:00:26.775015+03:00 isa-VirtualBox systemd[1]: systemd-hostnamed.service: Deactivated successfully.
	//timestamp := m[1]
	//hostname  := m[2]
	//app       := m[3]
	//pid       := m[4]
	//message   := m[5]

	NormalizedLog.Process_name = matches[3]
	NormalizedLog.Event_description = matches[5]
	NormalizedLog.Process_name = "syslog"
	NormalizedLog.Severity = "Info"

	return NormalizedLog, nil
}

// функция генерации ID по содержанию лога для дедупликации логов (чтобы 1 и тот же лог дважды не записывался)
// принимает сырой лог
// возвращает хеш строку на основе данных из сырого лога
func (syslog_p *ParseSyslogStruct) GenerateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}
