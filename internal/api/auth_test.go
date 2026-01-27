package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAuthConfig(t *testing.T) {
	keys := []APIKey{
		{Key: "key1", Name: "client1", Scopes: []string{"metrics:write"}},
		{Key: "key2", Name: "client2", Scopes: []string{"metrics:read"}},
	}

	config := NewAuthConfig(keys)

	if config == nil {
		t.Fatal("NewAuthConfig returned nil")
	}

	if len(config.APIKeys) != 2 {
		t.Errorf("Expected 2 API keys, got %d", len(config.APIKeys))
	}

	if _, exists := config.APIKeys["key1"]; !exists {
		t.Error("key1 not found in API keys map")
	}

	if _, exists := config.APIKeys["key2"]; !exists {
		t.Error("key2 not found in API keys map")
	}

	if config.APIKeys["key1"].Name != "client1" {
		t.Errorf("Expected client1, got %s", config.APIKeys["key1"].Name)
	}
}

func TestAuthMiddleware_ValidKey(t *testing.T) {
	keys := []APIKey{
		{Key: "test-key-123", Name: "test-client", Scopes: []string{"metrics:write"}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{"metrics:write"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer test-key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "success" {
		t.Errorf("Expected 'success', got %s", rec.Body.String())
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	keys := []APIKey{
		{Key: "test-key", Name: "test", Scopes: []string{"metrics:write"}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Error("Expected error message in body")
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"missing Bearer", "test-key-123"},
		{"wrong prefix", "Basic dGVzdDp0ZXN0"},
		{"extra spaces", "Bearer  test-key-123"},
		{"no key", "Bearer "},
		{"only Bearer", "Bearer"},
	}

	keys := []APIKey{
		{Key: "test-key-123", Name: "test", Scopes: []string{"metrics:write"}},
	}
	config := NewAuthConfig(keys)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := config.AuthMiddleware([]string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401 for %s, got %d", tt.name, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	keys := []APIKey{
		{Key: "valid-key", Name: "test", Scopes: []string{"metrics:write"}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InsufficientScopes(t *testing.T) {
	keys := []APIKey{
		{Key: "read-only-key", Name: "reader", Scopes: []string{"metrics:read"}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{"metrics:write"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer read-only-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoScopesRequired(t *testing.T) {
	keys := []APIKey{
		{Key: "test-key", Name: "test", Scopes: []string{}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MultipleScopes(t *testing.T) {
	keys := []APIKey{
		{Key: "admin-key", Name: "admin", Scopes: []string{"metrics:write", "metrics:read", "alerts:read"}},
	}
	config := NewAuthConfig(keys)

	handler := config.AuthMiddleware([]string{"metrics:write", "alerts:read"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer admin-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingOneScope(t *testing.T) {
	keys := []APIKey{
		{Key: "partial-key", Name: "partial", Scopes: []string{"metrics:write"}},
	}
	config := NewAuthConfig(keys)

	// Requires both metrics:write and alerts:read
	handler := config.AuthMiddleware([]string{"metrics:write", "alerts:read"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer partial-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

func TestHasScopes(t *testing.T) {
	config := NewAuthConfig([]APIKey{})

	tests := []struct {
		name           string
		keyScopes      []string
		requiredScopes []string
		expected       bool
	}{
		{
			name:           "exact match",
			keyScopes:      []string{"metrics:write"},
			requiredScopes: []string{"metrics:write"},
			expected:       true,
		},
		{
			name:           "has all required",
			keyScopes:      []string{"metrics:write", "metrics:read", "alerts:read"},
			requiredScopes: []string{"metrics:write", "alerts:read"},
			expected:       true,
		},
		{
			name:           "missing one scope",
			keyScopes:      []string{"metrics:read"},
			requiredScopes: []string{"metrics:write"},
			expected:       false,
		},
		{
			name:           "empty required",
			keyScopes:      []string{"metrics:write"},
			requiredScopes: []string{},
			expected:       true,
		},
		{
			name:           "empty key scopes",
			keyScopes:      []string{},
			requiredScopes: []string{"metrics:write"},
			expected:       false,
		},
		{
			name:           "both empty",
			keyScopes:      []string{},
			requiredScopes: []string{},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.hasScopes(tt.keyScopes, tt.requiredScopes)
			if result != tt.expected {
				t.Errorf("hasScopes(%v, %v) = %v, want %v", tt.keyScopes, tt.requiredScopes, result, tt.expected)
			}
		})
	}
}

func TestCORSMiddleware_DevMode(t *testing.T) {
	config := &CORSConfig{
		DevMode:        true,
		AllowedOrigins: []string{},
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected wildcard CORS in dev mode, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	config := &CORSConfig{
		DevMode:        false,
		AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Vary") != "Origin" {
		t.Error("Expected Vary: Origin header")
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	config := &CORSConfig{
		DevMode:        false,
		AllowedOrigins: []string{"https://example.com"},
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Disallowed origin should not get CORS header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	config := &CORSConfig{
		DevMode:        false,
		AllowedOrigins: []string{"https://example.com"},
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Should not set CORS header when no Origin header present")
	}
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	config := &CORSConfig{
		DevMode: true,
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/metrics", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS should return 200, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Should set Allow-Methods header")
	}

	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Should set Allow-Headers header")
	}
}

func TestCORSMiddleware_HeadersPresent(t *testing.T) {
	config := &CORSConfig{
		DevMode: true,
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("Allow-Methods header missing")
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("Allow-Headers header missing")
	}
}

func TestIsAllowedOrigin(t *testing.T) {
	allowedOrigins := []string{
		"https://example.com",
		"https://app.example.com",
		"http://localhost:3000",
	}

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://example.com", true},
		{"https://app.example.com", true},
		{"http://localhost:3000", true},
		{"https://evil.com", false},
		{"https://example.com.evil.com", false},
		{"http://example.com", false}, // Different protocol
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result := isAllowedOrigin(tt.origin, allowedOrigins)
			if result != tt.expected {
				t.Errorf("isAllowedOrigin(%s) = %v, want %v", tt.origin, result, tt.expected)
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "success" {
		t.Errorf("Expected 'success', got %s", rec.Body.String())
	}
}

func TestMiddlewareChaining(t *testing.T) {
	// Test that middleware can be chained properly
	keys := []APIKey{
		{Key: "test-key", Name: "test", Scopes: []string{"metrics:write"}},
	}
	authConfig := NewAuthConfig(keys)
	corsConfig := &CORSConfig{DevMode: true}

	handler := LoggingMiddleware(
		CORSMiddleware(corsConfig)(
			authConfig.AuthMiddleware([]string{"metrics:write"})(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("chained"))
				}),
			),
		),
	)

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "chained" {
		t.Errorf("Expected 'chained', got %s", rec.Body.String())
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set in chain")
	}
}
