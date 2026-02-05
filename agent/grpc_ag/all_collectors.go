package grpc_ag

import (
	"context"
	"log"
	"siem-agent/collector"
	"siem-agent/config"
	"strings"
)

// функция старта коллекторов, проверяет какие коллекторы включены в конфиге и запускает их
// принимает контекст и ссылку на конфиг
// возвращает массив запущенных коллекторов
func StartAllCollectors(ctx context.Context, cfg *config.Config) []collector.LogCollector {
	var collectors []collector.LogCollector

	//запуск логирования windows
	if strings.Contains(cfg.Agent.OS, "windows") {
		//запуск логирования winEvent
		if cfg.Collectors.WinEvent.Enabled {
			wincollect := collector.NewWinEventCollector(cfg.Collectors.WinEvent.Channel, cfg.Collectors.WinEvent.EventID)
			if err := wincollect.Start_collect(ctx); err != nil {
				log.Println("Failed to start WinEvent collector")
			} else {
				collectors = append(collectors, wincollect)
			}
		}
		//запуск логирования sysmon
		if cfg.Collectors.Sysmon.Enabled {

		}
	}
	//запуск логирования linux
	if strings.Contains(cfg.Agent.OS, "linux") {
		//запуск сбора логов suricata
		if cfg.Collectors.Suricata.Enabled {

		}
		//запуск сбора логов squid
		if cfg.Collectors.Squid.Enabled {

		}
		//запуск сбора логов sysmon
		if cfg.Collectors.Syslog.Enabled {

		}
	}
	return collectors
}
