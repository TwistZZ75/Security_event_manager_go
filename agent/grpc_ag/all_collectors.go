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
			if cfg.Collectors.Syslog.Enabled {
				sysmoncollect := collector.NewSysmonCollector(cfg.Collectors.Sysmon.Channel, cfg.Collectors.Sysmon.EventID)
				if err := sysmoncollect.Start_collect(ctx); err != nil {
					log.Println("Failed to start Sysmon collector")
				} else {
					collectors = append(collectors, sysmoncollect)
				}
			}
		}
	} else { //запуск логирования linux
		//запуск сбора логов suricata
		if cfg.Collectors.Suricata.Enabled {
			suricatacollect := collector.NewSuricataCollector(cfg.Collectors.Suricata.LogPath)
			if err := suricatacollect.Start_collect(ctx); err != nil {
				log.Println("Failed to start Suricata collector")
			} else {
				collectors = append(collectors, suricatacollect)
			}
		}
		//запуск сбора логов squid
		if cfg.Collectors.Squid.Enabled {
			squidcollector := collector.NewSquidCollector(cfg.Collectors.Squid.LogPath)
			if err := squidcollector.Start_collect(ctx); err != nil {
				log.Println("Failed to start Squid collector")
			} else {
				collectors = append(collectors, squidcollector)
			}
		}
		//запуск сбора логов sysmon
		if cfg.Collectors.Syslog.Enabled {
			syslogcollector := collector.NewSyslogCollector(cfg.Collectors.Syslog.LogPath)
			if err := syslogcollector.Start_collect(ctx); err != nil {
				log.Println("Failed to start Syslog collector")
			} else {
				collectors = append(collectors, syslogcollector)
			}
		}
	}
	return collectors
}
