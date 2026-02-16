package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	NormalizedLog := &logstructure.NormalizedLog{
		ID:                sur.GenerateID(raw_log),
		PC_name:           raw_log.PC_name,
		Username:          raw_log.Username,
		Event_description: sur.Define_EventDescription(raw_log.Raw_data),
		Event_category:    sur.Define_EventCategory(raw_log.Raw_data),
		Process_name:      "Network",
		Severity:          sur.Define_Severity(raw_log.Raw_data),
		Timestamp:         raw_log.Event_timestamp,
		OS:                raw_log.OS,
		Source:            raw_log.Log_source,
		Raw_log:           raw_log.Raw_data,
	}
	//fmt.Println(NormalizedLog)
	return NormalizedLog, nil
}

func (sur *ParseSuricataStruct) GenerateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString((hashID[:]))
}

func (sur *ParseSuricataStruct) Define_EventDescription(raw_log string) string {
	var event SuricataEvent
	err := json.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	return event.Alert.Signature
}

func (sur *ParseSuricataStruct) Define_EventCategory(raw_log string) string {
	var event SuricataEvent
	err := json.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	return event.Alert.Category
}

func (sur *ParseSuricataStruct) Define_Severity(raw_log string) string {
	var event SuricataEvent
	err := json.Unmarshal([]byte(raw_log), &event)
	if err != nil {
		return err.Error()
	}
	level := event.Alert.Severity
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
