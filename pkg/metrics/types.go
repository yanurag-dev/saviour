package metrics

import "time"

// SystemMetrics contains all system-level metrics
type SystemMetrics struct {
	Timestamp   time.Time          `json:"timestamp"`
	AgentName   string             `json:"agent_name"`
	CPU         CPUMetrics         `json:"cpu"`
	Memory      MemoryMetrics      `json:"memory"`
	Disk        []DiskMetrics      `json:"disk"`
	Network     NetworkMetrics     `json:"network"`
	SystemInfo  SystemInfo         `json:"system_info"`
	Containers  []ContainerMetrics `json:"containers,omitempty"` // Docker container metrics
}

// CPUMetrics contains CPU usage information
type CPUMetrics struct {
	UsagePercent    float64   `json:"usage_percent"`     // Overall CPU usage
	PerCorePercent  []float64 `json:"per_core_percent"`  // Per-core usage
	LoadAvg1        float64   `json:"load_avg_1"`        // 1-minute load average
	LoadAvg5        float64   `json:"load_avg_5"`        // 5-minute load average
	LoadAvg15       float64   `json:"load_avg_15"`       // 15-minute load average
}

// MemoryMetrics contains memory usage information
type MemoryMetrics struct {
	Total       uint64  `json:"total"`        // Total memory in bytes
	Available   uint64  `json:"available"`    // Available memory in bytes
	Used        uint64  `json:"used"`         // Used memory in bytes
	UsedPercent float64 `json:"used_percent"` // Used percentage
	SwapTotal   uint64  `json:"swap_total"`   // Total swap in bytes
	SwapUsed    uint64  `json:"swap_used"`    // Used swap in bytes
	SwapPercent float64 `json:"swap_percent"` // Swap usage percentage
}

// DiskMetrics contains disk usage information for a single mount point
type DiskMetrics struct {
	MountPoint  string  `json:"mount_point"`  // Mount point path
	Device      string  `json:"device"`       // Device name
	FSType      string  `json:"fs_type"`      // Filesystem type
	Total       uint64  `json:"total"`        // Total space in bytes
	Used        uint64  `json:"used"`         // Used space in bytes
	Free        uint64  `json:"free"`         // Free space in bytes
	UsedPercent float64 `json:"used_percent"` // Used percentage
	InodesTotal uint64  `json:"inodes_total"` // Total inodes
	InodesUsed  uint64  `json:"inodes_used"`  // Used inodes
	InodesFree  uint64  `json:"inodes_free"`  // Free inodes
}

// NetworkMetrics contains network statistics
type NetworkMetrics struct {
	BytesSent   uint64 `json:"bytes_sent"`   // Total bytes sent
	BytesRecv   uint64 `json:"bytes_recv"`   // Total bytes received
	PacketsSent uint64 `json:"packets_sent"` // Total packets sent
	PacketsRecv uint64 `json:"packets_recv"` // Total packets received
	ErrorsIn    uint64 `json:"errors_in"`    // Input errors
	ErrorsOut   uint64 `json:"errors_out"`   // Output errors
	DropsIn     uint64 `json:"drops_in"`     // Dropped input packets
	DropsOut    uint64 `json:"drops_out"`    // Dropped output packets
}

// SystemInfo contains general system information
type SystemInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Uptime          uint64 `json:"uptime"` // System uptime in seconds
}

// ProcessMetrics contains process-specific metrics
type ProcessMetrics struct {
	Name        string  `json:"name"`
	PID         int32   `json:"pid"`
	Status      string  `json:"status"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    uint64  `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
}

// ContainerMetrics contains Docker container metrics
type ContainerMetrics struct {
	// Identity
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	ImageID string            `json:"image_id"`
	Labels  map[string]string `json:"labels,omitempty"`

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
