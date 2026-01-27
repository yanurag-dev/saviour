package server

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the server configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Auth       AuthConfig       `yaml:"auth"`
	Alerting   AlertingConfig   `yaml:"alerting"`
	GoogleChat GoogleChatConfig `yaml:"google_chat"`
	CORS       CORSConfig       `yaml:"cors"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	DevMode        bool     `yaml:"dev_mode"`
}

// AlertingConfig holds alerting configuration
type AlertingConfig struct {
	Enabled               bool          `yaml:"enabled"`
	CheckInterval         time.Duration `yaml:"check_interval"`
	HeartbeatTimeout      time.Duration `yaml:"heartbeat_timeout"`
	DeduplicationEnabled  bool          `yaml:"deduplication_enabled"`
	DeduplicationWindow   time.Duration `yaml:"deduplication_window"`
	SystemCPUThreshold    float64       `yaml:"system_cpu_threshold"`
	SystemMemoryThreshold float64       `yaml:"system_memory_threshold"`
	SystemDiskThreshold   float64       `yaml:"system_disk_threshold"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	APIKeys []APIKey `yaml:"api_keys"`
}

// APIKey represents an API key with permissions
type APIKey struct {
	Key    string   `json:"key" yaml:"key"`
	Name   string   `json:"name" yaml:"name"`
	Scopes []string `json:"scopes" yaml:"scopes"`
}

// GoogleChatConfig holds Google Chat webhook settings
type GoogleChatConfig struct {
	Enabled      bool   `yaml:"enabled"`
	WebhookURL   string `yaml:"webhook_url"`
	DashboardURL string `yaml:"dashboard_url"`
}

// LoadConfig loads server configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Alerting.CheckInterval == 0 {
		cfg.Alerting.CheckInterval = 30 * time.Second
	}
	if cfg.Alerting.HeartbeatTimeout == 0 {
		cfg.Alerting.HeartbeatTimeout = 2 * time.Minute
	}
	if cfg.Alerting.DeduplicationWindow == 0 {
		cfg.Alerting.DeduplicationWindow = 5 * time.Minute
	}

	// Set default thresholds if not specified
	if cfg.Alerting.SystemCPUThreshold == 0 {
		cfg.Alerting.SystemCPUThreshold = 80.0
	}
	if cfg.Alerting.SystemMemoryThreshold == 0 {
		cfg.Alerting.SystemMemoryThreshold = 85.0
	}
	if cfg.Alerting.SystemDiskThreshold == 0 {
		cfg.Alerting.SystemDiskThreshold = 90.0
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if len(c.Auth.APIKeys) == 0 {
		return fmt.Errorf("at least one API key must be configured")
	}

	for i, key := range c.Auth.APIKeys {
		if key.Key == "" {
			return fmt.Errorf("API key %d: key is required", i)
		}
		if key.Name == "" {
			return fmt.Errorf("API key %d: name is required", i)
		}
	}

	if c.GoogleChat.Enabled && c.GoogleChat.WebhookURL == "" {
		return fmt.Errorf("Google Chat webhook URL is required when enabled")
	}

	// Validate alerting configuration
	if c.Alerting.Enabled {
		if c.Alerting.CheckInterval <= 0 {
			return fmt.Errorf("alerting check_interval must be > 0, got: %v", c.Alerting.CheckInterval)
		}
		if c.Alerting.HeartbeatTimeout <= 0 {
			return fmt.Errorf("alerting heartbeat_timeout must be > 0, got: %v", c.Alerting.HeartbeatTimeout)
		}
		if c.Alerting.DeduplicationEnabled && c.Alerting.DeduplicationWindow <= 0 {
			return fmt.Errorf("alerting deduplication_window must be > 0 when deduplication is enabled, got: %v", c.Alerting.DeduplicationWindow)
		}

		// Validate threshold percentages (0-100)
		if c.Alerting.SystemCPUThreshold < 0 || c.Alerting.SystemCPUThreshold > 100 {
			return fmt.Errorf("alerting system_cpu_threshold must be between 0 and 100, got: %.2f", c.Alerting.SystemCPUThreshold)
		}
		if c.Alerting.SystemMemoryThreshold < 0 || c.Alerting.SystemMemoryThreshold > 100 {
			return fmt.Errorf("alerting system_memory_threshold must be between 0 and 100, got: %.2f", c.Alerting.SystemMemoryThreshold)
		}
		if c.Alerting.SystemDiskThreshold < 0 || c.Alerting.SystemDiskThreshold > 100 {
			return fmt.Errorf("alerting system_disk_threshold must be between 0 and 100, got: %.2f", c.Alerting.SystemDiskThreshold)
		}
	}

	// Validate CORS configuration
	if c.CORS.Enabled && !c.CORS.DevMode && len(c.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS enabled in production mode but no allowed_origins configured")
	}

	return nil
}

// Address returns the server address in host:port format
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
