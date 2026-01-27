package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/anurag/saviour/internal/collector"
	"github.com/anurag/saviour/internal/config"
	"github.com/anurag/saviour/pkg/metrics"
)

// Agent represents the monitoring agent
type Agent struct {
	config    *config.Config
	collector *collector.SystemCollector
	logger    *log.Logger
}

// New creates a new agent instance
func New(cfg *config.Config, logger *log.Logger) *Agent {
	return &Agent{
		config:    cfg,
		collector: collector.NewSystemCollector(cfg.Agent.Name, cfg.Metrics.DiskMounts),
		logger:    logger,
	}
}

// Run starts the agent's main loop
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Printf("Agent '%s' starting...", a.config.Agent.Name)
	a.logger.Printf("Collection interval: %v", a.config.Agent.CollectInterval)

	ticker := time.NewTicker(a.config.Agent.CollectInterval)
	defer ticker.Stop()

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
		case <-ticker.C:
			if err := a.collectAndProcess(); err != nil {
				a.logger.Printf("Error collecting metrics: %v", err)
			}
		}
	}
}

func (a *Agent) collectAndProcess() error {
	// Collect metrics
	m, err := a.collector.Collect()
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// For now, just log the metrics (later we'll send to server)
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
	// CPU threshold
	if m.CPU.UsagePercent > a.config.Alerts.CPUThreshold {
		a.logger.Printf("⚠️  ALERT: CPU usage (%.2f%%) exceeds threshold (%.2f%%)",
			m.CPU.UsagePercent, a.config.Alerts.CPUThreshold)
	}

	// Memory threshold
	if m.Memory.UsedPercent > a.config.Alerts.MemoryThreshold {
		a.logger.Printf("⚠️  ALERT: Memory usage (%.2f%%) exceeds threshold (%.2f%%)",
			m.Memory.UsedPercent, a.config.Alerts.MemoryThreshold)
	}

	// Disk threshold
	for _, disk := range m.Disk {
		if disk.UsedPercent > a.config.Alerts.DiskThreshold {
			a.logger.Printf("⚠️  ALERT: Disk usage on %s (%.2f%%) exceeds threshold (%.2f%%)",
				disk.MountPoint, disk.UsedPercent, a.config.Alerts.DiskThreshold)
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

	// Output JSON for debugging
	if a.config.Agent.Name != "" {
		jsonData, _ := json.MarshalIndent(m, "", "  ")
		a.logger.Printf("\n📄 JSON Output:\n%s\n", string(jsonData))
	}
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
