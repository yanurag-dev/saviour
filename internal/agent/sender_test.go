package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anurag/saviour/internal/server"
	"github.com/anurag/saviour/pkg/metrics"
)

func TestNewSender(t *testing.T) {
	sender := NewSender("http://localhost:8080", "test-api-key")

	if sender == nil {
		t.Fatal("NewSender returned nil")
	}

	if sender.serverURL != "http://localhost:8080" {
		t.Errorf("Expected serverURL 'http://localhost:8080', got '%s'", sender.serverURL)
	}

	if sender.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey 'test-api-key', got '%s'", sender.apiKey)
	}

	if sender.client == nil {
		t.Error("HTTP client not initialized")
	}

	if sender.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", sender.maxRetries)
	}

	if sender.retryBackoff != 2*time.Second {
		t.Errorf("Expected retryBackoff 2s, got %v", sender.retryBackoff)
	}

	if sender.ec2Client == nil {
		t.Error("EC2 client not initialized")
	}
}

func TestPushMetrics_Success(t *testing.T) {
	receivedPayload := false
	var capturedPayload MetricsPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if !strings.HasSuffix(r.URL.Path, "/api/v1/metrics/push") {
			t.Errorf("Expected /api/v1/metrics/push endpoint, got %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected 'Bearer test-api-key', got '%s'", auth)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedPayload)
		receivedPayload = true

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		CPU: metrics.CPUMetrics{
			UsagePercent: 45.5,
		},
	}

	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}

	if !receivedPayload {
		t.Error("Server did not receive payload")
	}

	if capturedPayload.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", capturedPayload.AgentName)
	}
}

func TestPushMetrics_NoServerURL(t *testing.T) {
	sender := NewSender("", "test-api-key")
	ctx := context.Background()

	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Errorf("Expected no error when serverURL is empty, got %v", err)
	}
}

func TestPushMetrics_WithEC2Metadata(t *testing.T) {
	var capturedPayload MetricsPayload

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	sender := NewSender(testServer.URL, "test-api-key")
	sender.ec2Metadata = &server.EC2Metadata{
		InstanceID:   "i-1234567890abcdef0",
		InstanceType: "t3.medium",
		Region:       "us-west-2",
	}

	ctx := context.Background()
	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}

	if capturedPayload.EC2Metadata == nil {
		t.Fatal("EC2 metadata not included in payload")
	}

	if capturedPayload.EC2Metadata.InstanceID != "i-1234567890abcdef0" {
		t.Errorf("Expected instance ID 'i-1234567890abcdef0', got '%s'", capturedPayload.EC2Metadata.InstanceID)
	}
}

func TestSendHeartbeat_Success(t *testing.T) {
	receivedHeartbeat := false
	var capturedPayload HeartbeatPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/v1/heartbeat") {
			t.Errorf("Expected /api/v1/heartbeat endpoint, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedPayload)
		receivedHeartbeat = true

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	err := sender.SendHeartbeat(ctx, "test-agent")
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}

	if !receivedHeartbeat {
		t.Error("Server did not receive heartbeat")
	}

	if capturedPayload.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", capturedPayload.AgentName)
	}

	if capturedPayload.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", capturedPayload.Status)
	}
}

func TestSendHeartbeat_NoServerURL(t *testing.T) {
	sender := NewSender("", "test-api-key")
	ctx := context.Background()

	err := sender.SendHeartbeat(ctx, "test-agent")
	if err != nil {
		t.Errorf("Expected no error when serverURL is empty, got %v", err)
	}
}

func TestSend_GzipCompression(t *testing.T) {
	receivedGzip := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		if contentEncoding == "gzip" {
			receivedGzip = true

			// Decompress and verify
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatalf("Failed to create gzip reader: %v", err)
			}
			defer reader.Close()

			var payload MetricsPayload
			if err := json.NewDecoder(reader).Decode(&payload); err != nil {
				t.Fatalf("Failed to decode gzipped payload: %v", err)
			}
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	// Create a large payload (> 1KB) to trigger compression
	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
		Containers: make([]metrics.ContainerMetrics, 10),
	}
	for i := range m.Containers {
		m.Containers[i] = metrics.ContainerMetrics{
			ID:    strings.Repeat("x", 100),
			Name:  strings.Repeat("y", 100),
			Image: strings.Repeat("z", 100),
		}
	}

	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}

	if !receivedGzip {
		t.Error("Expected gzip compression for large payload")
	}
}

func TestSend_SmallPayloadNoCompression(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		if contentEncoding == "gzip" {
			t.Error("Expected no compression for small payload")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	m := &metrics.SystemMetrics{
		AgentName: "test-agent",
		Timestamp: time.Now(),
	}

	err := sender.PushMetrics(ctx, m)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}
}

func TestSendWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)
	if err != nil {
		t.Fatalf("sendWithRetry failed: %v", err)
	}
}

func TestSendWithRetry_ServerError(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	sender.retryBackoff = 10 * time.Millisecond // Speed up test
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)

	if err == nil {
		t.Error("Expected error after retries")
	}

	// Should attempt: initial + 3 retries = 4 total
	if attempts != 4 {
		t.Errorf("Expected 4 attempts (1 initial + 3 retries), got %d", attempts)
	}
}

func TestSendWithRetry_EventualSuccess(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	sender.retryBackoff = 10 * time.Millisecond
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts before success, got %d", attempts)
	}
}

func TestSendWithRetry_ClientError(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)

	if err == nil {
		t.Error("Expected error for client error")
	}

	// Should not retry on 4xx errors
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry on client error), got %d", attempts)
	}
}

func TestSendWithRetry_RateLimitRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	sender.retryBackoff = 10 * time.Millisecond
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)

	if err != nil {
		t.Fatalf("Expected success after rate limit retry, got error: %v", err)
	}

	if attempts < 2 {
		t.Errorf("Expected at least 2 attempts for rate limit, got %d", attempts)
	}
}

func TestSendWithRetry_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	sender.retryBackoff = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	payload := map[string]string{"test": "data"}
	err := sender.sendWithRetry(ctx, server.URL, payload)

	if err == nil {
		t.Error("Expected error for cancelled context")
	}

	if err != context.DeadlineExceeded {
		t.Logf("Got error: %v", err)
	}
}

func TestSendWithRetry_ExponentialBackoff(t *testing.T) {
	attempts := 0
	attemptTimes := make([]time.Time, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptTimes = append(attemptTimes, time.Now())
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	sender.retryBackoff = 50 * time.Millisecond
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	_ = sender.sendWithRetry(ctx, server.URL, payload)

	if len(attemptTimes) < 2 {
		t.Fatal("Not enough attempts to verify backoff")
	}

	// Verify exponential backoff
	// 1st retry: 50ms, 2nd retry: 100ms, 3rd retry: 200ms
	for i := 1; i < len(attemptTimes); i++ {
		delay := attemptTimes[i].Sub(attemptTimes[i-1])
		expectedMinDelay := sender.retryBackoff * time.Duration(1<<uint(i-1))

		// Allow some margin for timing variance
		if delay < expectedMinDelay-10*time.Millisecond {
			t.Errorf("Retry %d: expected min delay %v, got %v", i, expectedMinDelay, delay)
		}
	}
}

func TestHTTPError_Error(t *testing.T) {
	err := &HTTPError{
		StatusCode: 500,
		Message:    "Internal Server Error",
	}

	expected := "HTTP 500: Internal Server Error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestIsRetryable_ServerErrors(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{429, true}, // Rate limit
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{422, false},
	}

	for _, tt := range tests {
		err := &HTTPError{StatusCode: tt.statusCode}
		result := isRetryable(err)

		if result != tt.retryable {
			t.Errorf("StatusCode %d: expected retryable=%v, got %v", tt.statusCode, tt.retryable, result)
		}
	}
}

func TestIsRetryable_NetworkErrors(t *testing.T) {
	// Network errors (not HTTPError) should be retryable
	err := &testError{msg: "network error"}
	if !isRetryable(err) {
		t.Error("Expected network errors to be retryable")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestSend_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		userAgent := r.Header.Get("User-Agent")
		if userAgent != "saviour-agent/1.0" {
			t.Errorf("Expected User-Agent 'saviour-agent/1.0', got '%s'", userAgent)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Authorization 'Bearer test-api-key', got '%s'", auth)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.send(ctx, server.URL, payload)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestSend_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Expected no Authorization header, got '%s'", auth)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL, "") // Empty API key
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.send(ctx, server.URL, payload)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestSend_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid request"))
	}))
	defer server.Close()

	sender := NewSender(server.URL, "test-api-key")
	ctx := context.Background()

	payload := map[string]string{"test": "data"}
	err := sender.send(ctx, server.URL, payload)

	if err == nil {
		t.Fatal("Expected error for bad request")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}

	if httpErr.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", httpErr.StatusCode)
	}

	if httpErr.Message != "Invalid request" {
		t.Errorf("Expected message 'Invalid request', got '%s'", httpErr.Message)
	}
}
