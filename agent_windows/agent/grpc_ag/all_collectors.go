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
		if cfg.Collectors.Sysmon.Enabled {
			sysmoncollect := collector.NewSysmonCollector(cfg.Collectors.Sysmon.Channel, cfg.Collectors.Sysmon.EventID)
			if err := sysmoncollect.Start_collect(ctx); err != nil {
				log.Println("Failed to start Sysmon collector")
			} else {
				collectors = append(collectors, sysmoncollect)
			}
		}
	}

	return collectors
}
