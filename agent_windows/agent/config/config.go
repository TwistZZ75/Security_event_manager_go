package config

import (
	collector "siem-agent/collector"

	"github.com/spf13/viper"
)

type Config struct {
	Agent      AgentConfig      `mapstructure:"agent"`
	Server     ServerConfig     `mapstructure:"server"`
	Collectors CollectorsConfig `mapstructure:"collectors"`
}

type AgentConfig struct {
	Hostname string `mapstructure:"hostname"`
	OS       string `mapstructure:"os"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
	Port    string `mapstructure:"port"`
}

type CollectorsConfig struct {
	WinEvent WinEventConfig `mapstructure:"winevent"`
	Sysmon   SysmonConfig   `mapstructure:"sysmon"`
}

type WinEventConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Channel string `mapstructure:"channel"`
	EventID string `mapstructure:"eventid"`
}

type SysmonConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Channel string `mapstructure:"channel"`
	EventID string `mapstructure:"eventid"`
}

// функция загрузки yaml конфига
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("agent_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")
	viper.AddConfigPath("C:\\Program Files\\SIEM Agent\\")

	viper.AutomaticEnv()

	baseInfo := collector.NewBaseCollectorInfo()
	// Значения по умолчанию
	viper.SetDefault("agent.hostname", baseInfo.PC_name)
	viper.SetDefault("agent.os", baseInfo.OS)
	viper.SetDefault("server.address", "localhost")
	viper.SetDefault("server.port", "5000")

	// Настройки Windows Event Log по умолчанию
	viper.SetDefault("collectors.winevent.enabled", false)
	viper.SetDefault("collectors.winevent.channel", "Security")
	viper.SetDefault("collectors.winevent.eventid", "*")

	// Настройки Sysmon по умолчанию
	viper.SetDefault("collectors.sysmon.enabled", false)
	viper.SetDefault("collectors.sysmon.channel", "Microsoft-Windows-Sysmon/Operational")
	viper.SetDefault("collectors.sysmon.eventid", "*")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
