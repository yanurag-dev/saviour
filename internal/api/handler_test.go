package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anurag/saviour/internal/server"
	"github.com/anurag/saviour/pkg/metrics"
)

func TestNewHandler(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.state != state {
		t.Error("Handler state not set correctly")
	}
}

func TestHandleMetricsPush_ValidRequest(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
			CPU: metrics.CPUMetrics{
				UsagePercent: 45.2,
			},
			Memory: metrics.MemoryMetrics{
				Total:       8589934592, // 8GB
				Used:        4294967296, // 4GB
				UsedPercent: 50.0,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify agent was added to state
	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state")
	}

	if agent.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", agent.AgentName)
	}

	if agent.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", agent.Status)
	}
}

func TestHandleMetricsPush_InvalidMethod(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("GET", "/api/v1/metrics/push", nil)
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

func TestHandleMetricsPush_RequestTooLarge(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	// Create a large payload (11MB > 10MB limit)
	largeData := make([]byte, 11*1024*1024)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(largeData))
	req.ContentLength = int64(len(largeData))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", rec.Code)
	}
}

func TestHandleMetricsPush_InvalidJSON(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestHandleMetricsPush_MissingAgentName(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.MetricsPushPayload{
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestHandleMetricsPush_WithEC2Metadata(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		EC2Metadata: &server.EC2Metadata{
			InstanceID:       "i-1234567890abcdef0",
			InstanceType:     "t3.medium",
			Region:           "us-west-2",
			AvailabilityZone: "us-west-2a",
		},
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state")
	}

	if agent.EC2InstanceID != "i-1234567890abcdef0" {
		t.Errorf("Expected EC2 instance ID 'i-1234567890abcdef0', got '%s'", agent.EC2InstanceID)
	}
}

func TestHandleMetricsPush_WithContainers(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
			Containers: []metrics.ContainerMetrics{
				{
					ID:            "container-123",
					Name:          "nginx",
					Image:         "nginx:latest",
					State:         "running",
					Health:        "healthy",
					CPUPercent:    25.5,
					MemoryUsage:   104857600, // 100MB
					MemoryLimit:   536870912, // 512MB
					RestartCount:  0,
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state")
	}

	if len(agent.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(agent.Containers))
	}

	container := agent.Containers[0]
	if container.ID != "container-123" {
		t.Errorf("Expected container ID 'container-123', got '%s'", container.ID)
	}

	if container.Name != "nginx" {
		t.Errorf("Expected container name 'nginx', got '%s'", container.Name)
	}

	expectedMemPercent := float64(104857600) / float64(536870912) * 100.0
	if container.MemoryPercent != expectedMemPercent {
		t.Errorf("Expected memory percent %.2f, got %.2f", expectedMemPercent, container.MemoryPercent)
	}
}

func TestHandleMetricsPush_GzipCompressed(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
		},
	}

	// Compress the payload
	jsonData, _ := json.Marshal(payload)
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, _ = gzWriter.Write(jsonData)
	gzWriter.Close()

	req := httptest.NewRequest("POST", "/api/v1/metrics/push", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.HandleMetricsPush(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state after gzip request")
	}

	if agent.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", agent.AgentName)
	}
}

func TestHandleHeartbeat_ValidRequest(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.HeartbeatPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/heartbeat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleHeartbeat(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify agent exists in state
	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state")
	}

	if agent.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", agent.AgentName)
	}

	if agent.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", agent.Status)
	}
}

func TestHandleHeartbeat_InvalidMethod(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("GET", "/api/v1/heartbeat", nil)
	rec := httptest.NewRecorder()

	handler.HandleHeartbeat(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

func TestHandleHeartbeat_InvalidJSON(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("POST", "/api/v1/heartbeat", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	handler.HandleHeartbeat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestHandleHeartbeat_MissingAgentName(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	payload := server.HeartbeatPayload{
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/heartbeat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleHeartbeat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestHandleHeartbeat_UpdatesExistingAgent(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	// First, create an agent with metrics
	metricsPayload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
		},
	}

	body, _ := json.Marshal(metricsPayload)
	req := httptest.NewRequest("POST", "/api/v1/metrics/push", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.HandleMetricsPush(rec, req)

	// Get the original LastSeen
	agent1, _ := state.GetAgent("test-agent")
	originalLastSeen := agent1.LastSeen

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Send a heartbeat
	heartbeatPayload := server.HeartbeatPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	body, _ = json.Marshal(heartbeatPayload)
	req = httptest.NewRequest("POST", "/api/v1/heartbeat", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler.HandleHeartbeat(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify LastSeen was updated
	agent2, _ := state.GetAgent("test-agent")
	if !agent2.LastSeen.After(originalLastSeen) {
		t.Error("LastSeen was not updated by heartbeat")
	}
}

func TestHandleHealth_ValidRequest(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	// Add some agents (UpdateAgent automatically sets status to "online")
	state.UpdateAgent(&server.ServerState{
		AgentName: "agent1",
	})
	state.UpdateAgent(&server.ServerState{
		AgentName: "agent2",
	})
	state.UpdateAgent(&server.ServerState{
		AgentName: "agent3",
	})

	// Mark one agent as offline by checking with a 0 timeout
	// This will mark all agents as offline, so we need to refresh the online ones
	state.CheckOfflineAgents(0)

	// Refresh two agents to bring them back online
	state.UpdateAgent(&server.ServerState{
		AgentName: "agent1",
	})
	state.UpdateAgent(&server.ServerState{
		AgentName: "agent2",
	})

	// Add an alert
	state.AddAlert(&server.Alert{
		ID:        "alert1",
		AgentName: "agent1",
		Status:    "active",
	})

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", health["status"])
	}

	if health["agents_online"] != float64(2) {
		t.Errorf("Expected 2 agents online, got %v", health["agents_online"])
	}

	if health["agents_offline"] != float64(1) {
		t.Errorf("Expected 1 agent offline, got %v", health["agents_offline"])
	}

	if health["active_alerts"] != float64(1) {
		t.Errorf("Expected 1 active alert, got %v", health["active_alerts"])
	}
}

func TestHandleHealth_InvalidMethod(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("POST", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

func TestHandleHealth_EmptyState(t *testing.T) {
	state := server.NewStateStore()
	handler := NewHandler(state)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["agents_online"] != float64(0) {
		t.Errorf("Expected 0 agents online, got %v", health["agents_online"])
	}

	if health["active_alerts"] != float64(0) {
		t.Errorf("Expected 0 active alerts, got %v", health["active_alerts"])
	}
}

func TestGetEC2InstanceID(t *testing.T) {
	handler := NewHandler(nil)

	tests := []struct {
		name     string
		metadata *server.EC2Metadata
		expected string
	}{
		{
			name: "with metadata",
			metadata: &server.EC2Metadata{
				InstanceID: "i-1234567890abcdef0",
			},
			expected: "i-1234567890abcdef0",
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getEC2InstanceID(tt.metadata)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConvertContainers(t *testing.T) {
	handler := NewHandler(nil)

	containers := []metrics.ContainerMetrics{
		{
			ID:            "container1",
			Name:          "nginx",
			Image:         "nginx:latest",
			State:         "running",
			Health:        "healthy",
			CPUPercent:    25.5,
			MemoryUsage:   104857600, // 100MB
			MemoryLimit:   536870912, // 512MB
			RestartCount:  2,
		},
		{
			ID:            "container2",
			Name:          "redis",
			Image:         "redis:alpine",
			State:         "exited",
			Health:        "none",
			CPUPercent:    0,
			MemoryUsage:   0,
			MemoryLimit:   0,
			RestartCount:  0,
		},
	}

	result := handler.convertContainers(containers)

	if len(result) != 2 {
		t.Fatalf("Expected 2 containers, got %d", len(result))
	}

	// Check first container
	if result[0].ID != "container1" {
		t.Errorf("Expected ID 'container1', got '%s'", result[0].ID)
	}

	if result[0].Name != "nginx" {
		t.Errorf("Expected name 'nginx', got '%s'", result[0].Name)
	}

	expectedMemPercent := float64(104857600) / float64(536870912) * 100.0
	if result[0].MemoryPercent != expectedMemPercent {
		t.Errorf("Expected memory percent %.2f, got %.2f", expectedMemPercent, result[0].MemoryPercent)
	}

	// Check second container (with 0 limit)
	if result[1].MemoryPercent != 0 {
		t.Errorf("Expected memory percent 0 for zero limit, got %.2f", result[1].MemoryPercent)
	}
}

func TestCalculateMemoryPercent(t *testing.T) {
	tests := []struct {
		name     string
		usage    uint64
		limit    uint64
		expected float64
	}{
		{
			name:     "50% usage",
			usage:    536870912, // 512MB
			limit:    1073741824, // 1GB
			expected: 50.0,
		},
		{
			name:     "100% usage",
			usage:    1073741824, // 1GB
			limit:    1073741824, // 1GB
			expected: 100.0,
		},
		{
			name:     "zero limit",
			usage:    100,
			limit:    0,
			expected: 0,
		},
		{
			name:     "zero usage",
			usage:    0,
			limit:    1073741824,
			expected: 0,
		},
		{
			name:     "partial usage",
			usage:    268435456, // 256MB
			limit:    1073741824, // 1GB
			expected: 25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMemoryPercent(tt.usage, tt.limit)
			if result != tt.expected {
				t.Errorf("Expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestCountOnlineAgents(t *testing.T) {
	agents := []*server.ServerState{
		{AgentName: "agent1", Status: "online"},
		{AgentName: "agent2", Status: "online"},
		{AgentName: "agent3", Status: "offline"},
		{AgentName: "agent4", Status: "degraded"},
		{AgentName: "agent5", Status: "online"},
	}

	count := countOnlineAgents(agents)
	if count != 3 {
		t.Errorf("Expected 3 online agents, got %d", count)
	}
}

func TestCountOfflineAgents(t *testing.T) {
	agents := []*server.ServerState{
		{AgentName: "agent1", Status: "online"},
		{AgentName: "agent2", Status: "offline"},
		{AgentName: "agent3", Status: "offline"},
		{AgentName: "agent4", Status: "degraded"},
		{AgentName: "agent5", Status: "offline"},
	}

	count := countOfflineAgents(agents)
	if count != 3 {
		t.Errorf("Expected 3 offline agents, got %d", count)
	}
}

func TestCountAgents_EmptyList(t *testing.T) {
	var agents []*server.ServerState

	online := countOnlineAgents(agents)
	offline := countOfflineAgents(agents)

	if online != 0 {
		t.Errorf("Expected 0 online agents, got %d", online)
	}

	if offline != 0 {
		t.Errorf("Expected 0 offline agents, got %d", offline)
	}
}
