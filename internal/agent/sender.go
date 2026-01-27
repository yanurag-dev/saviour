package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anurag/saviour/pkg/metrics"
)

// Sender handles pushing metrics to the central server
type Sender struct {
	serverURL   string
	apiKey      string
	client      *http.Client
	maxRetries  int
	retryBackoff time.Duration
}

// NewSender creates a new metrics sender
func NewSender(serverURL, apiKey string) *Sender {
	return &Sender{
		serverURL: serverURL,
		apiKey:    apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries:   3,
		retryBackoff: 2 * time.Second,
	}
}

// MetricsPayload represents the data sent to the server
type MetricsPayload struct {
	AgentName string                `json:"agent_name"`
	Timestamp time.Time             `json:"timestamp"`
	System    *metrics.SystemMetrics `json:"metrics"`
}

// HeartbeatPayload represents a lightweight heartbeat
type HeartbeatPayload struct {
	AgentName string    `json:"agent_name"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "online"
}

// PushMetrics sends metrics to the central server
func (s *Sender) PushMetrics(ctx context.Context, m *metrics.SystemMetrics) error {
	if s.serverURL == "" {
		// No server configured, skip push
		return nil
	}

	payload := MetricsPayload{
		AgentName: m.AgentName,
		Timestamp: m.Timestamp,
		System:    m,
	}

	endpoint := s.serverURL + "/api/v1/metrics/push"
	return s.sendWithRetry(ctx, endpoint, payload)
}

// SendHeartbeat sends a lightweight heartbeat signal
func (s *Sender) SendHeartbeat(ctx context.Context, agentName string) error {
	if s.serverURL == "" {
		return nil
	}

	payload := HeartbeatPayload{
		AgentName: agentName,
		Timestamp: time.Now(),
		Status:    "online",
	}

	endpoint := s.serverURL + "/api/v1/heartbeat"
	return s.sendWithRetry(ctx, endpoint, payload)
}

// sendWithRetry sends a request with exponential backoff retry
func (s *Sender) sendWithRetry(ctx context.Context, endpoint string, payload interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			backoff := s.retryBackoff * time.Duration(1<<uint(attempt-1)) // Exponential backoff
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := s.send(ctx, endpoint, payload)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return err // Don't retry on client errors
		}
	}

	return fmt.Errorf("failed after %d retries: %w", s.maxRetries, lastErr)
}

// send performs the actual HTTP POST
func (s *Sender) send(ctx context.Context, endpoint string, payload interface{}) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Compress payload if large (> 1KB)
	var body io.Reader
	var contentEncoding string
	if len(jsonData) > 1024 {
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)
		if _, err := gzipWriter.Write(jsonData); err != nil {
			return fmt.Errorf("failed to compress payload: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
		body = &buf
		contentEncoding = "gzip"
	} else {
		body = bytes.NewReader(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	req.Header.Set("User-Agent", "saviour-agent/1.0")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // Success
	}

	// Read error response
	bodyBytes, _ := io.ReadAll(resp.Body)
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Message:    string(bodyBytes),
	}
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// isRetryable determines if an error should trigger a retry
func isRetryable(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		// Retry on 5xx server errors and 429 rate limit
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}
	// Retry on network errors
	return true
}
