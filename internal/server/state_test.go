package server

import (
	"sync"
	"testing"
	"time"

	"github.com/anurag/saviour/pkg/metrics"
)

func TestNewStateStore(t *testing.T) {
	store := NewStateStore()

	if store == nil {
		t.Fatal("NewStateStore returned nil")
	}

	if store.agents == nil {
		t.Error("agents map not initialized")
	}

	if store.alerts == nil {
		t.Error("alerts map not initialized")
	}

	if len(store.agents) != 0 {
		t.Errorf("agents map should be empty, got %d entries", len(store.agents))
	}

	if len(store.alerts) != 0 {
		t.Errorf("alerts map should be empty, got %d entries", len(store.alerts))
	}
}

func TestUpdateAgent_NewAgent(t *testing.T) {
	store := NewStateStore()

	state := &ServerState{
		AgentName:     "test-agent",
		EC2InstanceID: "i-12345",
		SystemMetrics: metrics.SystemMetrics{
			CPU: metrics.CPUMetrics{
				UsagePercent: 50.0,
			},
		},
	}

	store.UpdateAgent(state)

	retrieved, exists := store.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found after update")
	}

	if retrieved.AgentName != "test-agent" {
		t.Errorf("AgentName = %v, want %v", retrieved.AgentName, "test-agent")
	}

	if retrieved.Status != "online" {
		t.Errorf("Status = %v, want online", retrieved.Status)
	}

	if retrieved.LastSeen.IsZero() {
		t.Error("LastSeen should be set")
	}

	if retrieved.SystemMetrics.CPU.UsagePercent != 50.0 {
		t.Errorf("CPU.UsagePercent = %v, want 50.0", retrieved.SystemMetrics.CPU.UsagePercent)
	}
}

func TestUpdateAgent_ExistingAgent(t *testing.T) {
	store := NewStateStore()

	// First update
	state1 := &ServerState{
		AgentName: "test-agent",
		Containers: []ContainerState{
			{ID: "c1", Name: "container1", State: "running"},
		},
	}
	store.UpdateAgent(state1)

	// Second update with new data
	state2 := &ServerState{
		AgentName: "test-agent",
		Containers: []ContainerState{
			{ID: "c1", Name: "container1", State: "exited"},
			{ID: "c2", Name: "container2", State: "running"},
		},
	}
	store.UpdateAgent(state2)

	retrieved, exists := store.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found after second update")
	}

	if len(retrieved.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(retrieved.Containers))
	}

	// Check state change detection
	c1 := retrieved.Containers[0]
	if c1.State != "exited" {
		t.Errorf("Container state = %v, want exited", c1.State)
	}

	if c1.PreviousState != "running" {
		t.Errorf("PreviousState = %v, want running", c1.PreviousState)
	}

	if c1.LastStateChange.IsZero() {
		t.Error("LastStateChange should be set for state change")
	}
}

func TestUpdateAgent_PreservesActiveAlerts(t *testing.T) {
	store := NewStateStore()

	// First update with alert
	state1 := &ServerState{
		AgentName: "test-agent",
		ActiveAlerts: []Alert{
			{ID: "alert1", AlertType: "high_cpu"},
		},
	}
	store.UpdateAgent(state1)

	// Second update without alerts
	state2 := &ServerState{
		AgentName: "test-agent",
	}
	store.UpdateAgent(state2)

	retrieved, _ := store.GetAgent("test-agent")

	if len(retrieved.ActiveAlerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(retrieved.ActiveAlerts))
	}

	if retrieved.ActiveAlerts[0].ID != "alert1" {
		t.Error("Active alerts not preserved across updates")
	}
}

func TestMergeContainerStates_NewContainer(t *testing.T) {
	store := NewStateStore()

	previous := []ContainerState{}
	current := []ContainerState{
		{ID: "c1", Name: "container1", State: "running"},
	}

	merged := store.mergeContainerStates(previous, current)

	if len(merged) != 1 {
		t.Errorf("Expected 1 merged container, got %d", len(merged))
	}

	if merged[0].LastStateChange.IsZero() {
		t.Error("New container should have LastStateChange set")
	}
}

func TestMergeContainerStates_StateChanged(t *testing.T) {
	store := NewStateStore()

	baseTime := time.Now().Add(-1 * time.Minute)
	previous := []ContainerState{
		{ID: "c1", Name: "container1", State: "running", LastStateChange: baseTime},
	}
	current := []ContainerState{
		{ID: "c1", Name: "container1", State: "exited"},
	}

	merged := store.mergeContainerStates(previous, current)

	if merged[0].State != "exited" {
		t.Errorf("State = %v, want exited", merged[0].State)
	}

	if merged[0].PreviousState != "running" {
		t.Errorf("PreviousState = %v, want running", merged[0].PreviousState)
	}

	if merged[0].LastStateChange == baseTime {
		t.Error("LastStateChange should be updated on state change")
	}
}

func TestMergeContainerStates_StateUnchanged(t *testing.T) {
	store := NewStateStore()

	baseTime := time.Now().Add(-1 * time.Minute)
	previous := []ContainerState{
		{
			ID:              "c1",
			Name:            "container1",
			State:           "running",
			PreviousState:   "created",
			LastStateChange: baseTime,
		},
	}
	current := []ContainerState{
		{ID: "c1", Name: "container1", State: "running"},
	}

	merged := store.mergeContainerStates(previous, current)

	if merged[0].PreviousState != "created" {
		t.Errorf("PreviousState = %v, want created", merged[0].PreviousState)
	}

	if merged[0].LastStateChange != baseTime {
		t.Error("LastStateChange should be preserved when state unchanged")
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	store := NewStateStore()

	_, exists := store.GetAgent("nonexistent")

	if exists {
		t.Error("Expected agent not to exist")
	}
}

func TestGetAgent_ReturnsDeepCopy(t *testing.T) {
	store := NewStateStore()

	state := &ServerState{
		AgentName: "test-agent",
		Containers: []ContainerState{
			{ID: "c1", Name: "container1", State: "running"},
		},
		ActiveAlerts: []Alert{
			{ID: "alert1", AlertType: "high_cpu", Details: map[string]interface{}{"cpu": 90.0}},
		},
	}
	store.UpdateAgent(state)

	// Get agent and modify returned copy
	retrieved, _ := store.GetAgent("test-agent")
	retrieved.AgentName = "modified"
	retrieved.Containers[0].State = "exited"
	retrieved.ActiveAlerts[0].ID = "modified"
	retrieved.ActiveAlerts[0].Details["cpu"] = 100.0

	// Get again and verify original unchanged
	original, _ := store.GetAgent("test-agent")

	if original.AgentName != "test-agent" {
		t.Error("AgentName was modified in store")
	}

	if original.Containers[0].State != "running" {
		t.Error("Container state was modified in store")
	}

	if original.ActiveAlerts[0].ID != "alert1" {
		t.Error("Alert ID was modified in store")
	}

	if original.ActiveAlerts[0].Details["cpu"] != 90.0 {
		t.Error("Alert details were modified in store")
	}
}

func TestGetAllAgents(t *testing.T) {
	store := NewStateStore()

	store.UpdateAgent(&ServerState{AgentName: "agent1"})
	store.UpdateAgent(&ServerState{AgentName: "agent2"})
	store.UpdateAgent(&ServerState{AgentName: "agent3"})

	agents := store.GetAllAgents()

	if len(agents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(agents))
	}

	// Verify all agents returned
	found := make(map[string]bool)
	for _, agent := range agents {
		found[agent.AgentName] = true
	}

	for _, name := range []string{"agent1", "agent2", "agent3"} {
		if !found[name] {
			t.Errorf("Agent %s not found in results", name)
		}
	}
}

func TestGetAllAgents_EmptyStore(t *testing.T) {
	store := NewStateStore()

	agents := store.GetAllAgents()

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestUpdateHeartbeat_NewAgent(t *testing.T) {
	store := NewStateStore()

	store.UpdateHeartbeat("new-agent")

	state, exists := store.GetAgent("new-agent")
	if !exists {
		t.Fatal("Agent not created by heartbeat")
	}

	if state.AgentName != "new-agent" {
		t.Errorf("AgentName = %v, want new-agent", state.AgentName)
	}

	if state.Status != "online" {
		t.Errorf("Status = %v, want online", state.Status)
	}

	if state.LastSeen.IsZero() {
		t.Error("LastSeen should be set")
	}
}

func TestUpdateHeartbeat_ExistingAgent(t *testing.T) {
	store := NewStateStore()

	// Create agent
	store.UpdateAgent(&ServerState{
		AgentName: "test-agent",
		Status:    "offline",
	})

	// Update heartbeat
	time.Sleep(10 * time.Millisecond)
	store.UpdateHeartbeat("test-agent")

	state, _ := store.GetAgent("test-agent")

	if state.Status != "online" {
		t.Errorf("Status = %v, want online", state.Status)
	}

	// LastSeen should be very recent
	if time.Since(state.LastSeen) > 100*time.Millisecond {
		t.Error("LastSeen not updated properly")
	}
}

func TestCheckOfflineAgents(t *testing.T) {
	store := NewStateStore()

	// Create agents with different last seen times
	now := time.Now()

	// Agent 1: Recent heartbeat (online)
	store.UpdateAgent(&ServerState{AgentName: "agent1"})
	store.agents["agent1"].LastSeen = now

	// Agent 2: Old heartbeat (should be offline)
	store.UpdateAgent(&ServerState{AgentName: "agent2"})
	store.agents["agent2"].LastSeen = now.Add(-5 * time.Minute)

	// Agent 3: Already offline (shouldn't be returned again)
	store.UpdateAgent(&ServerState{AgentName: "agent3"})
	store.agents["agent3"].LastSeen = now.Add(-10 * time.Minute)
	store.agents["agent3"].Status = "offline"

	// Check with 2 minute timeout
	offline := store.CheckOfflineAgents(2 * time.Minute)

	if len(offline) != 1 {
		t.Errorf("Expected 1 offline agent, got %d", len(offline))
	}

	if offline[0].AgentName != "agent2" {
		t.Errorf("Wrong agent marked offline: %v", offline[0].AgentName)
	}

	// Verify agent2 is now marked offline in store
	state, _ := store.GetAgent("agent2")
	if state.Status != "offline" {
		t.Error("Agent2 should be marked offline in store")
	}

	// Verify agent1 still online
	state, _ = store.GetAgent("agent1")
	if state.Status != "online" {
		t.Error("Agent1 should still be online")
	}
}

func TestAddAlert(t *testing.T) {
	store := NewStateStore()

	// Create agent first
	store.UpdateAgent(&ServerState{AgentName: "test-agent"})

	alert := &Alert{
		ID:        "alert1",
		AgentName: "test-agent",
		AlertType: "high_cpu",
		Severity:  "warning",
		Message:   "CPU usage high",
		Status:    "active",
	}

	store.AddAlert(alert)

	// Verify alert in alerts map
	retrieved, exists := store.GetAlert("alert1")
	if !exists {
		t.Fatal("Alert not found after adding")
	}

	if retrieved.ID != "alert1" {
		t.Errorf("Alert ID = %v, want alert1", retrieved.ID)
	}

	// Verify alert added to agent's active alerts
	state, _ := store.GetAgent("test-agent")
	if len(state.ActiveAlerts) != 1 {
		t.Errorf("Expected 1 active alert on agent, got %d", len(state.ActiveAlerts))
	}

	if state.ActiveAlerts[0].ID != "alert1" {
		t.Error("Alert not added to agent's active alerts")
	}
}

func TestAddAlert_AgentNotFound(t *testing.T) {
	store := NewStateStore()

	alert := &Alert{
		ID:        "alert1",
		AgentName: "nonexistent",
		AlertType: "high_cpu",
		Status:    "active",
	}

	// Should not panic
	store.AddAlert(alert)

	// Alert should still be in alerts map
	_, exists := store.GetAlert("alert1")
	if !exists {
		t.Error("Alert not added to alerts map")
	}
}

func TestResolveAlert(t *testing.T) {
	store := NewStateStore()

	// Create agent and alert
	store.UpdateAgent(&ServerState{AgentName: "test-agent"})
	alert := &Alert{
		ID:          "alert1",
		AgentName:   "test-agent",
		AlertType:   "high_cpu",
		Status:      "active",
		TriggeredAt: time.Now(),
	}
	store.AddAlert(alert)

	// Resolve alert
	store.ResolveAlert("alert1")

	// Verify alert marked as resolved
	retrieved, _ := store.GetAlert("alert1")
	if retrieved.Status != "resolved" {
		t.Errorf("Alert status = %v, want resolved", retrieved.Status)
	}

	if retrieved.ResolvedAt == nil {
		t.Error("ResolvedAt should be set")
	}

	// Verify alert removed from agent's active alerts
	state, _ := store.GetAgent("test-agent")
	if len(state.ActiveAlerts) != 0 {
		t.Errorf("Expected 0 active alerts, got %d", len(state.ActiveAlerts))
	}
}

func TestResolveAlert_NotFound(t *testing.T) {
	store := NewStateStore()

	// Should not panic
	store.ResolveAlert("nonexistent")
}

func TestGetActiveAlerts(t *testing.T) {
	store := NewStateStore()

	store.UpdateAgent(&ServerState{AgentName: "agent1"})

	// Add active alerts
	store.AddAlert(&Alert{ID: "alert1", AgentName: "agent1", Status: "active"})
	store.AddAlert(&Alert{ID: "alert2", AgentName: "agent1", Status: "active"})

	// Add resolved alert
	alert3 := &Alert{ID: "alert3", AgentName: "agent1", Status: "active"}
	store.AddAlert(alert3)
	store.ResolveAlert("alert3")

	active := store.GetActiveAlerts()

	if len(active) != 2 {
		t.Errorf("Expected 2 active alerts, got %d", len(active))
	}

	// Verify only active alerts returned
	for _, alert := range active {
		if alert.Status != "active" {
			t.Errorf("Non-active alert returned: %v", alert.Status)
		}
	}
}

func TestGetAlertsByAgent(t *testing.T) {
	store := NewStateStore()

	store.UpdateAgent(&ServerState{AgentName: "agent1"})
	store.UpdateAgent(&ServerState{AgentName: "agent2"})

	store.AddAlert(&Alert{ID: "alert1", AgentName: "agent1", Status: "active"})
	store.AddAlert(&Alert{ID: "alert2", AgentName: "agent1", Status: "resolved"})
	store.AddAlert(&Alert{ID: "alert3", AgentName: "agent2", Status: "active"})

	alerts := store.GetAlertsByAgent("agent1")

	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts for agent1, got %d", len(alerts))
	}

	for _, alert := range alerts {
		if alert.AgentName != "agent1" {
			t.Errorf("Wrong agent alert returned: %v", alert.AgentName)
		}
	}
}

func TestGetAlert_NotFound(t *testing.T) {
	store := NewStateStore()

	_, exists := store.GetAlert("nonexistent")

	if exists {
		t.Error("Expected alert not to exist")
	}
}

// TestConcurrency verifies thread-safety of StateStore
func TestConcurrency(t *testing.T) {
	store := NewStateStore()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.UpdateAgent(&ServerState{
					AgentName: "agent" + string(rune(id)),
					SystemMetrics: metrics.SystemMetrics{
						CPU: metrics.CPUMetrics{
							UsagePercent: float64(j),
						},
					},
				})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.GetAgent("agent" + string(rune(id)))
				store.GetAllAgents()
			}
		}(i)
	}

	// Concurrent heartbeats
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.UpdateHeartbeat("agent" + string(rune(id)))
			}
		}(i)
	}

	// Concurrent alert operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				alert := &Alert{
					ID:        "alert" + string(rune(id)),
					AgentName: "agent" + string(rune(id)),
					Status:    "active",
				}
				store.AddAlert(alert)
				store.GetActiveAlerts()
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	agents := store.GetAllAgents()
	if len(agents) == 0 {
		t.Error("No agents found after concurrent operations")
	}
}

func TestConcurrency_RaceDetection(t *testing.T) {
	// This test should be run with: go test -race
	store := NewStateStore()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.UpdateAgent(&ServerState{
				AgentName: "test-agent",
				Containers: []ContainerState{
					{ID: "c1", State: "running"},
				},
			})
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			state, exists := store.GetAgent("test-agent")
			if exists && state != nil {
				// Access fields to trigger potential race
				_ = state.AgentName
				_ = len(state.Containers)
			}
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}
