package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"siem-server/internal/logsstructure"
	logstructure "siem-server/internal/logsstructure"
)

type SuricataEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"event_type"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dest_ip"`
	Alert     struct {
		Signature string `json:"signature"`
		Category  string `json:"category"`
		Severity  int    `json:"severity"`
	} `json:"alert"`
}

type ParseSuricataStruct struct{}

func NewSuricataParse() *ParseSuricataStruct {
	return &ParseSuricataStruct{}
}

func (sur *ParseSuricataStruct) Parser(raw_log *logsstructure.RawLog) (*logsstructure.NormalizedLog, error) {
	var event SuricataEvent
	err := json.Unmarshal([]byte(raw_log.Raw_data), &event)
	if err != nil {
		slog.Error("Cannot unmarshal json", "error", err)
		return nil, fmt.Errorf("Cannot unmarshal json %w", err)
	}
	NormalizedLog := &logstructure.NormalizedLog{
		ID:                sur.generateID(raw_log),
		PC_name:           raw_log.PC_name,
		Username:          raw_log.Username,
		Event_description: event.Alert.Signature,
		Event_category:    event.EventType,
		Process_name:      "Network",
		Severity:          sur.defineSeverity(event.Alert.Severity),
		Timestamp:         raw_log.Event_timestamp,
		OS:                raw_log.OS,
		Source:            raw_log.Log_source,
		Raw_log:           raw_log.Raw_data,
		RawLogCached:      eventToMap(event),
	}
	//fmt.Println(NormalizedLog)
	return NormalizedLog, nil
}

func (sur *ParseSuricataStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}

func (sur *ParseSuricataStruct) defineSeverity(level int) string {
	switch level {
	case 1:
		return "Low"
	case 2:
		return "Medium"
	case 3:
		return "High"
	case 4:
		return "Critical"
	default:
		return "Info"
	}
}

// eventToMap конвертирует SuricataEvent в map для быстрого доступа по полям
func eventToMap(e SuricataEvent) map[string]interface{} {
	return map[string]interface{}{
		"timestamp":  e.Timestamp,
		"event_type": e.EventType,
		"src_ip":     e.SrcIP,
		"dest_ip":    e.DstIP,
		"alert": map[string]interface{}{
			"signature": e.Alert.Signature,
			"category":  e.Alert.Category,
			"severity":  e.Alert.Severity,
		},
	}
}
