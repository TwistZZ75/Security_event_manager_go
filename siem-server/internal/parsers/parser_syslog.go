package parsers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	logstructure "siem-server/internal/logsstructure"
	"strings"
)

var (
	reSyslogMain   = regexp.MustCompile(`^(\S+)\s+(\S+)\s+([^\[:]+)(?:\[(\d+)\])?:\s+(.*)$`)
	reUserSSH      = regexp.MustCompile(`(?:for|user)\s+(\S+)\s+(?:from|port)`)
	reInvalidUser  = regexp.MustCompile(`(?i)invalid user\s+(\S+)`)
	reSudoUser     = regexp.MustCompile(`^(\S+)\s+:`)
	rePAMUser      = regexp.MustCompile(`for user\s+(\S+)`)
	reSystemdUnit1 = regexp.MustCompile(`(?:Started|Stopped|Failed|Reloaded)\s+(.+?)\.?$`)
	reSystemdUnit2 = regexp.MustCompile(`^([\w@.\\-]+\.service)`)
	reLogindUser   = regexp.MustCompile(`of user\s+(\S+)`)
	reOOMProcess   = regexp.MustCompile(`Kill(?:ed)? process \d+ \(([^)]+)\)`)
)

type SyslogParser struct{}

func NewSyslogParse() *SyslogParser {
	return &SyslogParser{}
}

func (s *SyslogParser) Parse(raw_log *logstructure.RawLog) (*logstructure.NormalizedLog, error) {
	entry := &logstructure.NormalizedLog{
		ID:        s.generateID(raw_log),
		PC_name:   raw_log.PC_name,
		Username:  raw_log.Username,
		Timestamp: raw_log.Event_timestamp,
		OS:        raw_log.OS,
		Source:    raw_log.Log_source,
		Raw_log:   raw_log.Raw_data,
	}

	// Формат: timestamp hostname app[pid]: message
	m := reSyslogMain.FindStringSubmatch(raw_log.Raw_data)
	if len(m) < 6 {
		entry.Event_category = "System Event"
		entry.Event_description = raw_log.Raw_data
		entry.Process_name = "syslog"
		entry.Severity = "Info"
		return entry, nil
	}

	app := strings.TrimSpace(m[3])
	msg := strings.TrimSpace(m[5])

	entry.Process_name = app
	entry.Event_category, entry.Severity, entry.Event_description = s.categorize(app, msg)
	return entry, nil
}

// =============================================================================
// categorize
//
// Возвращает (category, severity, description).
//
// Категории для аутентификационных событий унифицированы с parser_xml.go
// и parser_authlog.go — одни и те же строки для одних и тех же действий:
//
//	"Logon Success"               ← sshd accepted, PAM session (login context)
//	"Logon Failure"               ← sshd failed, PAM auth failure
//	"Logoff"                      ← sshd session closed, PAM session closed
//	"Privilege Escalation"        ← sudo COMMAND=
//	"Privilege Escalation Failure"← sudo wrong pass / not in sudoers
//
// Системные события (специфика Linux без аналогов в Windows):
//
//	"Service Installed" / "Service Started" / "Service Stopped"
//	"System Startup" / "System Shutdown"
//	"OOM Killer", "Disk I/O", "Network Interface", etc.
//
// =============================================================================
func (s *SyslogParser) categorize(app, msg string) (category, severity, description string) {
	appL := strings.ToLower(app)
	msgL := strings.ToLower(msg)

	// ── Ядро ──────────────────────────────────────────────────────────────────
	if appL == "kernel" || strings.HasPrefix(appL, "kernel") {
		switch {
		case containsAny(msgL, "oom", "out of memory", "killed process"):
			proc := extractOOMProcess(msg)
			desc := "OOM Killer activated"
			if proc != "" {
				desc += ": killed " + proc
			}
			return "OOM Killer", "Warning", desc

		case containsAny(msgL, "usb", "ehci", "xhci", "new usb device"):
			return "USB Device", "Info", "USB device event: " + truncate(msg, 80)

		case containsAny(msgL, "panic", "oops", "bug:"):
			return "Kernel Error", "Error", "Kernel panic/oops: " + truncate(msg, 80)

		case containsAny(msgL, "i/o error", "ata", "nvme", "blk", "sd") && containsAny(msgL, "error", "fail"):
			return "Disk I/O", "Warning", "Disk I/O error: " + truncate(msg, 80)

		case containsAny(msgL, "link up", "link down", "link is up", "link is down"):
			return "Network Interface", "Info", "Network interface change: " + truncate(msg, 80)

		case containsAny(msgL, "iptables", "nf_tables", "netfilter"):
			return "Firewall", "Info", "Firewall event: " + truncate(msg, 80)

		case containsAny(msgL, "error", "fail", "segfault"):
			return "Kernel Error", "Error", "Kernel error: " + truncate(msg, 80)
		}
		return "Kernel", "Info", truncate(msg, 100)
	}

	// ── SSH — syslog может содержать SSH если auth.log не разделён ────────────
	// Категории унифицированы с parser_authlog.go
	if containsAny(appL, "sshd") {
		ip := extractIPFromMsg(msg)
		user := extractUserFromSSHMsg(msg)

		switch {
		case containsAny(msgL, "failed password", "authentication failure", "invalid user", "no such user"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure for "+user, ip)

		case containsAny(msgL, "accepted password", "accepted publickey", "accepted keyboard"):
			return "Logon Success", "Info",
				buildSuccessDesc("Successful logon by "+user, ip)

		case containsAny(msgL, "session opened"):
			return "Logon Success", "Info",
				"Successful logon by " + user + " (session opened)"

		case containsAny(msgL, "session closed"):
			return "Logoff", "Info", "Logoff by " + user + " (session closed)"

		case containsAny(msgL, "disconnected", "connection closed", "connection reset"):
			return "Logoff", "Info", "Logoff by " + user + " (disconnected)"

		case containsAny(msgL, "too many authentication", "maximum authentication"):
			return "Logon Failure", "Warning",
				buildFailDesc("Logon failure (max attempts) for "+user, ip)
		}
		return "Authentication", "Info", truncate(msg, 100)
	}

	// ── sudo ──────────────────────────────────────────────────────────────────
	// Unified: "Privilege Escalation" / "Privilege Escalation Failure"
	if appL == "sudo" {
		user := extractSudoUser(msg)
		cmd := extractSudoCommand(msg)

		switch {
		case containsAny(msgL, "incorrect password", "authentication failure"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (incorrect password)"

		case containsAny(msgL, "not in sudoers", "user not in sudoers", "is not allowed"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure by " + user + " (not in sudoers)"

		case strings.Contains(msg, "COMMAND="):
			desc := "Privilege escalation by " + user
			if cmd != "" {
				desc += ": " + cmd
			}
			return "Privilege Escalation", "Info", desc
		}
		return "Privilege Escalation", "Info", "Sudo: " + truncate(msg, 100)
	}

	// ── su ────────────────────────────────────────────────────────────────────
	if appL == "su" || appL == "su:" {
		switch {
		case containsAny(msgL, "failed", "authentication failure", "incorrect"):
			return "Privilege Escalation Failure", "Warning",
				"Privilege escalation failure (su failed)"
		case containsAny(msgL, "successful su", "session opened"):
			return "Privilege Escalation", "Info",
				"Privilege escalation (su success)"
		case containsAny(msgL, "session closed"):
			return "Logoff", "Info", "Logoff (su session closed)"
		}
		return "Privilege Escalation", "Info", truncate(msg, 100)
	}

	// ── PAM (в syslog попадает когда auth.log = syslog) ───────────────────────
	// Unified: "Logon Failure" / "Logon Success" / "Logoff"
	if strings.HasPrefix(appL, "pam") || strings.Contains(appL, "pam_") {
		user := extractPAMUser(msg)
		svc := extractServiceFromPAM(msg)

		switch {
		case containsAny(msgL, "authentication failure", "auth could not identify", "check pass"):
			return "Logon Failure", "Warning", "Logon failure for " + user

		case containsAny(msgL, "session opened"):
			if containsAny(svc, "sudo", "su") {
				return "Privilege Escalation", "Info",
					"Privilege escalation by " + user + " (PAM session)"
			}
			return "Logon Success", "Info",
				"Successful logon by " + user + " (session opened)"

		case containsAny(msgL, "session closed"):
			if containsAny(svc, "sudo", "su") {
				return "Logoff", "Info", "Logoff by " + user + " (sudo/su session closed)"
			}
			return "Logoff", "Info", "Logoff by " + user + " (session closed)"

		case containsAny(msgL, "account locked", "locked out", "maximum failure"):
			return "Account Lockout", "Warning", "Account locked: " + user

		case containsAny(msgL, "account unlocked"):
			return "Account Unlocked", "Info", "Account unlocked: " + user

		case containsAny(msgL, "password changed", "new password"):
			return "Password Change", "Info", "Password changed for " + user
		}
		return "Authentication", "Info", truncate(msg, 100)
	}

	// ── systemd / systemd-logind / systemctl ───────────────────────────────────
	// Unified: "System Startup" / "System Shutdown" совпадают с Windows 6005/6006/4608/4609
	if appL == "systemd" || strings.HasPrefix(appL, "systemd") {
		switch {
		// System Startup / Shutdown
		case containsAny(msgL, "reached target", "startup finished", "system is up"):
			return "System Startup", "Info", "System startup: " + truncate(msg, 80)
		case containsAny(msgL, "shutting down", "reboot", "poweroff", "halt"):
			return "System Shutdown", "Info", "System shutdown: " + truncate(msg, 80)

		// Services — unified: "Service Installed"/"Service Started"/"Service Stopped"
		case containsAny(msgL, "started") && !containsAny(msgL, "fail"):
			svc := extractSystemdUnit(msg)
			return "Service Started", "Info", fmt.Sprintf("Service started: %s", svc)
		case containsAny(msgL, "stopped", "deactivated successfully"):
			svc := extractSystemdUnit(msg)
			return "Service Stopped", "Info", fmt.Sprintf("Service stopped: %s", svc)
		case containsAny(msgL, "failed", "failure"):
			svc := extractSystemdUnit(msg)
			return "Service Failure", "Warning", fmt.Sprintf("Service failed: %s", svc)
		case containsAny(msgL, "reloaded"):
			svc := extractSystemdUnit(msg)
			return "Service Reloaded", "Info", fmt.Sprintf("Service reloaded: %s", svc)

		// Время
		case containsAny(msgL, "synchronized", "ntp", "time zone"):
			return "Time Sync", "Info", truncate(msg, 80)

		// Сессии (logind)
		case containsAny(msgL, "new session") && containsAny(appL, "logind"):
			user := extractLogindUser(msg)
			return "Logon Success", "Info", "Successful logon by " + user + " (new session)"
		case containsAny(msgL, "removed session") && containsAny(appL, "logind"):
			user := extractLogindUser(msg)
			return "Logoff", "Info", "Logoff by " + user + " (session removed)"
		}
		return "System Service", "Info", truncate(msg, 100)
	}

	// ── Cron ──────────────────────────────────────────────────────────────────
	if containsAny(appL, "cron", "crond", "anacron") {
		if containsAny(msgL, "error", "failed", "cannot") {
			return "Scheduled Task Error", "Warning", "Cron error: " + truncate(msg, 80)
		}
		return "Scheduled Task", "Info", "Cron: " + truncate(msg, 80)
	}

	// ── DHCP ──────────────────────────────────────────────────────────────────
	if containsAny(appL, "dhclient", "dhcpd", "dhcpcd") {
		switch {
		case containsAny(msgL, "bound", "obtained", "lease"):
			return "DHCP Lease", "Info", "DHCP lease: " + truncate(msg, 80)
		case containsAny(msgL, "fail", "no lease", "timeout"):
			return "DHCP Failure", "Warning", "DHCP failure: " + truncate(msg, 80)
		}
		return "DHCP", "Info", truncate(msg, 80)
	}

	// ── DNS ───────────────────────────────────────────────────────────────────
	if containsAny(appL, "named", "bind", "unbound", "dnsmasq") {
		if containsAny(msgL, "denied", "refused", "error") {
			return "DNS Error", "Warning", "DNS error: " + truncate(msg, 80)
		}
		return "DNS", "Info", truncate(msg, 80)
	}

	// ── NetworkManager / wpa_supplicant ───────────────────────────────────────
	if containsAny(appL, "networkmanager", "wpa_supplicant") {
		switch {
		case containsAny(msgL, "connected", "activated"):
			return "Network Connected", "Info", "Network connected: " + truncate(msg, 80)
		case containsAny(msgL, "disconnected", "deactivated", "failed"):
			return "Network Disconnected", "Info", "Network disconnected: " + truncate(msg, 80)
		}
		return "Network Manager", "Info", truncate(msg, 80)
	}

	// ── Безопасность: AppArmor / auditd / SELinux ─────────────────────────────
	if containsAny(appL, "apparmor", "audit", "auditd", "selinux") {
		if containsAny(msgL, "denied", "violation") {
			return "Security Policy Denied", "Warning", "Security policy denied: " + truncate(msg, 80)
		}
		return "Security Audit", "Info", truncate(msg, 80)
	}

	// ── Fail2Ban ──────────────────────────────────────────────────────────────
	if containsAny(appL, "fail2ban") {
		switch {
		case containsAny(msgL, "ban", "banned"):
			ip := extractIPFromMsg(msg)
			desc := "Intrusion blocked"
			if ip != "" {
				desc += ": " + ip
			}
			return "Intrusion Blocked", "Warning", desc
		case containsAny(msgL, "unban"):
			ip := extractIPFromMsg(msg)
			return "IP Unblocked", "Info", "IP unblocked: " + ip
		}
		return "Fail2Ban", "Info", truncate(msg, 80)
	}

	// ── S.M.A.R.T. / disk health ──────────────────────────────────────────────
	if containsAny(appL, "smartd", "hddtemp") {
		if containsAny(msgL, "error", "fail", "reallocated", "pending") {
			return "Disk Health Warning", "Warning", "Disk health: " + truncate(msg, 80)
		}
		return "Disk Health", "Info", truncate(msg, 80)
	}

	// ── Пакетный менеджер ─────────────────────────────────────────────────────
	if containsAny(appL, "apt", "dpkg", "yum", "dnf", "rpm", "pacman", "snap") {
		switch {
		case containsAny(msgL, "install", "upgrade", "updated"):
			return "Software Installation", "Info", "Package installed: " + truncate(msg, 80)
		case containsAny(msgL, "remove", "purge", "uninstall"):
			return "Software Removal", "Info", "Package removed: " + truncate(msg, 80)
		case containsAny(msgL, "error", "fail"):
			return "Package Error", "Warning", "Package error: " + truncate(msg, 80)
		}
		return "Package Management", "Info", truncate(msg, 80)
	}

	// ── Общие паттерны по тексту (fallback) ───────────────────────────────────
	switch {
	case containsAny(msgL, "panic", "fatal", "segfault", "abort", "core dumped"):
		return "Application Error", "Error", "Application error: " + truncate(msg, 80)
	case containsAny(msgL, "error", "failed"):
		return "Application Error", "Warning", truncate(msg, 100)
	case containsAny(msgL, "warn", "warning"):
		return "Application Warning", "Warning", truncate(msg, 100)
	}

	return "System Event", "Info", truncate(msg, 120)
}

func (s *SyslogParser) generateID(raw_log *logstructure.RawLog) string {
	data := raw_log.Log_source + raw_log.PC_name + raw_log.Username + raw_log.Raw_data
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ── Вспомогательные функции ────────────────────────────────────────────────────

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// extractIPFromMsg — общий, объявлен в parser_authlog.go

func extractUserFromSSHMsg(msg string) string {
	// "Failed password for <user> from ..."
	// "Accepted password for <user> from ..."
	if m := reUserSSH.FindStringSubmatch(msg); len(m) > 1 && m[1] != "invalid" {
		return m[1]
	}
	// "Invalid user <user> from ..."
	if m := reInvalidUser.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractSudoUser(msg string) string {
	// Sudo log: "username : TTY=pts/0 ; PWD=/ ; USER=root ; COMMAND=..."
	if m := reSudoUser.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractSudoCommand(msg string) string {
	if idx := strings.Index(msg, "COMMAND="); idx != -1 {
		cmd := strings.TrimSpace(msg[idx+8:])
		// Берём только имя исполняемого (до пробела)
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			p := strings.Split(parts[0], "/")
			return p[len(p)-1]
		}
	}
	return ""
}

func extractPAMUser(msg string) string {
	// "session opened for user root by ..."
	if m := rePAMUser.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractSystemdUnit(msg string) string {
	// "Started Apache HTTP Server." / "apache2.service: ..."
	if m := reSystemdUnit1.FindStringSubmatch(msg); len(m) > 1 {
		return strings.TrimRight(m[1], ".")
	}
	// "unitname.service: ..."
	if m := reSystemdUnit2.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return truncate(msg, 50)
}

func extractLogindUser(msg string) string {
	// "New session 5 of user root."
	if m := reLogindUser.FindStringSubmatch(msg); len(m) > 1 {
		return strings.TrimRight(m[1], ".")
	}
	return ""
}

func extractOOMProcess(msg string) string {
	if m := reOOMProcess.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}
