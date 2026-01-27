package api

import (
	"log"
	"net/http"
	"strings"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	APIKeys map[string]APIKey // key: api_key_string, value: APIKey details
}

// APIKey represents an API key with permissions
type APIKey struct {
	Key    string   `json:"key"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// NewAuthConfig creates a new auth configuration
func NewAuthConfig(keys []APIKey) *AuthConfig {
	keyMap := make(map[string]APIKey)
	for _, key := range keys {
		keyMap[key.Key] = key
	}
	return &AuthConfig{
		APIKeys: keyMap,
	}
}

// AuthMiddleware validates API key from Authorization header
func (ac *AuthConfig) AuthMiddleware(requiredScopes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Printf("Missing Authorization header from %s", r.RemoteAddr)
				http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Printf("Invalid Authorization header format from %s", r.RemoteAddr)
				http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			apiKey := parts[1]

			// Validate API key
			key, valid := ac.APIKeys[apiKey]
			if !valid {
				log.Printf("Invalid API key from %s", r.RemoteAddr)
				http.Error(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
				return
			}

			// Check scopes if required
			if len(requiredScopes) > 0 && !ac.hasScopes(key.Scopes, requiredScopes) {
				log.Printf("Insufficient permissions for %s (key: %s)", r.RemoteAddr, key.Name)
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add API key name to request context for logging
			log.Printf("Authenticated request from %s (key: %s)", r.RemoteAddr, key.Name)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// hasScopes checks if the key has all required scopes
func (ac *AuthConfig) hasScopes(keyScopes, requiredScopes []string) bool {
	scopeMap := make(map[string]bool)
	for _, scope := range keyScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}
	return true
}

// CORS middleware for development
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
