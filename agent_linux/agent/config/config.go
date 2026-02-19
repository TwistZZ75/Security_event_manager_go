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
	Suricata SuricataConfig `mapstructure:"suricata"`
	Squid    SquidConfig    `mapstructure:"squid"`
	Syslog   SyslogConfig   `mapstructure:"syslog"`
}

type SuricataConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	LogPath string `mapstructure:"log_path"`
}

type SquidConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	LogPath string `mapstructure:"log_path"`
}

type SyslogConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	LogPath string `mapstructure:"log_path"`
}

// функция загрузки yaml конфига
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("agent_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	baseInfo := collector.NewBaseCollectorInfo()
	// Значения по умолчанию
	viper.SetDefault("agent.hostname", baseInfo.PC_name)
	viper.SetDefault("agent.os", baseInfo.OS)
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
