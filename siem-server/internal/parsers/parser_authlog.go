package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

type ParseAuthStruct struct{}

func NewAuthParse() *ParseAuthStruct {
	return &ParseAuthStruct{}
}

func (auth_p *ParseAuthStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	NormalizedLog := &logstructure.NormalizedLog{
		ID:        auth_p.GenerateID(raw_log),
		PC_name:   raw_log.PC_name,
		Username:  raw_log.Username,
		Timestamp: raw_log.Event_timestamp,
		OS:        raw_log.OS,
		Source:    raw_log.Log_source,
		Raw_log:   raw_log.Raw_data,
	}

	re := regexp.MustCompile(`^(\S+\+\S+)\s+(\S+)\s+([^:]+):\s+(.*)$`)
	matches := re.FindStringSubmatch(raw_log.Raw_data)
	if len(matches) != 5 {
		return nil, fmt.Errorf("не удалось распарсить строку syslog: %s", raw_log.Raw_data)
	}
	//пример лога: 2026-03-07T07:52:27.536793+03:00 isa-VirtualBox gdm-password]: pam_unix(gdm-password:auth): authentication failure; logname= uid=0 euid=0 tty=/dev/tty1 ruser= rhost=  user=isa
	//timestamp := m[1]
	//hostname  := m[2]
	//app       := m[3]
	//pid       := m[4]
	//message   := m[5]
	NormalizedLog.Process_name = strings.TrimSuffix(matches[3], "]")
	NormalizedLog.Event_description = matches[4]
	message := matches[4]
	var params map[string]string

	if idx := strings.Index(message, ";"); idx != -1 {

		paramStr := strings.TrimSpace(message[idx+1:])
		params = parseParams(paramStr)
	} else {

		params = map[string]string{}
	}
	NormalizedLog.Username = params["user"]
	NormalizedLog.Event_category = "Authentication"
	//NormalizedLog.Process_name = "syslog"
	NormalizedLog.Severity = "Info"

	return NormalizedLog, nil
}

// функция генерации ID по содержанию лога для дедупликации логов (чтобы 1 и тот же лог дважды не записывался)
// принимает сырой лог
// возвращает хеш строку на основе данных из сырого лога
func (auth_p *ParseAuthStruct) GenerateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}

// parseParams разбирает строку параметров вида "key=value key2=value2"
func parseParams(s string) map[string]string {
	params := make(map[string]string)
	// Разделяем по пробелам, но значения могут содержать пробелы?
	// В данном случае значения простые
	pairs := strings.Fields(s)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			params[key] = value
		}
	}
	return params
}
