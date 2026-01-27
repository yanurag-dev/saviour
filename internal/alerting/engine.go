package alerting

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StateStore interface for accessing server state
type StateStore interface {
	GetAllAgents() []*ServerState
	CheckOfflineAgents(timeout time.Duration) []*ServerState
	AddAlert(alert *Alert)
}

// ServerState represents an agent's state (simplified interface)
type ServerState struct {
	AgentName     string
	Status        string
	LastSeen      time.Time
	SystemMetrics SystemMetrics
	Containers    []ContainerState
	ActiveAlerts  []Alert
}

// SystemMetrics holds system metrics (simplified interface)
type SystemMetrics struct {
	CPU    CPUMetrics
	Memory MemoryMetrics
	Disk   []DiskMetrics
}

// CPUMetrics holds CPU metrics
type CPUMetrics struct {
	UsagePercent float64
}

// MemoryMetrics holds memory metrics
type MemoryMetrics struct {
	UsedPercent float64
}

// DiskMetrics holds disk metrics
type DiskMetrics struct {
	MountPoint  string
	UsedPercent float64
}

// ContainerState holds container state
type ContainerState struct {
	ID             string
	Name           string
	State          string
	PreviousState  string
	Health         string
	CPUPercent     float64
	MemoryPercent  float64
	RestartCount   int
}

// Alert represents an alert
type Alert struct {
	ID          string
	AgentName   string
	AlertType   string
	Severity    string
	Message     string
	Details     map[string]interface{}
	TriggeredAt time.Time
	ResolvedAt  *time.Time
	Status      string
	NotifiedAt  *time.Time
}

// Config holds alerting configuration
type Config struct {
	Enabled               bool
	CheckInterval         time.Duration
	HeartbeatTimeout      time.Duration
	DeduplicationEnabled  bool
	DeduplicationWindow   time.Duration
	SystemCPUThreshold    float64
	SystemMemoryThreshold float64
	SystemDiskThreshold   float64
}

// Notifier interface for sending notifications
type Notifier interface {
	SendAlert(alert *Alert) error
}

// Engine handles alert detection and management
type Engine struct {
	state        StateStore
	config       *Config
	notifier     Notifier
	mu           sync.RWMutex
	recentAlerts map[string]time.Time // For deduplication: alertKey -> lastSent
}

// NewEngine creates a new alert detection engine
func NewEngine(state StateStore, config *Config, notifier Notifier) *Engine {
	return &Engine{
		state:        state,
		config:       config,
		notifier:     notifier,
		recentAlerts: make(map[string]time.Time),
	}
}

// Start begins the alert detection loop
func (e *Engine) Start() {
	if !e.config.Enabled {
		log.Println("Alert engine disabled")
		return
	}

	// Validate check interval to prevent panic in time.NewTicker
	checkInterval := e.config.CheckInterval
	if checkInterval <= 0 {
		log.Printf("Warning: Invalid check interval (%v), using default 30s", checkInterval)
		checkInterval = 30 * time.Second
		e.config.CheckInterval = checkInterval
	}

	log.Printf("Starting alert engine (check interval: %v)", checkInterval)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		e.checkAlerts()
	}
}

// checkAlerts performs all alert checks
func (e *Engine) checkAlerts() {
	// Check for offline agents
	e.checkOfflineAgents()

	// Check system and container metrics for all agents
	agents := e.state.GetAllAgents()
	for _, agent := range agents {
		if agent.Status == "online" {
			e.checkSystemAlerts(agent)
			e.checkContainerAlerts(agent)
		}
	}

	// Cleanup old deduplication entries
	e.cleanupDeduplication()
}

// checkOfflineAgents checks for agents that haven't sent heartbeat
func (e *Engine) checkOfflineAgents() {
	offline := e.state.CheckOfflineAgents(e.config.HeartbeatTimeout)

	for _, agent := range offline {
		alertKey := fmt.Sprintf("agent_offline:%s", agent.AgentName)
		if e.shouldSendAlert(alertKey) {
			alert := &Alert{
				ID:          uuid.New().String(),
				AgentName:   agent.AgentName,
				AlertType:   "agent_offline",
				Severity:    "critical",
				Message:     fmt.Sprintf("🔴 Agent Offline\nAgent: %s\nLast Seen: %s", agent.AgentName, agent.LastSeen.Format(time.RFC3339)),
				Details: map[string]interface{}{
					"agent_name": agent.AgentName,
					"last_seen":  agent.LastSeen,
				},
				TriggeredAt: time.Now(),
				Status:      "active",
			}

			e.state.AddAlert(alert)
			if err := e.notifier.SendAlert(alert); err != nil {
				log.Printf("Failed to send alert: %v", err)
			} else {
				now := time.Now()
				alert.NotifiedAt = &now
				e.markAlertSent(alertKey)
			}
		}
	}
}

// checkSystemAlerts checks system-level thresholds
func (e *Engine) checkSystemAlerts(agent *ServerState) {
	// CPU alert
	if e.config.SystemCPUThreshold > 0 && agent.SystemMetrics.CPU.UsagePercent > e.config.SystemCPUThreshold {
		alertKey := fmt.Sprintf("system_cpu:%s", agent.AgentName)
		if e.shouldSendAlert(alertKey) {
			alert := &Alert{
				ID:        uuid.New().String(),
				AgentName: agent.AgentName,
				AlertType: "system_cpu_high",
				Severity:  "warning",
				Message:   fmt.Sprintf("⚠️ High CPU Usage\nAgent: %s\nCPU: %.1f%%", agent.AgentName, agent.SystemMetrics.CPU.UsagePercent),
				Details: map[string]interface{}{
					"agent_name":  agent.AgentName,
					"cpu_percent": agent.SystemMetrics.CPU.UsagePercent,
				},
				TriggeredAt: time.Now(),
				Status:      "active",
			}
			e.sendAlert(alert, alertKey)
		}
	}

	// Memory alert
	if e.config.SystemMemoryThreshold > 0 && agent.SystemMetrics.Memory.UsedPercent > e.config.SystemMemoryThreshold {
		alertKey := fmt.Sprintf("system_memory:%s", agent.AgentName)
		if e.shouldSendAlert(alertKey) {
			alert := &Alert{
				ID:        uuid.New().String(),
				AgentName: agent.AgentName,
				AlertType: "system_memory_high",
				Severity:  "warning",
				Message:   fmt.Sprintf("⚠️ High Memory Usage\nAgent: %s\nMemory: %.1f%%", agent.AgentName, agent.SystemMetrics.Memory.UsedPercent),
				Details: map[string]interface{}{
					"agent_name":     agent.AgentName,
					"memory_percent": agent.SystemMetrics.Memory.UsedPercent,
				},
				TriggeredAt: time.Now(),
				Status:      "active",
			}
			e.sendAlert(alert, alertKey)
		}
	}

	// Disk alert
	for _, disk := range agent.SystemMetrics.Disk {
		if e.config.SystemDiskThreshold > 0 && disk.UsedPercent > e.config.SystemDiskThreshold {
			alertKey := fmt.Sprintf("system_disk:%s:%s", agent.AgentName, disk.MountPoint)
			if e.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        uuid.New().String(),
					AgentName: agent.AgentName,
					AlertType: "system_disk_high",
					Severity:  "critical",
					Message:   fmt.Sprintf("🚨 High Disk Usage\nAgent: %s\nMount: %s\nUsage: %.1f%%", agent.AgentName, disk.MountPoint, disk.UsedPercent),
					Details: map[string]interface{}{
						"agent_name":   agent.AgentName,
						"mount_point":  disk.MountPoint,
						"disk_percent": disk.UsedPercent,
					},
					TriggeredAt: time.Now(),
					Status:      "active",
				}
				e.sendAlert(alert, alertKey)
			}
		}
	}
}

// checkContainerAlerts checks container-specific alerts
func (e *Engine) checkContainerAlerts(agent *ServerState) {
	for _, container := range agent.Containers {
		// Container stopped
		if container.PreviousState == "running" && (container.State == "exited" || container.State == "dead") {
			alertKey := fmt.Sprintf("container_stopped:%s:%s", agent.AgentName, container.ID)
			if e.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        uuid.New().String(),
					AgentName: agent.AgentName,
					AlertType: "container_stopped",
					Severity:  "critical",
					Message:   fmt.Sprintf("💀 Container Stopped\nAgent: %s\nContainer: %s\nState: %s", agent.AgentName, container.Name, container.State),
					Details: map[string]interface{}{
						"agent_name":     agent.AgentName,
						"container_id":   container.ID,
						"container_name": container.Name,
						"state":          container.State,
						"previous_state": container.PreviousState,
					},
					TriggeredAt: time.Now(),
					Status:      "active",
				}
				e.sendAlert(alert, alertKey)
			}
		}

		// Container unhealthy
		if container.Health == "unhealthy" {
			alertKey := fmt.Sprintf("container_unhealthy:%s:%s", agent.AgentName, container.ID)
			if e.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        uuid.New().String(),
					AgentName: agent.AgentName,
					AlertType: "container_unhealthy",
					Severity:  "warning",
					Message:   fmt.Sprintf("🏥 Container Unhealthy\nAgent: %s\nContainer: %s", agent.AgentName, container.Name),
					Details: map[string]interface{}{
						"agent_name":     agent.AgentName,
						"container_id":   container.ID,
						"container_name": container.Name,
						"health":         container.Health,
					},
					TriggeredAt: time.Now(),
					Status:      "active",
				}
				e.sendAlert(alert, alertKey)
			}
		}

		// Container high CPU
		if container.CPUPercent > 90.0 {
			alertKey := fmt.Sprintf("container_cpu:%s:%s", agent.AgentName, container.ID)
			if e.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        uuid.New().String(),
					AgentName: agent.AgentName,
					AlertType: "container_cpu_high",
					Severity:  "warning",
					Message:   fmt.Sprintf("⚠️ Container High CPU\nAgent: %s\nContainer: %s\nCPU: %.1f%%", agent.AgentName, container.Name, container.CPUPercent),
					Details: map[string]interface{}{
						"agent_name":     agent.AgentName,
						"container_id":   container.ID,
						"container_name": container.Name,
						"cpu_percent":    container.CPUPercent,
					},
					TriggeredAt: time.Now(),
					Status:      "active",
				}
				e.sendAlert(alert, alertKey)
			}
		}

		// Container high memory
		if container.MemoryPercent > 95.0 {
			alertKey := fmt.Sprintf("container_memory:%s:%s", agent.AgentName, container.ID)
			if e.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        uuid.New().String(),
					AgentName: agent.AgentName,
					AlertType: "container_memory_high",
					Severity:  "critical",
					Message:   fmt.Sprintf("🚨 Container High Memory\nAgent: %s\nContainer: %s\nMemory: %.1f%%", agent.AgentName, container.Name, container.MemoryPercent),
					Details: map[string]interface{}{
						"agent_name":     agent.AgentName,
						"container_id":   container.ID,
						"container_name": container.Name,
						"memory_percent": container.MemoryPercent,
					},
					TriggeredAt: time.Now(),
					Status:      "active",
				}
				e.sendAlert(alert, alertKey)
			}
		}
	}
}

// shouldSendAlert checks if alert should be sent based on deduplication
func (e *Engine) shouldSendAlert(alertKey string) bool {
	if !e.config.DeduplicationEnabled {
		return true
	}

	e.mu.RLock()
	lastSent, exists := e.recentAlerts[alertKey]
	e.mu.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastSent) > e.config.DeduplicationWindow
}

// markAlertSent marks an alert as sent for deduplication
func (e *Engine) markAlertSent(alertKey string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.recentAlerts[alertKey] = time.Now()
}

// sendAlert sends an alert and updates state
func (e *Engine) sendAlert(alert *Alert, alertKey string) {
	e.state.AddAlert(alert)
	if err := e.notifier.SendAlert(alert); err != nil {
		log.Printf("Failed to send alert: %v", err)
	} else {
		now := time.Now()
		alert.NotifiedAt = &now
		e.markAlertSent(alertKey)
		log.Printf("Alert sent: %s - %s", alert.AlertType, alert.AgentName)
	}
}

// cleanupDeduplication removes old deduplication entries
func (e *Engine) cleanupDeduplication() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	for key, lastSent := range e.recentAlerts {
		if now.Sub(lastSent) > e.config.DeduplicationWindow*2 {
			delete(e.recentAlerts, key)
		}
	}
}
