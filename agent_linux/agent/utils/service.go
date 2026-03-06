package utils

import (
	"fmt"
	"os"
	"os/exec"
	"siem-agent/config"
)

type AgService struct {
}

// InstallSystemdService создает systemd unit файл
func (as *AgService) InstallSystemdService(cfg *config.Config) error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}
	servAddr := cfg.Server.Address

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=/etc/siem-agent
Restart=on-failure
RestartSec=10
User=root
Environment="SIEM_SERVER_ADDR=%s"

PermissionsStartOnly=true
ExecStartPre=/bin/mkdir -p /var/log/SIEM_service
ExecStartPre=/bin/chown syslog:adm /var/log/SIEM_service
ExecStartPre=/bin/chmod 755 /var/log/SIEM_service
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=SIEM_agent
[Install]
WantedBy=multi-user.target
`, serviceDesc, exepath, servAddr)

	serviceFile := "/etc/systemd/system/siem-agent.service"
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	// Перезагружаем systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	return nil
}

// RemoveSystemdService удаляет systemd service
func (as *AgService) RemoveSystemdService() error {
	// Останавливаем сервис
	exec.Command("systemctl", "stop", serviceName).Run()

	// Отключаем автозапуск
	exec.Command("systemctl", "disable", serviceName).Run()

	// Удаляем unit файл
	serviceFile := "/etc/systemd/system/siem-agent.service"
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %v", err)
	}

	// Перезагружаем systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	return nil
}
