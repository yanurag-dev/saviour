package server

import (
	"time"

	"github.com/anurag/saviour/internal/alerting"
	"github.com/anurag/saviour/pkg/metrics"
)

// AlertingAdapter adapts ServerState to alerting.ServerState
type AlertingAdapter struct {
	store *StateStore
}

// NewAlertingAdapter creates a new adapter
func NewAlertingAdapter(store *StateStore) *AlertingAdapter {
	return &AlertingAdapter{store: store}
}

// GetAllAgents returns all agents in alerting format
func (a *AlertingAdapter) GetAllAgents() []*alerting.ServerState {
	agents := a.store.GetAllAgents()
	result := make([]*alerting.ServerState, len(agents))

	for i, agent := range agents {
		result[i] = a.convertServerState(agent)
	}

	return result
}

// CheckOfflineAgents checks for offline agents
func (a *AlertingAdapter) CheckOfflineAgents(timeout time.Duration) []*alerting.ServerState {
	offline := a.store.CheckOfflineAgents(timeout)
	result := make([]*alerting.ServerState, len(offline))

	for i, agent := range offline {
		result[i] = a.convertServerState(agent)
	}

	return result
}

// AddAlert adds an alert
func (a *AlertingAdapter) AddAlert(alert *alerting.Alert) {
	serverAlert := &Alert{
		ID:          alert.ID,
		AgentName:   alert.AgentName,
		AlertType:   alert.AlertType,
		Severity:    alert.Severity,
		Message:     alert.Message,
		Details:     alert.Details,
		TriggeredAt: alert.TriggeredAt,
		ResolvedAt:  alert.ResolvedAt,
		Status:      alert.Status,
		NotifiedAt:  alert.NotifiedAt,
	}
	a.store.AddAlert(serverAlert)
}

// convertServerState converts server.ServerState to alerting.ServerState
func (a *AlertingAdapter) convertServerState(state *ServerState) *alerting.ServerState {
	containers := make([]alerting.ContainerState, len(state.Containers))
	for i, c := range state.Containers {
		containers[i] = alerting.ContainerState{
			ID:            c.ID,
			Name:          c.Name,
			State:         c.State,
			PreviousState: c.PreviousState,
			Health:        c.Health,
			CPUPercent:    c.CPUPercent,
			MemoryPercent: c.MemoryPercent,
			RestartCount:  c.RestartCount,
		}
	}

	alerts := make([]alerting.Alert, len(state.ActiveAlerts))
	for i, a := range state.ActiveAlerts {
		alerts[i] = alerting.Alert{
			ID:          a.ID,
			AgentName:   a.AgentName,
			AlertType:   a.AlertType,
			Severity:    a.Severity,
			Message:     a.Message,
			Details:     a.Details,
			TriggeredAt: a.TriggeredAt,
			ResolvedAt:  a.ResolvedAt,
			Status:      a.Status,
			NotifiedAt:  a.NotifiedAt,
		}
	}

	return &alerting.ServerState{
		AgentName: state.AgentName,
		Status:    state.Status,
		LastSeen:  state.LastSeen,
		SystemMetrics: alerting.SystemMetrics{
			CPU: alerting.CPUMetrics{
				UsagePercent: state.SystemMetrics.CPU.UsagePercent,
			},
			Memory: alerting.MemoryMetrics{
				UsedPercent: state.SystemMetrics.Memory.UsedPercent,
			},
			Disk: a.convertDiskMetrics(state.SystemMetrics.Disk),
		},
		Containers:   containers,
		ActiveAlerts: alerts,
	}
}

// convertDiskMetrics converts disk metrics from metrics package
func (a *AlertingAdapter) convertDiskMetrics(disks []metrics.DiskMetrics) []alerting.DiskMetrics {
	result := make([]alerting.DiskMetrics, len(disks))
	for i, d := range disks {
		result[i] = alerting.DiskMetrics{
			MountPoint:  d.MountPoint,
			UsedPercent: d.UsedPercent,
		}
	}
	return result
}
