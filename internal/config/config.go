package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ServiceConfig struct {
	Auth string `json:"auth"`
	TPM  int    `json:"tpm"`
}

type GlobalConfig struct {
	OllamaURL         string `json:"ollama_url"`
	AdminAPIKey       string `json:"admin_api_key"`
	Port              string `json:"port"`
	MaxBodySize       int    `json:"max_body_size"`
	StreamBufferSize  int    `json:"stream_buffer_size"`
	StreamTimeoutSecs int    `json:"stream_timeout_seconds"`
	TimeoutSecs       int    `json:"timeout_seconds"`
}

type Config struct {
	Global   GlobalConfig             `json:"_global"`
	Services map[string]ServiceConfig `json:"services"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults for global config
	if cfg.Global.Port == "" {
		cfg.Global.Port = "3000"
	}
	if cfg.Global.OllamaURL == "" {
		cfg.Global.OllamaURL = "http://localhost:11434"
	}
	if cfg.Global.MaxBodySize == 0 {
		cfg.Global.MaxBodySize = 10 * 1024 * 1024 // 10 MB
	}
	if cfg.Global.StreamBufferSize == 0 {
		cfg.Global.StreamBufferSize = 4096
	}
	if cfg.Global.StreamTimeoutSecs == 0 {
		cfg.Global.StreamTimeoutSecs = 30
	}
	if cfg.Global.TimeoutSecs == 0 {
		cfg.Global.TimeoutSecs = 15
	}

	// Set defaults for each service
	for name, svc := range cfg.Services {
		if svc.TPM == 0 {
			svc.TPM = 60
		}
		cfg.Services[name] = svc
	}

	if cfg.Global.AdminAPIKey == "" {
		return nil, fmt.Errorf("missing required _global.admin_api_key")
	}
	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("no services configured")
	}
	for name, svc := range cfg.Services {
		if svc.Auth == "" {
			return nil, fmt.Errorf("missing auth for service %q", name)
		}
	}

	return &cfg, nil
}

func (c *Config) GetService(name string) (*ServiceConfig, error) {
	svc, ok := c.Services[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return &svc, nil
}

func (c *Config) GetGlobal() GlobalConfig {
	return c.Global
}

func (g GlobalConfig) StreamTimeout() time.Duration {
	return time.Duration(g.StreamTimeoutSecs) * time.Second
}
