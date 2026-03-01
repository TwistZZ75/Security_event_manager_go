package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// installSystemdService создает systemd unit файл
func installSystemdService() error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=10
User=root
Environment="SIEM_SERVER_ADDR=localhost:50051"

[Install]
WantedBy=multi-user.target
`, serviceDesc, exepath)

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

// removeSystemdService удаляет systemd service
func removeSystemdService() error {
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
