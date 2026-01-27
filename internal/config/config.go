package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration
type Config struct {
	Agent        AgentConfig        `yaml:"agent"`
	Metrics      MetricsConfig      `yaml:"metrics"`
	HealthChecks []HealthCheckConfig `yaml:"health_checks"`
	Alerts       AlertsConfig       `yaml:"alerts"`
}

// AgentConfig contains agent-specific settings
type AgentConfig struct {
	Name            string        `yaml:"name"`
	ServerURL       string        `yaml:"server_url"`
	APIKey          string        `yaml:"api_key"`
	CollectInterval time.Duration `yaml:"collect_interval"`
}

// MetricsConfig defines what metrics to collect
type MetricsConfig struct {
	System     bool              `yaml:"system"`
	Processes  []ProcessConfig   `yaml:"processes"`
	DiskMounts []string          `yaml:"disk_mounts"`
}

// ProcessConfig defines a process to monitor
type ProcessConfig struct {
	Name             string `yaml:"name"`
	RestartOnFailure bool   `yaml:"restart_on_failure"`
}

// HealthCheckConfig defines a health check
type HealthCheckConfig struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"` // http, tcp, ping, script
	URL      string        `yaml:"url,omitempty"`
	Host     string        `yaml:"host,omitempty"`
	Port     int           `yaml:"port,omitempty"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// AlertsConfig defines alert thresholds
type AlertsConfig struct {
	CPUThreshold    float64 `yaml:"cpu_threshold"`
	MemoryThreshold float64 `yaml:"memory_threshold"`
	DiskThreshold   float64 `yaml:"disk_threshold"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Agent.CollectInterval == 0 {
		cfg.Agent.CollectInterval = 10 * time.Second
	}
	if cfg.Agent.Name == "" {
		hostname, _ := os.Hostname()
		cfg.Agent.Name = hostname
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if c.Agent.CollectInterval < time.Second {
		return fmt.Errorf("collect_interval must be at least 1 second")
	}
	return nil
}
