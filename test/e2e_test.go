package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anurag/saviour/internal/agent"
	"github.com/anurag/saviour/internal/api"
	"github.com/anurag/saviour/internal/server"
	"github.com/anurag/saviour/pkg/metrics"
)

// TestEndToEnd_MetricsPush tests the flow: Agent → Server → State
func TestEndToEnd_MetricsPush(t *testing.T) {
	// 1. Setup: Create server state store
	state := server.NewStateStore()

	// 2. Setup: Create API handler
	handler := api.NewHandler(state)

	// 3. Setup: Create test server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/metrics/push" {
			handler.HandleMetricsPush(w, r)
		} else if r.URL.Path == "/api/v1/heartbeat" {
			handler.HandleHeartbeat(w, r)
		} else if r.URL.Path == "/api/v1/health" {
			handler.HandleHealth(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	// 4. Test: Push metrics with high CPU
	payload := server.MetricsPushPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		SystemMetrics: metrics.SystemMetrics{
			Timestamp: time.Now(),
			AgentName: "test-agent",
			CPU: metrics.CPUMetrics{
				UsagePercent: 95.0,
			},
			Memory: metrics.MemoryMetrics{
				UsedPercent: 75.0,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiServer.URL+"/api/v1/metrics/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to push metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// 5. Verify: Agent should be in state
	agent, exists := state.GetAgent("test-agent")
	if !exists {
		t.Fatal("Agent not found in state after push")
	}

	if agent.Status != "online" {
		t.Errorf("Expected agent status 'online', got '%s'", agent.Status)
	}

	if agent.SystemMetrics.CPU.UsagePercent != 95.0 {
		t.Errorf("Expected CPU 95.0%%, got %.1f%%", agent.SystemMetrics.CPU.UsagePercent)
	}

	// 6. Test: Push heartbeat
	heartbeat := server.HeartbeatPayload{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	body, _ = json.Marshal(heartbeat)
	req, _ = http.NewRequest("POST", apiServer.URL+"/api/v1/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for heartbeat, got %d", resp.StatusCode)
	}

	// 7. Test: Check health endpoint
	req, _ = http.NewRequest("GET", apiServer.URL+"/api/v1/health", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)

	if health["status"] != "ok" {
		t.Errorf("Expected health status 'ok', got '%v'", health["status"])
	}

	if health["agents_online"] != float64(1) {
		t.Errorf("Expected 1 agent online, got %v", health["agents_online"])
	}
}

// TestEndToEnd_AgentSender tests the agent sender component
func TestEndToEnd_AgentSender(t *testing.T) {
	// Setup: Create mock server
	receivedMetrics := false
	receivedHeartbeat := false

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/metrics/push" {
			receivedMetrics = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success"}`))
		} else if r.URL.Path == "/api/v1/heartbeat" {
			receivedHeartbeat = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success"}`))
		}
	}))
	defer testServer.Close()

	// Create sender
	sender := agent.NewSender(testServer.URL, "test-api-key")

	// Test: Push metrics
	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		CPU: metrics.CPUMetrics{
			UsagePercent: 45.5,
		},
	}

	ctx := context.Background()
	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}

	if !receivedMetrics {
		t.Error("Server did not receive metrics")
	}

	// Test: Send heartbeat
	err = sender.SendHeartbeat(ctx, "test-agent")
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}

	if !receivedHeartbeat {
		t.Error("Server did not receive heartbeat")
	}
}

// TestEndToEnd_OfflineDetection tests offline agent detection
func TestEndToEnd_OfflineDetection(t *testing.T) {
	// Setup state store
	state := server.NewStateStore()

	// Add an agent
	state.UpdateAgent(&server.ServerState{
		AgentName: "test-agent",
		Status:    "online",
	})

	// Wait for heartbeat timeout
	time.Sleep(100 * time.Millisecond)

	// Check for offline agents
	offline := state.CheckOfflineAgents(50 * time.Millisecond)

	if len(offline) != 1 {
		t.Fatalf("Expected 1 offline agent, got %d", len(offline))
	}

	if offline[0].AgentName != "test-agent" {
		t.Errorf("Expected 'test-agent' offline, got '%s'", offline[0].AgentName)
	}

	// Verify agent is marked as offline
	agent, _ := state.GetAgent("test-agent")
	if agent.Status != "offline" {
		t.Errorf("Expected agent status 'offline', got '%s'", agent.Status)
	}
}

// TestEndToEnd_MultipleAgents tests multiple agents pushing to same server
func TestEndToEnd_MultipleAgents(t *testing.T) {
	state := server.NewStateStore()
	handler := api.NewHandler(state)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/metrics/push" {
			handler.HandleMetricsPush(w, r)
		}
	}))
	defer apiServer.Close()

	// Push metrics from 3 different agents
	agents := []string{"agent-1", "agent-2", "agent-3"}

	for _, agentName := range agents {
		payload := server.MetricsPushPayload{
			AgentName: agentName,
			Timestamp: time.Now(),
			SystemMetrics: metrics.SystemMetrics{
				Timestamp: time.Now(),
				AgentName: agentName,
				CPU: metrics.CPUMetrics{
					UsagePercent: 50.0,
				},
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", apiServer.URL+"/api/v1/metrics/push", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to push metrics for %s: %v", agentName, err)
		}
		resp.Body.Close()
	}

	// Verify all agents are in state
	allAgents := state.GetAllAgents()
	if len(allAgents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(allAgents))
	}

	// Verify each agent
	for _, agentName := range agents {
		agent, exists := state.GetAgent(agentName)
		if !exists {
			t.Errorf("Agent %s not found", agentName)
		}
		if agent.Status != "online" {
			t.Errorf("Agent %s status: expected 'online', got '%s'", agentName, agent.Status)
		}
	}
}
