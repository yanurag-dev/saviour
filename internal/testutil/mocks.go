package testutil

import (
	"context"
	"sync"
	"time"
)

// MockDockerClient provides a mock Docker client for testing
type MockDockerClient struct {
	mu         sync.RWMutex
	containers []MockContainer
	err        error
}

// MockContainer represents a mock container
type MockContainer struct {
	ID       string
	Name     string
	Image    string
	State    string
	Health   string
	CPUUsage float64
	MemUsage uint64
	MemLimit uint64
}

// NewMockDockerClient creates a new mock Docker client
func NewMockDockerClient() *MockDockerClient {
	return &MockDockerClient{
		containers: make([]MockContainer, 0),
	}
}

// AddContainer adds a mock container
func (m *MockDockerClient) AddContainer(c MockContainer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.containers = append(m.containers, c)
}

// SetError sets an error to be returned by the mock
func (m *MockDockerClient) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// GetContainers returns the mock containers
func (m *MockDockerClient) GetContainers(ctx context.Context) ([]MockContainer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.containers, nil
}

// MockNotifier provides a mock notifier for testing alerts
type MockNotifier struct {
	mu           sync.RWMutex
	notifications []Notification
	shouldFail   bool
}

// Notification represents a sent notification
type Notification struct {
	AlertType string
	Severity  string
	Message   string
	SentAt    time.Time
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		notifications: make([]Notification, 0),
	}
}

// Send sends a mock notification
func (m *MockNotifier) Send(alertType, severity, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return &MockError{Message: "mock notifier error"}
	}

	m.notifications = append(m.notifications, Notification{
		AlertType: alertType,
		Severity:  severity,
		Message:   message,
		SentAt:    time.Now(),
	})
	return nil
}

// GetNotifications returns all sent notifications
func (m *MockNotifier) GetNotifications() []Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

// SetShouldFail sets whether the notifier should fail
func (m *MockNotifier) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// Reset clears all notifications
func (m *MockNotifier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = make([]Notification, 0)
}

// MockError represents a mock error
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}
