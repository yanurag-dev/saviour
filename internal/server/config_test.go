package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: 9090

auth:
  api_keys:
    - key: "test-key-123"
      name: "test-client"
      scopes: ["metrics:write"]

alerting:
  enabled: true
  check_interval: 45s
  heartbeat_timeout: 3m
  system_cpu_threshold: 85.0
  system_memory_threshold: 90.0
  system_disk_threshold: 95.0

google_chat:
  enabled: true
  webhook_url: "https://chat.googleapis.com/v1/spaces/xxx"
  dashboard_url: "https://dashboard.example.com"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify server config
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %v, want 9090", cfg.Server.Port)
	}

	// Verify auth config
	if len(cfg.Auth.APIKeys) != 1 {
		t.Fatalf("Expected 1 API key, got %d", len(cfg.Auth.APIKeys))
	}
	if cfg.Auth.APIKeys[0].Key != "test-key-123" {
		t.Errorf("APIKey.Key = %v, want test-key-123", cfg.Auth.APIKeys[0].Key)
	}
	if cfg.Auth.APIKeys[0].Name != "test-client" {
		t.Errorf("APIKey.Name = %v, want test-client", cfg.Auth.APIKeys[0].Name)
	}
	if len(cfg.Auth.APIKeys[0].Scopes) != 1 || cfg.Auth.APIKeys[0].Scopes[0] != "metrics:write" {
		t.Errorf("APIKey.Scopes = %v, want [metrics:write]", cfg.Auth.APIKeys[0].Scopes)
	}

	// Verify alerting config
	if !cfg.Alerting.Enabled {
		t.Error("Alerting should be enabled")
	}
	if cfg.Alerting.CheckInterval != 45*time.Second {
		t.Errorf("CheckInterval = %v, want 45s", cfg.Alerting.CheckInterval)
	}
	if cfg.Alerting.HeartbeatTimeout != 3*time.Minute {
		t.Errorf("HeartbeatTimeout = %v, want 3m", cfg.Alerting.HeartbeatTimeout)
	}
	if cfg.Alerting.SystemCPUThreshold != 85.0 {
		t.Errorf("SystemCPUThreshold = %v, want 85.0", cfg.Alerting.SystemCPUThreshold)
	}

	// Verify Google Chat config
	if !cfg.GoogleChat.Enabled {
		t.Error("GoogleChat should be enabled")
	}
	if cfg.GoogleChat.WebhookURL != "https://chat.googleapis.com/v1/spaces/xxx" {
		t.Errorf("WebhookURL incorrect")
	}
}

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config
	configContent := `
auth:
  api_keys:
    - key: "test-key"
      name: "test"
      scopes: ["metrics:write"]
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify defaults applied
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Default host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Default port = %v, want 8080", cfg.Server.Port)
	}
	if cfg.Alerting.CheckInterval != 30*time.Second {
		t.Errorf("Default CheckInterval = %v, want 30s", cfg.Alerting.CheckInterval)
	}
	if cfg.Alerting.HeartbeatTimeout != 2*time.Minute {
		t.Errorf("Default HeartbeatTimeout = %v, want 2m", cfg.Alerting.HeartbeatTimeout)
	}
	if cfg.Alerting.DeduplicationWindow != 5*time.Minute {
		t.Errorf("Default DeduplicationWindow = %v, want 5m", cfg.Alerting.DeduplicationWindow)
	}
	if cfg.Alerting.SystemCPUThreshold != 80.0 {
		t.Errorf("Default SystemCPUThreshold = %v, want 80.0", cfg.Alerting.SystemCPUThreshold)
	}
	if cfg.Alerting.SystemMemoryThreshold != 85.0 {
		t.Errorf("Default SystemMemoryThreshold = %v, want 85.0", cfg.Alerting.SystemMemoryThreshold)
	}
	if cfg.Alerting.SystemDiskThreshold != 90.0 {
		t.Errorf("Default SystemDiskThreshold = %v, want 90.0", cfg.Alerting.SystemDiskThreshold)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: invalid_port
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	_, err = LoadConfig(configPath)

	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Auth: AuthConfig{
			APIKeys: []APIKey{
				{Key: "test-key", Name: "test", Scopes: []string{"metrics:write"}},
			},
		},
		Alerting: AlertingConfig{
			Enabled:               true,
			CheckInterval:         30 * time.Second,
			HeartbeatTimeout:      2 * time.Minute,
			DeduplicationEnabled:  true,
			DeduplicationWindow:   5 * time.Minute,
			SystemCPUThreshold:    80.0,
			SystemMemoryThreshold: 85.0,
			SystemDiskThreshold:   90.0,
		},
		GoogleChat: GoogleChatConfig{
			Enabled:    true,
			WebhookURL: "https://chat.googleapis.com/v1/spaces/xxx",
		},
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validation failed for valid config: %v", err)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
		{"port way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: tt.port},
				Auth: AuthConfig{
					APIKeys: []APIKey{{Key: "test", Name: "test"}},
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for port %d", tt.port)
			}
		})
	}
}

func TestValidate_NoAPIKeys(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth:   AuthConfig{APIKeys: []APIKey{}},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for empty API keys")
	}
}

func TestValidate_APIKeyMissingKey(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{
				{Key: "", Name: "test"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing API key")
	}
}

func TestValidate_APIKeyMissingName(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{
				{Key: "test-key", Name: ""},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing API key name")
	}
}

func TestValidate_GoogleChatEnabledWithoutWebhook(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		GoogleChat: GoogleChatConfig{
			Enabled:    true,
			WebhookURL: "",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for Google Chat enabled without webhook URL")
	}
}

func TestValidate_AlertingInvalidCheckInterval(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		Alerting: AlertingConfig{
			Enabled:       true,
			CheckInterval: 0,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid check interval")
	}
}

func TestValidate_AlertingInvalidHeartbeatTimeout(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		Alerting: AlertingConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			HeartbeatTimeout: -1 * time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid heartbeat timeout")
	}
}

func TestValidate_AlertingInvalidDeduplicationWindow(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		Alerting: AlertingConfig{
			Enabled:              true,
			CheckInterval:        30 * time.Second,
			HeartbeatTimeout:     2 * time.Minute,
			DeduplicationEnabled: true,
			DeduplicationWindow:  0,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid deduplication window")
	}
}

func TestValidate_AlertingInvalidThresholds(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		field     string
	}{
		{"CPU negative", -1.0, "cpu"},
		{"CPU too high", 101.0, "cpu"},
		{"Memory negative", -5.0, "memory"},
		{"Memory too high", 150.0, "memory"},
		{"Disk negative", -10.0, "disk"},
		{"Disk too high", 200.0, "disk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Auth: AuthConfig{
					APIKeys: []APIKey{{Key: "test", Name: "test"}},
				},
				Alerting: AlertingConfig{
					Enabled:               true,
					CheckInterval:         30 * time.Second,
					HeartbeatTimeout:      2 * time.Minute,
					SystemCPUThreshold:    80.0,
					SystemMemoryThreshold: 85.0,
					SystemDiskThreshold:   90.0,
				},
			}

			switch tt.field {
			case "cpu":
				cfg.Alerting.SystemCPUThreshold = tt.threshold
			case "memory":
				cfg.Alerting.SystemMemoryThreshold = tt.threshold
			case "disk":
				cfg.Alerting.SystemDiskThreshold = tt.threshold
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for %s threshold %.2f", tt.field, tt.threshold)
			}
		})
	}
}

func TestValidate_CORSEnabledWithoutOrigins(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		CORS: CORSConfig{
			Enabled:        true,
			DevMode:        false,
			AllowedOrigins: []string{},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for CORS enabled without allowed origins")
	}
}

func TestValidate_CORSDevModeWithoutOrigins(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		CORS: CORSConfig{
			Enabled:        true,
			DevMode:        true,
			AllowedOrigins: []string{},
		},
	}

	// Dev mode should allow empty origins
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Dev mode CORS should allow empty origins: %v", err)
	}
}

func TestValidate_AlertingDisabled(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{{Key: "test", Name: "test"}},
		},
		Alerting: AlertingConfig{
			Enabled: false,
			// Invalid values, but should not be checked when disabled
			CheckInterval:    0,
			HeartbeatTimeout: 0,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validation should pass when alerting disabled: %v", err)
	}
}

func TestAddress(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"localhost", "localhost", 8080, "localhost:8080"},
		{"all interfaces", "0.0.0.0", 9090, "0.0.0.0:9090"},
		{"specific IP", "192.168.1.10", 3000, "192.168.1.10:3000"},
		{"IPv6", "::1", 8080, "::1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Host: tt.host,
					Port: tt.port,
				},
			}

			got := cfg.Address()
			if got != tt.want {
				t.Errorf("Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidate_MultipleAPIKeys(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			APIKeys: []APIKey{
				{Key: "key1", Name: "client1", Scopes: []string{"metrics:write"}},
				{Key: "key2", Name: "client2", Scopes: []string{"metrics:read"}},
				{Key: "key3", Name: "client3", Scopes: []string{"metrics:write", "alerts:read"}},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validation failed for multiple valid API keys: %v", err)
	}
}

func TestValidate_EdgeCaseThresholds(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		valid bool
	}{
		{"zero", 0.0, true},
		{"exactly 100", 100.0, true},
		{"0.01", 0.01, true},
		{"99.99", 99.99, true},
		{"-0.01", -0.01, false},
		{"100.01", 100.01, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Auth: AuthConfig{
					APIKeys: []APIKey{{Key: "test", Name: "test"}},
				},
				Alerting: AlertingConfig{
					Enabled:               true,
					CheckInterval:         30 * time.Second,
					HeartbeatTimeout:      2 * time.Minute,
					SystemCPUThreshold:    tt.value,
					SystemMemoryThreshold: tt.value,
					SystemDiskThreshold:   tt.value,
				},
			}

			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("Expected valid threshold %.2f to pass, got error: %v", tt.value, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected invalid threshold %.2f to fail", tt.value)
			}
		})
	}
}
