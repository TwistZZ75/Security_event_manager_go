package grpc_ag

import (
	"context"
	"log"
	"siem-agent/collector"
	"siem-agent/config"
)

// функция старта коллекторов, проверяет какие коллекторы включены в конфиге и запускает их
// принимает контекст и ссылку на конфиг
// возвращает массив запущенных коллекторов
func StartAllCollectors(ctx context.Context, cfg *config.Config) []collector.LogCollector {
	var collectors []collector.LogCollector

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
	return collectors
}
