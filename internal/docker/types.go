package docker

import "time"

// ContainerInfo represents a Docker container with its metrics
type ContainerInfo struct {
	// Identity
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	ImageID string            `json:"image_id"`
	Labels  map[string]string `json:"labels"`

	// State
	State         string    `json:"state"`          // running, exited, paused, restarting, dead
	Status        string    `json:"status"`         // Up 2 hours, Exited (0) 5 minutes ago
	Health        string    `json:"health"`         // healthy, unhealthy, starting, none
	ExitCode      int       `json:"exit_code"`      // Exit code when stopped
	OOMKilled     bool      `json:"oom_killed"`     // Was killed due to OOM
	RestartCount  int       `json:"restart_count"`  // Number of times restarted
	
	// Timestamps
	Created   time.Time `json:"created"`
	StartedAt time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`

	// Resource Metrics
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`    // bytes
	MemoryLimit   uint64  `json:"memory_limit"`    // bytes
	MemoryPercent float64 `json:"memory_percent"`

	// Network I/O
	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`

	// Block I/O
	BlockReadBytes  uint64 `json:"block_read_bytes"`
	BlockWriteBytes uint64 `json:"block_write_bytes"`

	// PIDs
	PIDs uint64 `json:"pids"` // Number of processes in container
}

// FilterConfig defines container filtering options
type FilterConfig struct {
	// Monitor all containers (default: true)
	MonitorAll bool

	// Filter by labels (e.g., "monitor=true", "env=production")
	Labels []string

	// Filter by name patterns (e.g., "api-*", "web-*")
	Names []string

	// Filter by image patterns (e.g., "mycompany/*", "nginx:*")
	Images []string
}

// AlertConfig defines alert thresholds for containers
type AlertConfig struct {
	CPUThreshold     float64 // CPU usage percentage threshold
	MemoryThreshold  float64 // Memory usage percentage threshold
	RestartThreshold int     // Number of restarts to alert on
	RestartWindow    int     // Time window in seconds for restart count
}

// ContainerAlertOverride defines per-container alert overrides
type ContainerAlertOverride struct {
	Name             string  // Container name pattern
	CPUThreshold     float64 // Override CPU threshold
	MemoryThreshold  float64 // Override memory threshold
	RestartThreshold int     // Override restart threshold
}
