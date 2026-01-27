package server

import (
	"time"

	"github.com/anurag/saviour/pkg/metrics"
)

// ServerState represents the current state of an agent/server
type ServerState struct {
	AgentName     string    `json:"agent_name"`
	EC2InstanceID string    `json:"ec2_instance_id,omitempty"`
	LastSeen      time.Time `json:"last_seen"`
	Status        string    `json:"status"` // online, offline, degraded

	// Latest metrics
	SystemMetrics metrics.SystemMetrics `json:"system_metrics"`
	Containers    []ContainerState      `json:"containers,omitempty"`

	// Alert states
	ActiveAlerts []Alert `json:"active_alerts"`
}

// DiskMetrics represents disk metrics for a mount point
type DiskMetrics struct {
	MountPoint  string  `json:"mount_point"`
	UsedPercent float64 `json:"used_percent"`
}

// Clone creates a deep copy of ServerState to prevent data races
func (s *ServerState) Clone() *ServerState {
	if s == nil {
		return nil
	}

	clone := &ServerState{
		AgentName:     s.AgentName,
		EC2InstanceID: s.EC2InstanceID,
		LastSeen:      s.LastSeen,
		Status:        s.Status,
		SystemMetrics: s.SystemMetrics, // SystemMetrics contains primitives and can be copied
	}

	// Deep copy containers slice
	if len(s.Containers) > 0 {
		clone.Containers = make([]ContainerState, len(s.Containers))
		copy(clone.Containers, s.Containers)
	}

	// Deep copy active alerts slice
	if len(s.ActiveAlerts) > 0 {
		clone.ActiveAlerts = make([]Alert, len(s.ActiveAlerts))
		for i, alert := range s.ActiveAlerts {
			clone.ActiveAlerts[i] = alert
			// Deep copy the Details map if present
			if alert.Details != nil {
				clone.ActiveAlerts[i].Details = make(map[string]interface{})
				for k, v := range alert.Details {
					clone.ActiveAlerts[i].Details[k] = v
				}
			}
		}
	}

	return clone
}

// ContainerState tracks container state for change detection
type ContainerState struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Image           string    `json:"image"`
	State           string    `json:"state"`
	PreviousState   string    `json:"previous_state"`
	LastStateChange time.Time `json:"last_state_change"`
	RestartCount    int       `json:"restart_count"`
	AlertState      string    `json:"alert_state"` // ok, warning, critical
	Health          string    `json:"health"`
	CPUPercent      float64   `json:"cpu_percent"`
	MemoryPercent   float64   `json:"memory_percent"`
	MemoryUsage     uint64    `json:"memory_usage"`
	MemoryLimit     uint64    `json:"memory_limit"`
}

// Alert represents an active or historical alert
type Alert struct {
	ID          string                 `json:"id"`
	AgentName   string                 `json:"agent_name"`
	AlertType   string                 `json:"alert_type"`
	Severity    string                 `json:"severity"` // critical, warning, info
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"` // active, resolved, acknowledged
	NotifiedAt  *time.Time             `json:"notified_at,omitempty"`
}

// MetricsPushPayload is what agents send to the server
type MetricsPushPayload struct {
	AgentName     string                `json:"agent_name"`
	Timestamp     time.Time             `json:"timestamp"`
	EC2Metadata   *EC2Metadata          `json:"ec2_metadata,omitempty"`
	SystemMetrics metrics.SystemMetrics `json:"system_metrics"`
}

// EC2Metadata contains EC2 instance information
type EC2Metadata struct {
	InstanceID       string            `json:"instance_id"`
	InstanceType     string            `json:"instance_type"`
	Region           string            `json:"region"`
	AvailabilityZone string            `json:"availability_zone"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// HeartbeatPayload is a minimal payload for heartbeat checks
type HeartbeatPayload struct {
	AgentName string    `json:"agent_name"`
	Timestamp time.Time `json:"timestamp"`
}
