package alerting

import (
	"errors"
	"testing"
	"time"
)

// MockStateStore implements StateStore interface for testing
type MockStateStore struct {
	agents        []*ServerState
	offlineAgents []*ServerState
	alerts        []*Alert
}

func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		agents:        make([]*ServerState, 0),
		offlineAgents: make([]*ServerState, 0),
		alerts:        make([]*Alert, 0),
	}
}

func (m *MockStateStore) GetAllAgents() []*ServerState {
	return m.agents
}

func (m *MockStateStore) CheckOfflineAgents(timeout time.Duration) []*ServerState {
	return m.offlineAgents
}

func (m *MockStateStore) AddAlert(alert *Alert) {
	m.alerts = append(m.alerts, alert)
}

// MockNotifier implements Notifier interface for testing
type MockNotifier struct {
	sentAlerts []*Alert
	shouldFail bool
}

func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		sentAlerts: make([]*Alert, 0),
	}
}

func (m *MockNotifier) SendAlert(alert *Alert) error {
	if m.shouldFail {
		return errors.New("mock notifier error")
	}
	m.sentAlerts = append(m.sentAlerts, alert)
	return nil
}

func TestNewEngine(t *testing.T) {
	state := NewMockStateStore()
	config := &Config{
		Enabled:       true,
		CheckInterval: 30 * time.Second,
	}
	notifier := NewMockNotifier()

	engine := NewEngine(state, config, notifier)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	if engine.state != state {
		t.Error("Engine state not set correctly")
	}

	if engine.config != config {
		t.Error("Engine config not set correctly")
	}

	if engine.notifier != notifier {
		t.Error("Engine notifier not set correctly")
	}

	if engine.recentAlerts == nil {
		t.Error("recentAlerts map not initialized")
	}
}

func TestCheckOfflineAgents(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		HeartbeatTimeout:     1 * time.Minute,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	// Add an offline agent
	offlineAgent := &ServerState{
		AgentName: "offline-agent",
		Status:    "offline",
		LastSeen:  time.Now().Add(-2 * time.Minute),
	}
	state.offlineAgents = append(state.offlineAgents, offlineAgent)

	engine.checkOfflineAgents()

	// Verify alert was created
	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "agent_offline" {
		t.Errorf("Expected alert type 'agent_offline', got '%s'", alert.AlertType)
	}

	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}

	if alert.AgentName != "offline-agent" {
		t.Errorf("Expected agent name 'offline-agent', got '%s'", alert.AgentName)
	}

	// Verify notification was sent
	if len(notifier.sentAlerts) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifier.sentAlerts))
	}
}

func TestCheckOfflineAgents_NotificationFailure(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	notifier.shouldFail = true
	config := &Config{
		Enabled:              true,
		HeartbeatTimeout:     1 * time.Minute,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	offlineAgent := &ServerState{
		AgentName: "offline-agent",
		Status:    "offline",
		LastSeen:  time.Now().Add(-2 * time.Minute),
	}
	state.offlineAgents = append(state.offlineAgents, offlineAgent)

	engine.checkOfflineAgents()

	// Alert should still be added to state even if notification fails
	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert in state, got %d", len(state.alerts))
	}

	// Notification should have been attempted but failed
	if len(notifier.sentAlerts) != 0 {
		t.Errorf("Expected 0 successful notifications, got %d", len(notifier.sentAlerts))
	}
}

func TestCheckSystemAlerts_CPU(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		SystemCPUThreshold:   80.0,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 85.5,
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "system_cpu_high" {
		t.Errorf("Expected alert type 'system_cpu_high', got '%s'", alert.AlertType)
	}

	if alert.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alert.Severity)
	}
}

func TestCheckSystemAlerts_Memory(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:                 true,
		SystemMemoryThreshold:   90.0,
		DeduplicationEnabled:    false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			Memory: MemoryMetrics{
				UsedPercent: 92.5,
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "system_memory_high" {
		t.Errorf("Expected alert type 'system_memory_high', got '%s'", alert.AlertType)
	}

	if alert.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alert.Severity)
	}
}

func TestCheckSystemAlerts_Disk(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:               true,
		SystemDiskThreshold:   85.0,
		DeduplicationEnabled:  false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			Disk: []DiskMetrics{
				{
					MountPoint:  "/",
					UsedPercent: 88.5,
				},
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "system_disk_high" {
		t.Errorf("Expected alert type 'system_disk_high', got '%s'", alert.AlertType)
	}

	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}
}

func TestCheckSystemAlerts_MultipleDisksMountPoints(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:               true,
		SystemDiskThreshold:   80.0,
		DeduplicationEnabled:  false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			Disk: []DiskMetrics{
				{
					MountPoint:  "/",
					UsedPercent: 85.0,
				},
				{
					MountPoint:  "/data",
					UsedPercent: 90.0,
				},
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 2 {
		t.Fatalf("Expected 2 alerts (one per mount), got %d", len(state.alerts))
	}
}

func TestCheckSystemAlerts_BelowThreshold(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:                 true,
		SystemCPUThreshold:      80.0,
		SystemMemoryThreshold:   90.0,
		SystemDiskThreshold:     85.0,
		DeduplicationEnabled:    false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 70.0,
			},
			Memory: MemoryMetrics{
				UsedPercent: 75.0,
			},
			Disk: []DiskMetrics{
				{
					MountPoint:  "/",
					UsedPercent: 60.0,
				},
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 0 {
		t.Errorf("Expected 0 alerts when below threshold, got %d", len(state.alerts))
	}
}

func TestCheckSystemAlerts_ThresholdsDisabled(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:                 true,
		SystemCPUThreshold:      0, // Disabled
		SystemMemoryThreshold:   0, // Disabled
		SystemDiskThreshold:     0, // Disabled
		DeduplicationEnabled:    false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 95.0,
			},
			Memory: MemoryMetrics{
				UsedPercent: 95.0,
			},
			Disk: []DiskMetrics{
				{
					MountPoint:  "/",
					UsedPercent: 95.0,
				},
			},
		},
	}

	engine.checkSystemAlerts(agent)

	if len(state.alerts) != 0 {
		t.Errorf("Expected 0 alerts when thresholds are disabled, got %d", len(state.alerts))
	}
}

func TestCheckContainerAlerts_Stopped(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:            "container-123",
				Name:          "nginx",
				State:         "exited",
				PreviousState: "running",
			},
		},
	}

	engine.checkContainerAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "container_stopped" {
		t.Errorf("Expected alert type 'container_stopped', got '%s'", alert.AlertType)
	}

	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}
}

func TestCheckContainerAlerts_Unhealthy(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:     "container-123",
				Name:   "nginx",
				State:  "running",
				Health: "unhealthy",
			},
		},
	}

	engine.checkContainerAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "container_unhealthy" {
		t.Errorf("Expected alert type 'container_unhealthy', got '%s'", alert.AlertType)
	}

	if alert.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alert.Severity)
	}
}

func TestCheckContainerAlerts_HighCPU(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:         "container-123",
				Name:       "nginx",
				State:      "running",
				CPUPercent: 95.5,
			},
		},
	}

	engine.checkContainerAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "container_cpu_high" {
		t.Errorf("Expected alert type 'container_cpu_high', got '%s'", alert.AlertType)
	}

	if alert.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", alert.Severity)
	}
}

func TestCheckContainerAlerts_HighMemory(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:            "container-123",
				Name:          "nginx",
				State:         "running",
				MemoryPercent: 96.5,
			},
		},
	}

	engine.checkContainerAlerts(agent)

	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(state.alerts))
	}

	alert := state.alerts[0]
	if alert.AlertType != "container_memory_high" {
		t.Errorf("Expected alert type 'container_memory_high', got '%s'", alert.AlertType)
	}

	if alert.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", alert.Severity)
	}
}

func TestCheckContainerAlerts_MultipleAlerts(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:            "container-1",
				Name:          "nginx",
				State:         "running",
				Health:        "unhealthy",
				CPUPercent:    95.0,
				MemoryPercent: 97.0,
			},
		},
	}

	engine.checkContainerAlerts(agent)

	// Should create 3 alerts: unhealthy, high CPU, high memory
	if len(state.alerts) != 3 {
		t.Fatalf("Expected 3 alerts, got %d", len(state.alerts))
	}
}

func TestCheckContainerAlerts_NoAlerts(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	agent := &ServerState{
		AgentName: "test-agent",
		Status:    "online",
		Containers: []ContainerState{
			{
				ID:            "container-123",
				Name:          "nginx",
				State:         "running",
				PreviousState: "running",
				Health:        "healthy",
				CPUPercent:    45.0,
				MemoryPercent: 60.0,
			},
		},
	}

	engine.checkContainerAlerts(agent)

	if len(state.alerts) != 0 {
		t.Errorf("Expected 0 alerts for healthy container, got %d", len(state.alerts))
	}
}

func TestShouldSendAlert_DeduplicationDisabled(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	// Should always return true when deduplication is disabled
	if !engine.shouldSendAlert("test-alert") {
		t.Error("Expected shouldSendAlert to return true when deduplication is disabled")
	}

	// Even after marking as sent
	engine.markAlertSent("test-alert")
	if !engine.shouldSendAlert("test-alert") {
		t.Error("Expected shouldSendAlert to return true when deduplication is disabled")
	}
}

func TestShouldSendAlert_DeduplicationEnabled(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: true,
		DeduplicationWindow:  5 * time.Minute,
	}

	engine := NewEngine(state, config, notifier)

	alertKey := "test-alert"

	// First time should send
	if !engine.shouldSendAlert(alertKey) {
		t.Error("Expected shouldSendAlert to return true for new alert")
	}

	// Mark as sent
	engine.markAlertSent(alertKey)

	// Immediately after should not send (within deduplication window)
	if engine.shouldSendAlert(alertKey) {
		t.Error("Expected shouldSendAlert to return false within deduplication window")
	}

	// Manually set the time to past the deduplication window
	engine.mu.Lock()
	engine.recentAlerts[alertKey] = time.Now().Add(-6 * time.Minute)
	engine.mu.Unlock()

	// After window should send again
	if !engine.shouldSendAlert(alertKey) {
		t.Error("Expected shouldSendAlert to return true after deduplication window")
	}
}

func TestMarkAlertSent(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: true,
		DeduplicationWindow:  5 * time.Minute,
	}

	engine := NewEngine(state, config, notifier)

	alertKey := "test-alert"

	// Initially should not exist
	engine.mu.RLock()
	_, exists := engine.recentAlerts[alertKey]
	engine.mu.RUnlock()

	if exists {
		t.Error("Alert key should not exist initially")
	}

	// Mark as sent
	engine.markAlertSent(alertKey)

	// Should now exist
	engine.mu.RLock()
	timestamp, exists := engine.recentAlerts[alertKey]
	engine.mu.RUnlock()

	if !exists {
		t.Fatal("Alert key should exist after marking as sent")
	}

	if time.Since(timestamp) > 1*time.Second {
		t.Error("Timestamp should be recent")
	}
}

func TestCleanupDeduplication(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: true,
		DeduplicationWindow:  5 * time.Minute,
	}

	engine := NewEngine(state, config, notifier)

	// Add some old and recent alerts
	engine.recentAlerts["old-alert"] = time.Now().Add(-15 * time.Minute)
	engine.recentAlerts["recent-alert"] = time.Now().Add(-2 * time.Minute)
	engine.recentAlerts["very-old-alert"] = time.Now().Add(-1 * time.Hour)

	engine.cleanupDeduplication()

	engine.mu.RLock()
	defer engine.mu.RUnlock()

	// Recent alert should remain (within 2x deduplication window)
	if _, exists := engine.recentAlerts["recent-alert"]; !exists {
		t.Error("Recent alert should not be cleaned up")
	}

	// Old alerts should be removed (beyond 2x deduplication window)
	if _, exists := engine.recentAlerts["old-alert"]; exists {
		t.Error("Old alert should be cleaned up")
	}

	if _, exists := engine.recentAlerts["very-old-alert"]; exists {
		t.Error("Very old alert should be cleaned up")
	}

	// Should have exactly 1 entry remaining
	if len(engine.recentAlerts) != 1 {
		t.Errorf("Expected 1 recent alert, got %d", len(engine.recentAlerts))
	}
}

func TestCheckAlerts_Integration(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:                 true,
		HeartbeatTimeout:        1 * time.Minute,
		SystemCPUThreshold:      80.0,
		SystemMemoryThreshold:   90.0,
		SystemDiskThreshold:     85.0,
		DeduplicationEnabled:    false,
	}

	engine := NewEngine(state, config, notifier)

	// Add an offline agent
	state.offlineAgents = append(state.offlineAgents, &ServerState{
		AgentName: "offline-agent",
		Status:    "offline",
		LastSeen:  time.Now().Add(-2 * time.Minute),
	})

	// Add an online agent with high metrics
	state.agents = append(state.agents, &ServerState{
		AgentName: "online-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 85.0,
			},
			Memory: MemoryMetrics{
				UsedPercent: 95.0,
			},
			Disk: []DiskMetrics{
				{
					MountPoint:  "/",
					UsedPercent: 90.0,
				},
			},
		},
		Containers: []ContainerState{
			{
				ID:            "container-1",
				Name:          "nginx",
				State:         "exited",
				PreviousState: "running",
			},
		},
	})

	engine.checkAlerts()

	// Should have alerts for:
	// 1. Offline agent
	// 2. High CPU
	// 3. High memory
	// 4. High disk
	// 5. Container stopped
	if len(state.alerts) != 5 {
		t.Errorf("Expected 5 alerts total, got %d", len(state.alerts))
	}

	// Verify notifications were sent
	if len(notifier.sentAlerts) != 5 {
		t.Errorf("Expected 5 notifications, got %d", len(notifier.sentAlerts))
	}
}

func TestCheckAlerts_OnlyOnlineAgents(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:                 true,
		SystemCPUThreshold:      80.0,
		DeduplicationEnabled:    false,
	}

	engine := NewEngine(state, config, notifier)

	// Add an offline agent with high CPU (should not trigger system alerts)
	state.agents = append(state.agents, &ServerState{
		AgentName: "offline-agent",
		Status:    "offline",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 95.0,
			},
		},
	})

	// Add an online agent with normal CPU
	state.agents = append(state.agents, &ServerState{
		AgentName: "online-agent",
		Status:    "online",
		SystemMetrics: SystemMetrics{
			CPU: CPUMetrics{
				UsagePercent: 50.0,
			},
		},
	})

	engine.checkAlerts()

	// Should have no alerts (offline agent ignored, online agent below threshold)
	if len(state.alerts) != 0 {
		t.Errorf("Expected 0 alerts, got %d", len(state.alerts))
	}
}

func TestSendAlert(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: true,
		DeduplicationWindow:  5 * time.Minute,
	}

	engine := NewEngine(state, config, notifier)

	alert := &Alert{
		ID:        "alert-123",
		AgentName: "test-agent",
		AlertType: "test_alert",
		Severity:  "warning",
		Message:   "Test alert",
		Status:    "active",
	}

	alertKey := "test:alert"

	engine.sendAlert(alert, alertKey)

	// Verify alert was added to state
	if len(state.alerts) != 1 {
		t.Fatalf("Expected 1 alert in state, got %d", len(state.alerts))
	}

	// Verify notification was sent
	if len(notifier.sentAlerts) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifier.sentAlerts))
	}

	// Verify NotifiedAt was set
	if alert.NotifiedAt == nil {
		t.Error("NotifiedAt should be set after sending")
	}

	// Verify alert was marked as sent for deduplication
	engine.mu.RLock()
	_, exists := engine.recentAlerts[alertKey]
	engine.mu.RUnlock()

	if !exists {
		t.Error("Alert should be marked as sent for deduplication")
	}
}

func TestSendAlert_NotificationFails(t *testing.T) {
	state := NewMockStateStore()
	notifier := NewMockNotifier()
	notifier.shouldFail = true
	config := &Config{
		Enabled:              true,
		DeduplicationEnabled: false,
	}

	engine := NewEngine(state, config, notifier)

	alert := &Alert{
		ID:        "alert-123",
		AgentName: "test-agent",
		AlertType: "test_alert",
		Severity:  "warning",
		Message:   "Test alert",
		Status:    "active",
	}

	engine.sendAlert(alert, "test:alert")

	// Alert should still be added to state
	if len(state.alerts) != 1 {
		t.Errorf("Expected 1 alert in state even on notification failure, got %d", len(state.alerts))
	}

	// NotifiedAt should NOT be set if notification failed
	if alert.NotifiedAt != nil {
		t.Error("NotifiedAt should not be set when notification fails")
	}
}
