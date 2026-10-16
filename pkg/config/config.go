package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Ingest   IngestConfig   `json:"ingest"`
	Targets  []TargetConfig `json:"targets"`
	Settings SettingsConfig `json:"settings"`
}

type IngestConfig struct {
	Port      int    `json:"port"`
	StreamKey string `json:"streamKey"`
	App       string `json:"app"`
}

type TargetConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

type SettingsConfig struct {
	WebDashboardPort         int `json:"webDashboardPort"`
	ReconnectMaxDelaySeconds int `json:"reconnectMaxDelaySeconds"`
	GOPCacheMaxBytes         int `json:"gopCacheMaxBytes"`
	TargetQueueSize          int `json:"targetQueueSize"`
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			exampleData, errEx := os.ReadFile("config.example.json")
			if errEx == nil {
				_ = os.WriteFile(filePath, exampleData, 0644)
				return nil, fmt.Errorf("arquivo '%s' não existia e foi criado a partir de 'config.example.json'. Configure suas chaves de stream nele", filePath)
			}
		}
		return nil, fmt.Errorf("falha ao ler '%s': %w", filePath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("falha ao parsear JSON de '%s': %w", filePath, err)
	}

	if cfg.Ingest.Port <= 0 {
		cfg.Ingest.Port = 1935
	}
	if cfg.Ingest.App == "" {
		cfg.Ingest.App = "live"
	}
	if cfg.Settings.WebDashboardPort <= 0 {
		cfg.Settings.WebDashboardPort = 3008
	}
	if cfg.Settings.ReconnectMaxDelaySeconds <= 0 {
		cfg.Settings.ReconnectMaxDelaySeconds = 30
	}
	if cfg.Settings.GOPCacheMaxBytes <= 0 {
		cfg.Settings.GOPCacheMaxBytes = 10 * 1024 * 1024
	}
	if cfg.Settings.TargetQueueSize <= 0 {
		cfg.Settings.TargetQueueSize = 256
	}

	return &cfg, nil
}
