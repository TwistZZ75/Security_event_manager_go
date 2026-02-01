package main

import (
	"fmt"
	collector "siem-agent/collector"
	"siem-agent/config"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Println("config truble")
	}

	baseInfo := collector.NewBaseCollectorInfo()
	cfg.Agent.Hostname = baseInfo.PC_name
	cfg.Agent.OS = baseInfo.OS

}
