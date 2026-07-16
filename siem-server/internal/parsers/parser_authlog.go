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

// parseParams разбирает строку параметров вида "key=value key2=value2"
func parseParams(s string) map[string]string {
	params := make(map[string]string)
	for _, pair := range strings.Fields(s) {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return params
}

func (a *ParseAuthStruct) Parser(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	entry := &logstructure.NormalizedLog{
		ID:        a.generateID(raw_log),
		PC_name:   raw_log.PC_name,
		Username:  raw_log.Username,
		Timestamp: raw_log.Event_timestamp,
		OS:        raw_log.OS,
		Source:    raw_log.Log_source,
		Raw_log:   raw_log.Raw_data,
	}

	// Формат: timestamp hostname app[pid]: message
	re := regexp.MustCompile(`^(\S+)\s+(\S+)\s+([^\[:]+)(?:\[(\d+)\])?:\s+(.*)$`)
	m := re.FindStringSubmatch(raw_log.Raw_data)
	if len(m) < 6 {
		entry.Event_category = "Authentication"
		entry.Event_description = raw_log.Raw_data
		entry.Process_name = "auth"
		entry.Severity = "Info"
		return entry, nil
	}

	app := strings.TrimSpace(m[3])
	msg := strings.TrimSpace(m[5])
	entry.Process_name = app

	// Разбираем key=value параметры после ";"
	params := make(map[string]string)
	if idx := strings.Index(msg, ";"); idx != -1 {
		params = parseParams(strings.TrimSpace(msg[idx+1:]))
	}
	if u := params["user"]; u != "" {
		entry.Username = u
	}

	entry.Event_category, entry.Severity, entry.Event_description = a.categorize(app, msg, params, entry.Username, raw_log.PC_name)
	return entry, nil
}

// =============================================================================
// categorize
//
// Возвращает (category, severity, description).
//
// Категории унифицированы с parser_xml.go и parser_syslog.go:
//
//	"Logon Success"            ← SSH accepted, PAM session opened, GDM success
//	"Logon Failure"            ← SSH failed, PAM auth failure, GDM failure
//	"Logoff"                   ← SSH session closed, PAM session closed
//	"Account Lockout"          ← pam_faillock lockout
//	"Account Unlocked"         ← pam_faillock reset
//	"Privilege Escalation"     ← sudo COMMAND=
//	"Privilege Escalation Failure" ← sudo wrong pass / not in sudoers
//	"Account Created"          ← useradd
//	"Account Deleted"          ← userdel
//	"Account Modified"         ← usermod, chage, account enabled/disabled
//	"Password Change"          ← passwd, chpasswd
//	"Group Created/Deleted/Modified" ← groupadd/del/mod
//
// Описания содержат устойчивые ключевые слова для условий contains/regex:
//
//	"Logon Failure" → "Logon failure for <user> …"
//	"Logon Success" → "Successful logon by <user> …"
//	"Logoff"        → "Logoff by <user>"
//	"Privilege Escalation" → "Privilege escalation by <user>: <command>"
//	"Account Created" → "Account created: <name>"
//
// =============================================================================
func (a *ParseAuthStruct) categorize(app, msg string, params map[string]string, username, host string) (category, severity, description string) {
	appL := strings.ToLower(app)
	msgL := strings.ToLower(msg)

	// Вспомогательная функция: извлечь имя пользователя из сообщения
	extractUser := func() string {
		if username != "" && username != "root" {
			return username
		}
		// Попробуем найти "for <user>" или "user=<user>"
		if u := params["user"]; u != "" {
			return u
		}
		return username
	}

	// ── SSH (sshd) ────────────────────────────────────────────────────────────
	if containsAny(appL, "sshd") {
		user := extractUser()
		// Попытки извлечь IP из сообщения sshd
		ip := extractIPFromMsg(msg)

		switch {
		// Logon Failure
		case containsAny(msgL,
			"failed password", "authentication failure", "invalid user",
			"no such user", "connection refused", "permission denied"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure for "+user, ip)

		// Logon Success — accepted password/publickey
		case containsAny(msgL, "accepted password", "accepted publickey", "accepted keyboard"):
			return "Logon Success", "Info",
				buildSuccessDesc("Successful logon by "+user, ip)

		// Logon Success — session opened (уже после аутентификации)
		case containsAny(msgL, "session opened"):
			return "Logon Success", "Info",
				"Successful logon by " + user + " (session opened)"

		// Logoff
		case containsAny(msgL, "session closed"):
			return "Logoff", "Info", "Logoff by " + user + " (session closed)"
		case containsAny(msgL, "disconnected", "connection closed", "connection reset"):
			return "Logoff", "Info", "Logoff by " + user + " (disconnected)"

		// Отказ в подключении
		case containsAny(msgL, "refused", "denied"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure (connection refused) for "+user, ip)

		// Явный брутфорс
		case containsAny(msgL, "too many authentication", "maximum authentication"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure (max attempts) for "+user, ip)

		case containsAny(msgL, "publickey"):
			return "Logon Success", "Info",
				"Successful logon by " + user + " (public key)"

		case containsAny(msgL, "certificate invalid", "bad protocol"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure (protocol error) for "+user, ip)
		}
		return "Authentication", "Info", msg
	}

	// ── sudo ──────────────────────────────────────────────────────────────────
	if appL == "sudo" {
		user := extractUser()

		// Извлекаем команду из COMMAND=
		cmd := ""
		if idx := strings.Index(msg, "COMMAND="); idx != -1 {
			cmd = strings.TrimSpace(msg[idx+8:])
			if sp := strings.Index(cmd, " "); sp != -1 {
				cmd = cmd[:sp]
			}
		}

		switch {
		case containsAny(msgL, "incorrect password attempt"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (incorrect password)"

		case containsAny(msgL, "not in sudoers", "not allowed to run sudo", "is not allowed"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (not in sudoers)"

		case strings.Contains(msg, "COMMAND="):
			// Успешный sudo
			desc := "Privilege escalation by " + user
			if cmd != "" {
				desc += ": " + cmd
			}
			return "Privilege Escalation", "Info", desc

		case containsAny(msgL, "conversation failed", "pam_authenticate"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (auth failure)"
		}
		return "Privilege Escalation", "Info", "Sudo: " + msg
	}

	// ── su ────────────────────────────────────────────────────────────────────
	if appL == "su" || appL == "su:" {
		user := extractUser()
		switch {
		case containsAny(msgL, "authentication failure", "failed", "incorrect"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (su failed)"
		case containsAny(msgL, "successful su", "session opened"):
			return "Privilege Escalation", "Info",
				"Privilege escalation by " + user + " (su success)"
		case containsAny(msgL, "session closed"):
			return "Logoff", "Info", "Logoff by " + user + " (su session closed)"
		}
		return "Privilege Escalation", "Info", "Switch user: " + msg
	}

	// ── PAM (pam_unix, pam_faillock, pam_tally, etc.) ─────────────────────────
	if containsAny(appL, "pam_unix", "pam_faillock", "pam_tally", "pam_pwquality",
		"pam_cracklib", "pam_securetty", "pam_systemd") {
		user := extractUser()

		switch {
		// Logon Failure
		case containsAny(msgL, "authentication failure", "auth could not identify", "check pass"):
			return "Logon Failure", "Warning",
				"Logon failure for " + user

		// Logon Success
		case containsAny(msgL, "authentication success"):
			return "Logon Success", "Info",
				"Successful logon by " + user

		// session opened — зависит от контекста (ssh/login/sudo)
		// Определяем по содержимому сообщения: "for user <name> by <service>"
		case containsAny(msgL, "session opened"):
			svc := extractServiceFromPAM(msg)
			if containsAny(svc, "sudo", "su") {
				// Это sudo/su сессия — не вход пользователя
				return "Privilege Escalation", "Info",
					"Privilege escalation by " + user + " (PAM session opened)"
			}
			return "Logon Success", "Info",
				"Successful logon by " + user + " (session opened)"

		// Logoff
		case containsAny(msgL, "session closed"):
			svc := extractServiceFromPAM(msg)
			if containsAny(svc, "sudo", "su") {
				return "Logoff", "Info", "Logoff by " + user + " (sudo/su session closed)"
			}
			return "Logoff", "Info", "Logoff by " + user + " (session closed)"

		// Account Lockout
		case containsAny(msgL, "account locked", "locked out", "maximum failure", "too many failures"):
			return "Account Lockout", "Warning",
				"Account locked: " + user

		// Account Unlocked
		case containsAny(msgL, "account unlocked", "reset", "cleared"):
			if containsAny(msgL, "faillock", "tally") {
				return "Account Unlocked", "Info",
					"Account unlocked: " + user
			}

		// Password Change
		case containsAny(msgL, "password changed", "new password", "password expir"):
			return "Password Change", "Info",
				"Password changed for " + user

		// Weak password
		case containsAny(msgL, "bad password", "password quality", "too short", "too simple", "too weak"):
			return "Logon Failure", "Warning",
				"Logon failure for " + user + " (weak/rejected password)"
		}
		return "Authentication", "Info", msg
	}

	// ── useradd / userdel / usermod ────────────────────────────────────────────
	switch {
	case containsAny(appL, "useradd"):
		newUser := extractNewUser(msg)
		return "Account Created", "Info",
			fmt.Sprintf("Account created: %s", newUser)

	case containsAny(appL, "userdel"):
		delUser := extractNewUser(msg)
		return "Account Deleted", "Info",
			fmt.Sprintf("Account deleted: %s", delUser)

	case containsAny(appL, "usermod"):
		modUser := extractNewUser(msg)
		return "Account Modified", "Info",
			fmt.Sprintf("Account modified: %s", modUser)

	case containsAny(appL, "chage"):
		return "Account Modified", "Info",
			"Account modified (password policy changed): " + extractNewUser(msg)
	}

	// ── passwd / chpasswd ──────────────────────────────────────────────────────
	if containsAny(appL, "passwd", "chpasswd") {
		user := extractUser()
		if containsAny(msgL, "fail", "error", "authentication failure") {
			return "Logon Failure", "Warning",
				"Logon failure for " + user + " (password change failed)"
		}
		return "Password Change", "Info",
			"Password changed for " + user
	}

	// ── groupadd / groupdel / groupmod ─────────────────────────────────────────
	switch {
	case containsAny(appL, "groupadd"):
		return "Group Created", "Info", "Group created: " + extractNewUser(msg)
	case containsAny(appL, "groupdel"):
		return "Group Deleted", "Info", "Group deleted: " + extractNewUser(msg)
	case containsAny(appL, "groupmod"):
		return "Group Modified", "Info", "Group modified: " + extractNewUser(msg)
	}

	// ── login / gdm / lightdm / display manager ───────────────────────────────
	if containsAny(appL, "login", "gdm", "lightdm", "kdm", "xdm", "slim", "sddm") {
		user := extractUser()
		switch {
		case containsAny(msgL, "authentication failure", "failed", "invalid"):
			return "Logon Failure", "Warning",
				"Logon failure for " + user
		case containsAny(msgL, "session opened", "logged in"):
			return "Logon Success", "Info",
				"Successful logon by " + user
		case containsAny(msgL, "session closed", "logged out"):
			return "Logoff", "Info",
				"Logoff by " + user
		}
		return "Authentication", "Info", msg
	}

	// ── Fallback: по ключевым словам в сообщении ──────────────────────────────
	user := extractUser()
	switch {
	case containsAny(msgL, "authentication failure", "auth fail", "failed password"):
		return "Logon Failure", "Warning", "Logon failure for " + user
	case containsAny(msgL, "session opened"):
		return "Logon Success", "Info", "Successful logon by " + user
	case containsAny(msgL, "session closed"):
		return "Logoff", "Info", "Logoff by " + user
	case containsAny(msgL, "locked", "account disabled", "account locked", "account expir"):
		return "Account Lockout", "Warning", "Account locked: " + user
	}

	return "Authentication", "Info", msg
}

func (a *ParseAuthStruct) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// extractNewUser пытается достать имя нового пользователя/группы из сообщения useradd/userdel
func extractNewUser(msg string) string {
	// useradd: "new user: name=bob, UID=1001, ..."
	if idx := strings.Index(msg, "name="); idx != -1 {
		rest := msg[idx+5:]
		if end := strings.IndexAny(rest, ", "); end != -1 {
			return rest[:end]
		}
		return rest
	}
	// userdel: "delete user 'bob'"
	re := regexp.MustCompile(`'([^']+)'`)
	if m := re.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}
