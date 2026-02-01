package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Agent      AgentConfig
	Server     ServerConfig
	Collectors CollectorsConfig
}

type AgentConfig struct {
	Hostname string
	OS       string
}

type ServerConfig struct {
	Address string
	Port    string
}

type CollectorsConfig struct {
	WinEvent WinEventConfig
	Sysmon   SysmonConfig
	Suricata SuricataConfig
	Squid    SquidConfig
	Syslog   SyslogConfig
}

type WinEventConfig struct {
	Enabled bool
	Channel string
	EventID string
}

type SysmonConfig struct {
	Enabled bool
	Channel string
	EventID string
}

type SuricataConfig struct {
	Enabled bool
	LogPath string
}

type SquidConfig struct {
	Enabled bool
	LogPath string
}

type SyslogConfig struct {
	Enabled bool
	LogPath string
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("agent_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	// Значения по умолчанию
	viper.SetDefault("server.address", "localhost")
	viper.SetDefault("server.port", "5000")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
