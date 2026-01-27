package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/anurag/saviour/internal/collector"
	"github.com/anurag/saviour/internal/config"
	"github.com/anurag/saviour/internal/docker"
	"github.com/anurag/saviour/pkg/metrics"
)

// Agent represents the monitoring agent
type Agent struct {
	config          *config.Config
	systemCollector *collector.SystemCollector
	dockerCollector *collector.DockerCollector
	sender          *Sender
	logger          *log.Logger
	lastMetrics     *metrics.SystemMetrics // Store last collected metrics for push
}

// New creates a new agent instance
func New(cfg *config.Config, logger *log.Logger) (*Agent, error) {
	agent := &Agent{
		config:          cfg,
		systemCollector: collector.NewSystemCollector(cfg.Agent.Name, cfg.Metrics.DiskMounts),
		logger:          logger,
	}

	// Initialize Docker collector if enabled
	if cfg.Metrics.Docker.Enabled {
		filterConfig := docker.FilterConfig{
			MonitorAll: cfg.Metrics.Docker.MonitorAll,
			Labels:     cfg.Metrics.Docker.Filters.Labels,
			Names:      cfg.Metrics.Docker.Filters.Names,
			Images:     cfg.Metrics.Docker.Filters.Images,
		}

		dockerCollector, err := collector.NewDockerCollector(
			cfg.Metrics.Docker.Socket,
			filterConfig,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Docker collector: %w", err)
		}
		agent.dockerCollector = dockerCollector
		logger.Println("✓ Docker monitoring enabled")
	}

	// Initialize sender if server URL is configured
	if cfg.Agent.ServerURL != "" {
		agent.sender = NewSender(cfg.Agent.ServerURL, cfg.Agent.APIKey)
		logger.Printf("✓ Server push enabled: %s", cfg.Agent.ServerURL)
	} else {
		logger.Println("⚠️  No server URL configured - metrics will only be logged locally")
	}

	return agent, nil
}

// Run starts the agent's main loop
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Printf("Agent '%s' starting...", a.config.Agent.Name)
	a.logger.Printf("Collection interval: %v", a.config.Agent.CollectInterval)

	// Collection ticker
	collectTicker := time.NewTicker(a.config.Agent.CollectInterval)
	defer collectTicker.Stop()

	// Push ticker (if server configured)
	var pushTicker *time.Ticker
	if a.sender != nil {
		pushTicker = time.NewTicker(a.config.Agent.PushInterval)
		defer pushTicker.Stop()
		a.logger.Printf("Push interval: %v", a.config.Agent.PushInterval)
	}

	// Heartbeat ticker (if server configured)
	var heartbeatTicker *time.Ticker
	if a.sender != nil {
		heartbeatTicker = time.NewTicker(a.config.Agent.HeartbeatInterval)
		defer heartbeatTicker.Stop()
		a.logger.Printf("Heartbeat interval: %v", a.config.Agent.HeartbeatInterval)
	}

	// Collect immediately on start
	if err := a.collectAndProcess(); err != nil {
		a.logger.Printf("Error during initial collection: %v", err)
	}

	// Main loop
	for {
		select {
		case <-ctx.Done():
			a.logger.Println("Agent shutting down...")
			return ctx.Err()

		case <-collectTicker.C:
			if err := a.collectAndProcess(); err != nil {
				a.logger.Printf("Error collecting metrics: %v", err)
			}

		case <-func() <-chan time.Time {
			if pushTicker != nil {
				return pushTicker.C
			}
			return make(chan time.Time) // Never fires
		}():
			if a.lastMetrics != nil {
				if err := a.pushMetrics(ctx); err != nil {
					a.logger.Printf("Error pushing metrics: %v", err)
				} else {
					a.logger.Println("✓ Metrics pushed to server")
				}
			}

		case <-func() <-chan time.Time {
			if heartbeatTicker != nil {
				return heartbeatTicker.C
			}
			return make(chan time.Time) // Never fires
		}():
			if err := a.sendHeartbeat(ctx); err != nil {
				a.logger.Printf("Error sending heartbeat: %v", err)
			} else {
				a.logger.Println("♥ Heartbeat sent")
			}
		}
	}
}

// pushMetrics sends the last collected metrics to the server
func (a *Agent) pushMetrics(ctx context.Context) error {
	if a.sender == nil {
		return nil
	}
	return a.sender.PushMetrics(ctx, a.lastMetrics)
}

// sendHeartbeat sends a heartbeat to the server
func (a *Agent) sendHeartbeat(ctx context.Context) error {
	if a.sender == nil {
		return nil
	}
	return a.sender.SendHeartbeat(ctx, a.config.Agent.Name)
}

func (a *Agent) collectAndProcess() error {
	ctx := context.Background()
	
	// Collect system metrics
	m, err := a.systemCollector.Collect()
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// Collect Docker container metrics if enabled
	if a.dockerCollector != nil {
		containers, err := a.dockerCollector.Collect(ctx)
		if err != nil {
			a.logger.Printf("Warning: Docker collection failed: %v", err)
		} else {
			// Convert docker.ContainerInfo to metrics.ContainerMetrics
			m.Containers = make([]metrics.ContainerMetrics, len(containers))
			for i, c := range containers {
				m.Containers[i] = metrics.ContainerMetrics{
					ID:              c.ID,
					Name:            c.Name,
					Image:           c.Image,
					ImageID:         c.ImageID,
					Labels:          c.Labels,
					State:           c.State,
					Status:          c.Status,
					Health:          c.Health,
					ExitCode:        c.ExitCode,
					OOMKilled:       c.OOMKilled,
					RestartCount:    c.RestartCount,
					Created:         c.Created,
					StartedAt:       c.StartedAt,
					FinishedAt:      c.FinishedAt,
					CPUPercent:      c.CPUPercent,
					MemoryUsage:     c.MemoryUsage,
					MemoryLimit:     c.MemoryLimit,
					MemoryPercent:   c.MemoryPercent,
					NetworkRxBytes:  c.NetworkRxBytes,
					NetworkTxBytes:  c.NetworkTxBytes,
					BlockReadBytes:  c.BlockReadBytes,
					BlockWriteBytes: c.BlockWriteBytes,
					PIDs:            c.PIDs,
				}
			}
		}
	}

	// Store metrics for push
	a.lastMetrics = m

	// Process and log metrics
	if err := a.processMetrics(m); err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	return nil
}

func (a *Agent) processMetrics(m *metrics.SystemMetrics) error {
	// Check alert thresholds
	a.checkAlerts(m)

	// Output metrics (for now, just pretty print)
	// In production, this would send to the central server
	a.logMetrics(m)

	return nil
}

func (a *Agent) checkAlerts(m *metrics.SystemMetrics) {
	// System alerts
	if m.CPU.UsagePercent > a.config.Alerts.CPUThreshold {
		a.logger.Printf("⚠️  ALERT: CPU usage (%.2f%%) exceeds threshold (%.2f%%)",
			m.CPU.UsagePercent, a.config.Alerts.CPUThreshold)
	}

	if m.Memory.UsedPercent > a.config.Alerts.MemoryThreshold {
		a.logger.Printf("⚠️  ALERT: Memory usage (%.2f%%) exceeds threshold (%.2f%%)",
			m.Memory.UsedPercent, a.config.Alerts.MemoryThreshold)
	}

	for _, disk := range m.Disk {
		if disk.UsedPercent > a.config.Alerts.DiskThreshold {
			a.logger.Printf("⚠️  ALERT: Disk usage on %s (%.2f%%) exceeds threshold (%.2f%%)",
				disk.MountPoint, disk.UsedPercent, a.config.Alerts.DiskThreshold)
		}
	}

	// Container alerts
	if a.dockerCollector != nil {
		a.checkContainerAlerts(m.Containers)
	}
}

func (a *Agent) checkContainerAlerts(containers []metrics.ContainerMetrics) {
	defaultThreshold := a.config.Metrics.Docker.Alerts.Default

	for _, container := range containers {
		// Get threshold for this container (check overrides)
		cpuThreshold := defaultThreshold.CPUThreshold
		memThreshold := defaultThreshold.MemoryThreshold
		restartThreshold := defaultThreshold.RestartThreshold

		for _, override := range a.config.Metrics.Docker.Alerts.Overrides {
			// Support glob-style pattern matching (e.g., "api-*", "worker-*")
			matched := false
			if strings.Contains(override.Name, "*") || strings.Contains(override.Name, "?") {
				// Use glob pattern matching
				if match, err := filepath.Match(override.Name, container.Name); err == nil && match {
					matched = true
				}
			} else {
				// Exact match
				matched = (container.Name == override.Name)
			}

			if matched {
				if override.CPUThreshold > 0 {
					cpuThreshold = override.CPUThreshold
				}
				if override.MemoryThreshold > 0 {
					memThreshold = override.MemoryThreshold
				}
				if override.RestartThreshold > 0 {
					restartThreshold = override.RestartThreshold
				}
				// Don't break - allow multiple patterns to match and last one wins
			}
		}

		// Container state alerts
		if container.State == "exited" {
			a.logger.Printf("💀 ALERT: Container '%s' stopped (exit code: %d)",
				container.Name, container.ExitCode)
		}

		if container.Health == "unhealthy" {
			a.logger.Printf("🏥 ALERT: Container '%s' is unhealthy",
				container.Name)
		}

		if container.OOMKilled {
			a.logger.Printf("💥 ALERT: Container '%s' was OOM killed",
				container.Name)
		}

		// Resource alerts (only for running containers)
		if container.State == "running" {
			if container.CPUPercent > cpuThreshold {
				a.logger.Printf("⚠️  ALERT: Container '%s' CPU (%.2f%%) exceeds threshold (%.2f%%)",
					container.Name, container.CPUPercent, cpuThreshold)
			}

			if container.MemoryPercent > memThreshold {
				a.logger.Printf("⚠️  ALERT: Container '%s' memory (%.2f%%) exceeds threshold (%.2f%%)",
					container.Name, container.MemoryPercent, memThreshold)
			}
		}

		// Restart count alert
		if container.RestartCount > restartThreshold {
			a.logger.Printf("🔄 ALERT: Container '%s' restart count (%d) exceeds threshold (%d)",
				container.Name, container.RestartCount, restartThreshold)
		}
	}
}

func (a *Agent) logMetrics(m *metrics.SystemMetrics) {
	a.logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	a.logger.Printf("📊 Metrics collected at %s", m.Timestamp.Format(time.RFC3339))
	a.logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// System Info
	a.logger.Printf("🖥️  System: %s (%s %s)", m.SystemInfo.Hostname, m.SystemInfo.OS, m.SystemInfo.Platform)
	a.logger.Printf("   Uptime: %s", formatDuration(time.Duration(m.SystemInfo.Uptime)*time.Second))

	// CPU
	a.logger.Printf("💻 CPU Usage: %.2f%%", m.CPU.UsagePercent)
	a.logger.Printf("   Load Avg: %.2f (1m) | %.2f (5m) | %.2f (15m)",
		m.CPU.LoadAvg1, m.CPU.LoadAvg5, m.CPU.LoadAvg15)

	// Memory
	a.logger.Printf("🧠 Memory: %.2f%% used (%s / %s)",
		m.Memory.UsedPercent,
		formatBytes(m.Memory.Used),
		formatBytes(m.Memory.Total))
	if m.Memory.SwapTotal > 0 {
		a.logger.Printf("   Swap: %.2f%% used (%s / %s)",
			m.Memory.SwapPercent,
			formatBytes(m.Memory.SwapUsed),
			formatBytes(m.Memory.SwapTotal))
	}

	// Disk
	a.logger.Println("💾 Disk Usage:")
	for _, disk := range m.Disk {
		a.logger.Printf("   %s: %.2f%% used (%s / %s)",
			disk.MountPoint,
			disk.UsedPercent,
			formatBytes(disk.Used),
			formatBytes(disk.Total))
	}

	// Network
	a.logger.Printf("🌐 Network: ↑ %s sent | ↓ %s received",
		formatBytes(m.Network.BytesSent),
		formatBytes(m.Network.BytesRecv))

	// Docker containers
	if len(m.Containers) > 0 {
		a.logger.Printf("🐳 Containers: %d monitored", len(m.Containers))
		for _, container := range m.Containers {
			statusIcon := getContainerStatusIcon(container.State, container.Health)
			if container.State == "running" {
				a.logger.Printf("   %s %s: CPU %.1f%% | Mem %s (%.1f%%) | Restarts: %d",
					statusIcon,
					container.Name,
					container.CPUPercent,
					formatBytes(container.MemoryUsage),
					container.MemoryPercent,
					container.RestartCount)
			} else {
				a.logger.Printf("   %s %s: %s (exit code: %d)",
					statusIcon,
					container.Name,
					container.State,
					container.ExitCode)
			}
		}
	}

	// Output JSON for debugging
	if a.config.Agent.Name != "" {
		jsonData, _ := json.MarshalIndent(m, "", "  ")
		a.logger.Printf("\n📄 JSON Output:\n%s\n", string(jsonData))
	}
}

func getContainerStatusIcon(state, health string) string {
	if state == "running" {
		if health == "healthy" {
			return "🟢"
		} else if health == "unhealthy" {
			return "🔴"
		} else if health == "starting" {
			return "🟡"
		}
		return "🟢" // No health check defined
	} else if state == "exited" {
		return "⚫"
	} else if state == "restarting" {
		return "🔄"
	} else if state == "paused" {
		return "⏸️"
	}
	return "⚪"
}

// Helper functions

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
