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
	Docker     DockerConfig      `yaml:"docker"`
}

// DockerConfig defines Docker monitoring settings
type DockerConfig struct {
	Enabled    bool                       `yaml:"enabled"`
	Socket     string                     `yaml:"socket"`
	MonitorAll bool                       `yaml:"monitor_all"`
	Filters    DockerFilterConfig         `yaml:"filters"`
	Alerts     DockerAlertsConfig         `yaml:"alerts"`
}

// DockerFilterConfig defines container filtering options
type DockerFilterConfig struct {
	Labels []string `yaml:"labels"`
	Names  []string `yaml:"names"`
	Images []string `yaml:"images"`
}

// DockerAlertsConfig defines container alert thresholds
type DockerAlertsConfig struct {
	Default   ContainerAlertThreshold   `yaml:"default"`
	Overrides []ContainerAlertOverride  `yaml:"overrides"`
}

// ContainerAlertThreshold defines default alert thresholds for containers
type ContainerAlertThreshold struct {
	CPUThreshold     float64 `yaml:"cpu_threshold"`
	MemoryThreshold  float64 `yaml:"memory_threshold"`
	RestartThreshold int     `yaml:"restart_threshold"`
	RestartWindow    string  `yaml:"restart_window"` // e.g., "300s", "5m"
}

// ContainerAlertOverride defines per-container alert overrides
type ContainerAlertOverride struct {
	Name             string  `yaml:"name"`
	CPUThreshold     float64 `yaml:"cpu_threshold,omitempty"`
	MemoryThreshold  float64 `yaml:"memory_threshold,omitempty"`
	RestartThreshold int     `yaml:"restart_threshold,omitempty"`
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

	// Docker defaults
	if cfg.Metrics.Docker.Enabled {
		if cfg.Metrics.Docker.Socket == "" {
			cfg.Metrics.Docker.Socket = "/var/run/docker.sock"
		}
		if cfg.Metrics.Docker.Alerts.Default.CPUThreshold == 0 {
			cfg.Metrics.Docker.Alerts.Default.CPUThreshold = 80.0
		}
		if cfg.Metrics.Docker.Alerts.Default.MemoryThreshold == 0 {
			cfg.Metrics.Docker.Alerts.Default.MemoryThreshold = 90.0
		}
		if cfg.Metrics.Docker.Alerts.Default.RestartThreshold == 0 {
			cfg.Metrics.Docker.Alerts.Default.RestartThreshold = 5
		}
		if cfg.Metrics.Docker.Alerts.Default.RestartWindow == "" {
			cfg.Metrics.Docker.Alerts.Default.RestartWindow = "300s"
		}
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
