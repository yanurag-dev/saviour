package server

import (
	"sync"
	"time"
)

// StateStore manages the in-memory state of all agents
type StateStore struct {
	mu     sync.RWMutex
	agents map[string]*ServerState // key: agent_name
	alerts map[string]*Alert       // key: alert_id
}

// NewStateStore creates a new in-memory state store
func NewStateStore() *StateStore {
	return &StateStore{
		agents: make(map[string]*ServerState),
		alerts: make(map[string]*Alert),
	}
}

// UpdateAgent updates or creates agent state
func (s *StateStore) UpdateAgent(state *ServerState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.agents[state.AgentName]
	if exists {
		// Preserve previous container states for change detection
		state.Containers = s.mergeContainerStates(existing.Containers, state.Containers)
		
		// Preserve active alerts from previous state
		state.ActiveAlerts = existing.ActiveAlerts
	}

	// Update status based on last seen
	state.Status = "online"
	state.LastSeen = time.Now()

	s.agents[state.AgentName] = state
}

// mergeContainerStates merges previous and current container states
// to detect state changes
func (s *StateStore) mergeContainerStates(previous, current []ContainerState) []ContainerState {
	prevMap := make(map[string]ContainerState)
	for _, c := range previous {
		prevMap[c.ID] = c
	}

	merged := make([]ContainerState, 0, len(current))
	for _, curr := range current {
		if prev, exists := prevMap[curr.ID]; exists {
			// Check if state changed
			if curr.State != prev.State {
				curr.PreviousState = prev.State
				curr.LastStateChange = time.Now()
			} else {
				curr.PreviousState = prev.PreviousState
				curr.LastStateChange = prev.LastStateChange
			}
		} else {
			// New container
			curr.LastStateChange = time.Now()
		}
		merged = append(merged, curr)
	}

	return merged
}

// GetAgent retrieves agent state by name (returns a copy to prevent data races)
func (s *StateStore) GetAgent(agentName string) (*ServerState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.agents[agentName]
	if !exists {
		return nil, false
	}
	
	// Return a deep copy to prevent data races
	return state.Clone(), true
}

// GetAllAgents returns all agent states (returns copies to prevent data races)
func (s *StateStore) GetAllAgents() []*ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]*ServerState, 0, len(s.agents))
	for _, state := range s.agents {
		// Return deep copies to prevent data races
		states = append(states, state.Clone())
	}
	return states
}

// UpdateHeartbeat updates the last seen timestamp for an agent
func (s *StateStore) UpdateHeartbeat(agentName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.agents[agentName]
	if !exists {
		// Create minimal state for heartbeat-only agents
		state = &ServerState{
			AgentName: agentName,
			Status:    "online",
		}
		s.agents[agentName] = state
	}

	state.LastSeen = time.Now()
	state.Status = "online"
}

// CheckOfflineAgents marks agents as offline if they haven't sent heartbeat
func (s *StateStore) CheckOfflineAgents(timeout time.Duration) []*ServerState {
	s.mu.Lock()
	defer s.mu.Unlock()

	offline := make([]*ServerState, 0)
	now := time.Now()

	for _, state := range s.agents {
		if state.Status == "online" && now.Sub(state.LastSeen) > timeout {
			state.Status = "offline"
			// Return a deep copy to prevent data races
			offline = append(offline, state.Clone())
		}
	}

	return offline
}

// AddAlert adds a new alert to the store
func (s *StateStore) AddAlert(alert *Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alerts[alert.ID] = alert

	// Add to agent's active alerts
	if state, exists := s.agents[alert.AgentName]; exists {
		state.ActiveAlerts = append(state.ActiveAlerts, *alert)
	}
}

// ResolveAlert marks an alert as resolved
func (s *StateStore) ResolveAlert(alertID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if alert, exists := s.alerts[alertID]; exists {
		now := time.Now()
		alert.ResolvedAt = &now
		alert.Status = "resolved"

		// Remove from agent's active alerts
		if state, exists := s.agents[alert.AgentName]; exists {
			activeAlerts := make([]Alert, 0)
			for _, a := range state.ActiveAlerts {
				if a.ID != alertID {
					activeAlerts = append(activeAlerts, a)
				}
			}
			state.ActiveAlerts = activeAlerts
		}
	}
}

// GetActiveAlerts returns all active alerts (returns copies to prevent data races)
func (s *StateStore) GetActiveAlerts() []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := make([]*Alert, 0)
	for _, alert := range s.alerts {
		if alert.Status == "active" {
			// Return a deep copy to prevent data races
			alertCopy := *alert
			active = append(active, &alertCopy)
		}
	}
	return active
}

// GetAlertsByAgent returns all alerts for a specific agent (returns copies to prevent data races)
func (s *StateStore) GetAlertsByAgent(agentName string) []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*Alert, 0)
	for _, alert := range s.alerts {
		if alert.AgentName == agentName {
			// Return a deep copy to prevent data races
			alertCopy := *alert
			alerts = append(alerts, &alertCopy)
		}
	}
	return alerts
}

// GetAlert retrieves a specific alert by ID (returns a copy to prevent data races)
func (s *StateStore) GetAlert(alertID string) (*Alert, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, exists := s.alerts[alertID]
	if !exists {
		return nil, false
	}
	
	// Return a deep copy to prevent data races
	alertCopy := *alert
	return &alertCopy, true
}
