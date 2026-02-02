package main

import (
	"fmt"
	collector "siem-agent/collector"
	"siem-agent/config"
)

func main() {
	//загрузка конфига
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Println("config truble")
	}

	baseInfo := collector.NewBaseCollectorInfo()
	cfg.Agent.Hostname = baseInfo.PC_name
	cfg.Agent.OS = baseInfo.OS
	fmt.Printf("Config: %v\n", cfg)
}
