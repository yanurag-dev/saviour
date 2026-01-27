package api

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/anurag/saviour/internal/server"
	"github.com/anurag/saviour/pkg/metrics"
)

// Handler manages HTTP endpoints for the server
type Handler struct {
	state *server.StateStore
}

// NewHandler creates a new API handler
func NewHandler(state *server.StateStore) *Handler {
	return &Handler{
		state: state,
	}
}

// HandleMetricsPush handles POST /api/v1/metrics/push
func (h *Handler) HandleMetricsPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and potentially decompress body
	body, err := h.readBody(r)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer body.Close()

	// Parse metrics payload
	var payload server.MetricsPushPayload
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		log.Printf("Error decoding metrics payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.AgentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	// Create/update server state
	state := &server.ServerState{
		AgentName:     payload.AgentName,
		EC2InstanceID: h.getEC2InstanceID(payload.EC2Metadata),
		SystemMetrics: payload.SystemMetrics,
		Containers:    h.convertContainers(payload.SystemMetrics.Containers),
		ActiveAlerts:  []server.Alert{}, // Will be populated by alert engine
	}

	h.state.UpdateAgent(state)

	log.Printf("Received metrics from agent: %s", payload.AgentName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Metrics received",
	})
}

// HandleHeartbeat handles POST /api/v1/heartbeat
func (h *Handler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse heartbeat payload
	var payload server.HeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding heartbeat payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.AgentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	// Update heartbeat
	h.state.UpdateHeartbeat(payload.AgentName)

	log.Printf("Heartbeat received from agent: %s", payload.AgentName)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// HandleHealth handles GET /api/v1/health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := h.state.GetAllAgents()
	activeAlerts := h.state.GetActiveAlerts()

	health := map[string]interface{}{
		"status":         "ok",
		"agents_online":  countOnlineAgents(agents),
		"agents_offline": countOfflineAgents(agents),
		"active_alerts":  len(activeAlerts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// readBody handles reading and decompressing request body
func (h *Handler) readBody(r *http.Request) (io.ReadCloser, error) {
	// Check if body is gzip compressed
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}
	return r.Body, nil
}

// getEC2InstanceID extracts EC2 instance ID from metadata
func (h *Handler) getEC2InstanceID(metadata *server.EC2Metadata) string {
	if metadata != nil {
		return metadata.InstanceID
	}
	return ""
}

// convertContainers converts metrics containers to server container states
func (h *Handler) convertContainers(containers []metrics.ContainerMetrics) []server.ContainerState {
	result := make([]server.ContainerState, len(containers))
	for i, c := range containers {
		result[i] = server.ContainerState{
			ID:            c.ID,
			Name:          c.Name,
			Image:         c.Image,
			State:         c.State,
			Health:        c.Health,
			CPUPercent:    c.CPUPercent,
			MemoryPercent: calculateMemoryPercent(c.MemoryUsage, c.MemoryLimit),
			MemoryUsage:   c.MemoryUsage,
			MemoryLimit:   c.MemoryLimit,
			RestartCount:  c.RestartCount,
		}
	}
	return result
}

// calculateMemoryPercent calculates memory usage percentage
func calculateMemoryPercent(usage, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return float64(usage) / float64(limit) * 100.0
}

// Helper functions
func countOnlineAgents(agents []*server.ServerState) int {
	count := 0
	for _, agent := range agents {
		if agent.Status == "online" {
			count++
		}
	}
	return count
}

func countOfflineAgents(agents []*server.ServerState) int {
	count := 0
	for _, agent := range agents {
		if agent.Status == "offline" {
			count++
		}
	}
	return count
}
