package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/api/v1/metrics/push", handleMetricsPush)
	http.HandleFunc("/api/v1/heartbeat", handleHeartbeat)
	http.HandleFunc("/api/v1/health", handleHealth)

	addr := ":8080"
	log.Printf("🚀 Mock server starting on %s", addr)
	log.Printf("   Metrics endpoint: http://localhost%s/api/v1/metrics/push", addr)
	log.Printf("   Heartbeat endpoint: http://localhost%s/api/v1/heartbeat", addr)
	log.Printf("   Health endpoint: http://localhost%s/api/v1/health", addr)
	log.Println()

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleMetricsPush(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authorization
	auth := r.Header.Get("Authorization")
	log.Printf("📊 Metrics received | Auth: %s", maskToken(auth))

	// Handle decompression
	var reader io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to decompress", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		reader = gz
		log.Println("   ✓ Decompressed gzip payload")
	}

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Parse JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract info
	agentName, _ := payload["agent_name"].(string)
	timestamp, _ := payload["timestamp"].(string)
	
	// Count containers
	containerCount := 0
	if metrics, ok := payload["metrics"].(map[string]interface{}); ok {
		if containers, ok := metrics["containers"].([]interface{}); ok {
			containerCount = len(containers)
		}
	}

	log.Printf("   Agent: %s", agentName)
	log.Printf("   Timestamp: %s", timestamp)
	log.Printf("   Payload size: %d bytes", len(body))
	log.Printf("   Containers: %d", containerCount)
	log.Println()

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Metrics received",
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authorization
	auth := r.Header.Get("Authorization")
	
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Parse JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	agentName, _ := payload["agent_name"].(string)
	status, _ := payload["status"].(string)
	
	log.Printf("♥️  Heartbeat from %s | Status: %s | Auth: %s", agentName, status, maskToken(auth))

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Heartbeat received",
		"timestamp": time.Now().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func maskToken(auth string) string {
	if auth == "" {
		return "(none)"
	}
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		return auth
	}
	token := parts[1]
	if len(token) > 8 {
		return fmt.Sprintf("Bearer %s...%s", token[:4], token[len(token)-4:])
	}
	return auth
}
