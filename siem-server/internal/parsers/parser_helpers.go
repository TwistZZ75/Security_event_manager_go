package parsers

import (
	"regexp"
	"strings"
)

// =============================================================================
// Общие вспомогательные функции для всех парсеров.
// Файл: siem-server/internal/parsers/parser_helpers.go
// =============================================================================

// containsAny возвращает true если src содержит хотя бы одну из подстрок subs.
func containsAny(src string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(src, s) {
			return true
		}
	}
	return false
}

// buildFailDesc формирует описание провала входа с опциональным IP.
// Пример: "Logon failure for bob from 10.0.0.1"
func buildFailDesc(base, ip string) string {
	if ip != "" {
		return base + " from " + ip
	}
	return base
}

// buildSuccessDesc формирует описание успешного входа с опциональным IP.
// Пример: "Successful logon by bob from 10.0.0.1"
func buildSuccessDesc(base, ip string) string {
	if ip != "" {
		return base + " from " + ip
	}
	return base
}

// extractIPFromMsg пытается найти IPv4-адрес в строке syslog/auth-лога.
// sshd пишет: "Failed password for bob from 10.0.0.1 port 22"
func extractIPFromMsg(msg string) string {
	re := regexp.MustCompile(`from\s+(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	if m := re.FindStringSubmatch(msg); len(m) > 1 {
		return m[1]
	}
	return ""
}

// extractServiceFromPAM извлекает имя сервиса из PAM-сообщения.
// Формат: "pam_unix(sshd:session): session opened for user root by ..."
//
//	^^^^
func extractServiceFromPAM(msg string) string {
	re := regexp.MustCompile(`\(([^:)]+):`)
	if m := re.FindStringSubmatch(msg); len(m) > 1 {
		return strings.ToLower(m[1])
	}
	return ""
}
