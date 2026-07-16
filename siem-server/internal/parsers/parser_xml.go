package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

type Event struct {
	XMLName   xml.Name  `xml:"Event"`
	System    System    `xml:"System"`
	EventData EventData `xml:"EventData"`
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
	Data []DataField `xml:"Data"`
}

type DataField struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}

type Provider struct {
	Name string `xml:"Name,attr"`
	Guid string `xml:"Guid,attr"`
}

type ParseXmlStruct struct{}

func NewXmlParse() *ParseXmlStruct {
	return &ParseXmlStruct{}
}

func (xml_p *ParseXmlStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	NormalizedLog := &logstructure.NormalizedLog{
		ID:                xml_p.generateID(raw_log),
		PC_name:           raw_log.PC_name,
		Username:          raw_log.Username,
		Event_description: xml_p.Define_EventDescription(raw_log.Raw_data),
		Event_category:    xml_p.Define_EventCategory(raw_log.Raw_data),
		Process_name:      xml_p.Define_ProcessName(raw_log.Raw_data),
		Severity:          xml_p.Define_Severity(raw_log.Raw_data),
		Timestamp:         raw_log.Event_timestamp,
		OS:                raw_log.OS,
		Source:            raw_log.Log_source,
		Raw_log:           raw_log.Raw_data,
	}
	return NormalizedLog, nil
}

// sanitizeXML удаляет недопустимые в XML управляющие символы
func (xml_p *ParseXmlStruct) sanitizeXML(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0x9 || r == 0xA || r == 0xD {
			b.WriteRune(r)
			continue
		}
		if r < 0x20 {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (xml_p *ParseXmlStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	hashID := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hashID[:])
}

func (xml_p *ParseXmlStruct) Define_Severity(raw_log string) string {
	clean_raw_log := xml_p.sanitizeXML(raw_log)
	var event Event
	if err := xml.Unmarshal([]byte(clean_raw_log), &event); err != nil {
		return "Undefined"
	}
	switch event.System.Level {
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

func (xml_p *ParseXmlStruct) Define_EventCategory(raw_log string) string {
	clean := xml_p.sanitizeXML(raw_log)
	var event Event
	if err := xml.Unmarshal([]byte(clean), &event); err != nil {
		return "Parse Error"
	}
	switch event.System.Channel {
	case "Security":
		return xml_p.categorizeSecurityEvent(event.System.EventID)
	case "System":
		return xml_p.categorizeSystemEvent(event.System.EventID)
	case "Application":
		return xml_p.categorizeApplicationEvent(event)
	case "Microsoft-Windows-PowerShell/Operational":
		return "PowerShell"
	case "Microsoft-Windows-Sysmon/Operational":
		return xml_p.categorizeSysmonEvent(event.System.EventID)
	case "Microsoft-Windows-TaskScheduler/Operational":
		return "Scheduled Task"
	case "Microsoft-Windows-Windows Firewall With Advanced Security/Firewall":
		return "Firewall"
	default:
		return "Other"
	}
}

// =============================================================================
// categorizeSecurityEvent
// =============================================================================
func (xml_p *ParseXmlStruct) categorizeSecurityEvent(eventID int) string {
	switch {

	// ── Вход / выход ──────────────────────────────────────────────────────────
	// Unified: "Logon Success" / "Logon Failure" / "Logoff"
	case eventID == 4624:
		return "Logon Success"
	case eventID == 4625:
		return "Logon Failure"
	case eventID == 4634, eventID == 4647:
		return "Logoff"
	case eventID == 4648:
		// Явные учётные данные — тоже вход, но с флагом explicit
		return "Logon Success"
	case eventID == 4800:
		return "Workstation Locked"
	case eventID == 4801:
		return "Workstation Unlocked"

	// ── Привилегии ────────────────────────────────────────────────────────────
	// Unified: "Privilege Escalation" (аналог sudo COMMAND= на Linux)
	case eventID == 4672:
		return "Privilege Escalation"
	case eventID == 4673, eventID == 4674:
		return "Privilege Use"

	// ── Управление аккаунтами ─────────────────────────────────────────────────
	// Unified: разбиваем диапазон 4720-4726 на конкретные категории,
	// чтобы совпадать с Linux: "Account Created", "Account Deleted", etc.
	case eventID == 4720, eventID == 4721:
		return "Account Created"
	case eventID == 4722, eventID == 4725, eventID == 4738:
		// 4722 = enabled, 4725 = disabled, 4738 = changed → все "Account Modified"
		return "Account Modified"
	case eventID == 4723, eventID == 4724:
		// 4723 = попытка сменить пароль, 4724 = сброс пароля администратором
		return "Password Change"
	case eventID == 4726:
		return "Account Deleted"

	// ── Управление группами ───────────────────────────────────────────────────
	// Unified: "Group Created" / "Group Deleted" / "Group Modified"
	case eventID == 4727, eventID == 4731, eventID == 4754, eventID == 4759:
		return "Group Created"
	case eventID == 4730, eventID == 4734, eventID == 4758, eventID == 4763:
		return "Group Deleted"
	case eventID == 4728, eventID == 4729, eventID == 4732, eventID == 4733,
		eventID == 4756, eventID == 4757:
		return "Group Modified"

	// ── Блокировка / разблокировка аккаунта ──────────────────────────────────
	// Unified: "Account Lockout" / "Account Unlocked"
	case eventID >= 4740 && eventID <= 4743:
		return "Account Lockout"
	case eventID == 4767:
		return "Account Unlocked"

	// ── Политики ─────────────────────────────────────────────────────────────
	case eventID >= 4713 && eventID <= 4719:
		return "Policy Change"
	case eventID == 4739:
		return "Domain Policy Change"
	case eventID >= 4902 && eventID <= 4908:
		return "Audit Policy Change"

	// ── Процессы ─────────────────────────────────────────────────────────────
	// Unified: "Process Creation" / "Process Terminated" (совпадают с Sysmon)
	case eventID == 4688:
		return "Process Creation"
	case eventID == 4689:
		return "Process Terminated"

	// ── Сеть ─────────────────────────────────────────────────────────────────
	// Unified: "Network Connection" (совпадает с Sysmon 3)
	case eventID >= 5156 && eventID <= 5159:
		return "Network Connection"
	case eventID == 5140, eventID == 5145:
		return "File Share Access"
	case eventID == 5142, eventID == 5143, eventID == 5144:
		return "File Share Modified"

	// ── Объекты ───────────────────────────────────────────────────────────────
	case eventID >= 4656 && eventID <= 4663:
		return "Object Access"
	case eventID == 4670:
		return "Object Permission Change"

	// ── Криптография ─────────────────────────────────────────────────────────
	case eventID >= 5058 && eventID <= 5061:
		return "Cryptographic Operation"

	// ── Kerberos / NTLM ───────────────────────────────────────────────────────
	case eventID == 4768, eventID == 4769, eventID == 4770:
		return "Kerberos Authentication"
	case eventID == 4771:
		// Kerberos pre-auth failed = фактически Logon Failure
		return "Logon Failure"
	case eventID == 4776:
		return "NTLM Authentication"

	// ── Системные в Security-канале ───────────────────────────────────────────
	// Unified: "System Startup" / "System Shutdown" (совпадают с systemd на Linux)
	case eventID == 4608:
		return "System Startup"
	case eventID == 4609:
		return "System Shutdown"
	case eventID == 1102:
		return "Audit Log Cleared"
	case eventID == 4616:
		return "System Time Changed"

	default:
		return "Security Event"
	}
}

// categorizeSystemEvent — System Log (канал "System")
// Unified: "Service Installed"/"Service Started"/"Service Stopped" совпадают с systemd на Linux
func (xml_p *ParseXmlStruct) categorizeSystemEvent(eventID int) string {
	switch {
	case eventID == 7045:
		return "Service Installed"
	case eventID == 7036:
		// Уточнение происходит в описании (running/stopped)
		return "Service State Change"
	case eventID == 7040:
		return "Service Modified"
	// Unified: "System Startup" / "System Shutdown"
	case eventID == 6005:
		return "System Startup"
	case eventID == 6006:
		return "System Shutdown"
	case eventID == 6008:
		return "Unexpected Shutdown"
	case eventID == 6013:
		return "System Uptime"
	case eventID == 104:
		return "Audit Log Cleared"
	case eventID >= 1 && eventID <= 100:
		return "Kernel Event"
	default:
		return "System Event"
	}
}

// categorizeSysmonEvent — Sysmon Operational Log
// Unified: "Process Creation" / "Process Terminated" / "Network Connection" / "Registry Event"
func (xml_p *ParseXmlStruct) categorizeSysmonEvent(eventID int) string {
	switch eventID {
	case 1:
		return "Process Creation" // unified с Security 4688
	case 2:
		return "File Creation Time Changed"
	case 3:
		return "Network Connection" // unified с Security 5156
	case 5:
		return "Process Terminated" // unified с Security 4689
	case 6:
		return "Driver Loaded"
	case 7:
		return "Image Loaded"
	case 8:
		return "Remote Thread Created"
	case 10:
		return "Process Access"
	case 11:
		return "File Created"
	case 12, 13, 14:
		return "Registry Event"
	case 15:
		return "File Stream Created"
	case 17, 18:
		return "Pipe Event"
	case 22:
		return "DNS Query"
	case 25:
		return "Process Tampering"
	default:
		return "Sysmon Event"
	}
}

func (xml_p *ParseXmlStruct) categorizeApplicationEvent(event Event) string {
	provider := strings.ToLower(event.System.Provider.Name)
	switch {
	case strings.Contains(provider, "msiinstaller"):
		return "Software Installation"
	case strings.Contains(provider, "security-spp"):
		return "License Activation"
	case strings.Contains(provider, "wininit"):
		return "System Startup"
	default:
		return "Application Event"
	}
}

// =============================================================================
// Define_EventDescription
// =============================================================================
func (xml_p *ParseXmlStruct) Define_EventDescription(raw_log string) string {
	clean := xml_p.sanitizeXML(raw_log)
	var event Event
	if err := xml.Unmarshal([]byte(clean), &event); err != nil {
		return "Failed to parse event"
	}

	data := make(map[string]string)
	for _, d := range event.EventData.Data {
		data[d.Name] = d.Value
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v := data[k]; v != "" {
				return v
			}
		}
		return ""
	}

	switch event.System.EventID {

	// ── Вход / выход ──────────────────────────────────────────────────────────
	// Ключевые слова: "successful logon", "logon failure", "logoff"
	// Совпадают с Linux-описаниями: "Successful logon by ..." / "Logon failure for ..."

	case 4624: // Logon Success
		user := get("TargetUserName", "SubjectUserName")
		logonType := get("LogonType")
		ip := get("IpAddress")
		desc := "Successful logon"
		if user != "" {
			desc += " by " + user
		}
		if logonType != "" {
			desc += " (type " + logonType + ")"
		}
		if ip != "" && ip != "-" && ip != "127.0.0.1" && ip != "::1" {
			desc += " from " + ip
		}
		return desc

	case 4625: // Logon Failure
		user := get("TargetUserName")
		ip := get("IpAddress")
		reason := get("FailureReason", "SubStatus")
		desc := "Logon failure"
		if user != "" {
			desc += " for " + user
		}
		if ip != "" && ip != "-" && ip != "127.0.0.1" && ip != "::1" {
			desc += " from " + ip
		}
		if reason != "" {
			desc += " (" + reason + ")"
		}
		return desc

	case 4634, 4647: // Logoff
		user := get("TargetUserName")
		desc := "Logoff"
		if user != "" {
			desc += " by " + user
		}
		return desc

	case 4648: // Explicit Credential Logon → тоже Logon Success с примечанием
		user := get("SubjectUserName")
		target := get("TargetServerName")
		return fmt.Sprintf("Successful logon by %s using explicit credentials to %s", user, target)

	case 4800: // Workstation Locked
		user := get("TargetUserName")
		return "Workstation locked by " + user

	case 4801: // Workstation Unlocked
		user := get("TargetUserName")
		return "Workstation unlocked by " + user

	// ── Привилегии ────────────────────────────────────────────────────────────
	// Unified ключевые слова: "privilege escalation by", "privilege use"
	case 4672: // Privilege Escalation
		user := get("SubjectUserName")
		return "Privilege escalation by " + user

	case 4673, 4674:
		user := get("SubjectUserName")
		priv := get("PrivilegeList")
		desc := "Privilege use by " + user
		if priv != "" {
			desc += ": " + priv
		}
		return desc

	// ── Управление аккаунтами ─────────────────────────────────────────────────
	// Unified ключевые слова: "account created", "account deleted", "account modified",
	// "password changed", "account locked", "account unlocked"

	case 4720: // Account Created
		target := get("TargetUserName")
		subj := get("SubjectUserName")
		return fmt.Sprintf("Account created: %s by %s", target, subj)

	case 4722: // Account Enabled → Account Modified
		target := get("TargetUserName")
		return "Account modified (enabled): " + target

	case 4723: // Password Change attempt
		target := get("TargetUserName")
		return "Password changed for " + target

	case 4724: // Password Reset by admin → Password Change
		target := get("TargetUserName")
		subj := get("SubjectUserName")
		return fmt.Sprintf("Password changed (reset) for %s by %s", target, subj)

	case 4725: // Account Disabled → Account Modified
		target := get("TargetUserName")
		return "Account modified (disabled): " + target

	case 4726: // Account Deleted
		target := get("TargetUserName")
		subj := get("SubjectUserName")
		return fmt.Sprintf("Account deleted: %s by %s", target, subj)

	case 4738: // Account Modified
		target := get("TargetUserName")
		return "Account modified: " + target

	case 4740: // Account Lockout
		target := get("TargetUserName")
		caller := get("CallerComputerName")
		desc := "Account locked: " + target
		if caller != "" {
			desc += " from " + caller
		}
		return desc

	case 4767: // Account Unlocked
		target := get("TargetUserName")
		subj := get("SubjectUserName")
		return fmt.Sprintf("Account unlocked: %s by %s", target, subj)

	// ── Управление группами ───────────────────────────────────────────────────
	case 4727, 4731, 4754, 4759:
		return "Group created: " + get("TargetUserName", "TargetDomainName")
	case 4730, 4734, 4758, 4763:
		return "Group deleted: " + get("TargetUserName", "TargetDomainName")
	case 4728, 4729, 4732, 4733, 4756, 4757:
		member := get("MemberName", "MemberSid")
		group := get("TargetUserName")
		return fmt.Sprintf("Group modified: %s (member: %s)", group, member)

	// ── Процессы ─────────────────────────────────────────────────────────────
	// Unified: "Process created: <name>", "Process terminated: <name>"
	case 4688: // Process Creation
		proc := get("NewProcessName")
		parent := get("ParentProcessName")
		parts := strings.Split(proc, "\\")
		name := parts[len(parts)-1]
		if parent != "" {
			parentParts := strings.Split(parent, "\\")
			return fmt.Sprintf("Process created: %s (parent: %s)", name, parentParts[len(parentParts)-1])
		}
		return "Process created: " + name

	case 4689: // Process Terminated
		proc := get("ProcessName")
		parts := strings.Split(proc, "\\")
		return "Process terminated: " + parts[len(parts)-1]

	// ── Сеть ─────────────────────────────────────────────────────────────────
	case 5156:
		app := get("Application")
		dst := get("DestAddress") + ":" + get("DestPort")
		return "Network connection allowed: " + app + " → " + dst
	case 5157:
		app := get("Application")
		return "Network connection blocked: " + app
	case 5158:
		return "Listening socket allowed: " + get("Application")
	case 5140:
		return "File share accessed: " + get("ShareName") + " by " + get("SubjectUserName")
	case 5145:
		return "File share access checked: " + get("RelativeTargetName")

	// ── Службы ───────────────────────────────────────────────────────────────
	// Unified: "Service installed: <name>", "Service started: <name>", "Service stopped: <name>"
	case 7045:
		return "Service installed: " + get("ServiceName")
	case 7036:
		svc := get("param1")
		state := get("param2")
		if strings.Contains(strings.ToLower(state), "running") || strings.Contains(strings.ToLower(state), "started") {
			return "Service started: " + svc
		}
		return "Service stopped: " + svc

	// ── Системные ─────────────────────────────────────────────────────────────
	// Unified: "System startup", "System shutdown"
	case 6005:
		return "System startup (event log service started)"
	case 6006:
		return "System shutdown (event log service stopped)"
	case 6008:
		return "Unexpected system shutdown detected"
	case 1102:
		return "Audit log cleared by " + get("SubjectUserName")
	case 104:
		return "System log cleared"
	case 4608:
		return "System startup (Windows starting up)"
	case 4609:
		return "System shutdown (Windows shutting down)"
	case 4616:
		return "System time changed by " + get("SubjectUserName")

	// ── Kerberos / NTLM ───────────────────────────────────────────────────────
	case 4768:
		user := get("TargetUserName")
		ip := get("IpAddress")
		return fmt.Sprintf("Kerberos TGT requested by %s from %s", user, ip)
	case 4769:
		user := get("TargetUserName")
		svc := get("ServiceName")
		return fmt.Sprintf("Kerberos service ticket requested by %s for %s", user, svc)
	case 4771: // Kerberos pre-auth failed = Logon Failure
		user := get("TargetUserName")
		ip := get("IpAddress")
		return fmt.Sprintf("Logon failure (Kerberos pre-auth) for %s from %s", user, ip)
	case 4776:
		user := get("TargetUserName")
		ws := get("Workstation")
		return fmt.Sprintf("NTLM authentication for %s from %s", user, ws)

	default:
		if desc := get("Description", "Message", "param1"); desc != "" {
			if len(desc) > 120 {
				return desc[:120] + "…"
			}
			return desc
		}
		return fmt.Sprintf("Event ID %d on channel %s", event.System.EventID, event.System.Channel)
	}
}

func (xml_p *ParseXmlStruct) Define_ProcessName(raw_log string) string {
	clean_raw_log := xml_p.sanitizeXML(raw_log)
	var event Event
	if err := xml.Unmarshal([]byte(clean_raw_log), &event); err != nil {
		return ""
	}
	for _, d := range event.EventData.Data {
		if d.Name == "Application" || d.Name == "NewProcessName" || d.Name == "ProcessName" {
			parts := strings.Split(d.Value, "\\")
			if len(parts) > 0 && parts[len(parts)-1] != "" {
				return parts[len(parts)-1]
			}
			return d.Value
		}
	}
	return event.System.Provider.Name
}
