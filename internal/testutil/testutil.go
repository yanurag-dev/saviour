package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockHTTPServer creates a test HTTP server with custom handler
func MockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// WaitForCondition waits for a condition to be true within a timeout
func WaitForCondition(condition func() bool, timeout time.Duration, checkInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}

// AssertEventually asserts that a condition becomes true within a timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	if !WaitForCondition(condition, timeout, 10*time.Millisecond) {
		if len(msgAndArgs) > 0 {
			t.Fatalf("Condition not met within %v: %v", timeout, msgAndArgs[0])
		} else {
			t.Fatalf("Condition not met within %v", timeout)
		}
	}
}

// FixedTime returns a fixed time for consistent testing
func FixedTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

// MockTime provides a controllable time source for testing
type MockTime struct {
	current time.Time
}

// NewMockTime creates a new mock time starting at the given time
func NewMockTime(start time.Time) *MockTime {
	return &MockTime{current: start}
}

// Now returns the current mock time
func (m *MockTime) Now() time.Time {
	return m.current
}

// Advance advances the mock time by the given duration
func (m *MockTime) Advance(d time.Duration) {
	m.current = m.current.Add(d)
}

// Set sets the mock time to a specific value
func (m *MockTime) Set(t time.Time) {
	m.current = t
}
