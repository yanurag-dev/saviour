package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/anurag/saviour/internal/alerting"
	"github.com/anurag/saviour/internal/api"
	"github.com/anurag/saviour/internal/server"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "server.yaml", "Path to server configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := server.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting Saviour Server on %s", cfg.Address())

	// Initialize state store
	state := server.NewStateStore()

	// Initialize notifier
	var notifier alerting.Notifier
	if cfg.GoogleChat.Enabled {
		log.Printf("Google Chat notifications enabled")
		notifier = alerting.NewGoogleChatNotifier(cfg.GoogleChat.WebhookURL, cfg.GoogleChat.DashboardURL)
	} else {
		log.Printf("Using console notifier (Google Chat disabled)")
		notifier = alerting.NewConsoleNotifier()
	}

	// Create adapter for alerting
	stateAdapter := server.NewAlertingAdapter(state)

	// Convert alerting config
	alertConfig := &alerting.Config{
		Enabled:               cfg.Alerting.Enabled,
		CheckInterval:         cfg.Alerting.CheckInterval,
		HeartbeatTimeout:      cfg.Alerting.HeartbeatTimeout,
		DeduplicationEnabled:  cfg.Alerting.DeduplicationEnabled,
		DeduplicationWindow:   cfg.Alerting.DeduplicationWindow,
		SystemCPUThreshold:    cfg.Alerting.SystemCPUThreshold,
		SystemMemoryThreshold: cfg.Alerting.SystemMemoryThreshold,
		SystemDiskThreshold:   cfg.Alerting.SystemDiskThreshold,
	}

	// Initialize alert engine
	alertEngine := alerting.NewEngine(stateAdapter, alertConfig, notifier)

	// Start alert engine in background
	go alertEngine.Start()

	// Initialize API handler
	handler := api.NewHandler(state)

	// Convert API keys
	apiKeys := make([]api.APIKey, len(cfg.Auth.APIKeys))
	for i, k := range cfg.Auth.APIKeys {
		apiKeys[i] = api.APIKey{
			Key:    k.Key,
			Name:   k.Name,
			Scopes: k.Scopes,
		}
	}

	// Set up authentication
	authConfig := api.NewAuthConfig(apiKeys)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Metrics endpoints (require metrics:write scope)
	metricsAuth := authConfig.AuthMiddleware([]string{"metrics:write"})
	mux.Handle("/api/v1/metrics/push", metricsAuth(http.HandlerFunc(handler.HandleMetricsPush)))

	// Heartbeat endpoint (require heartbeat:write scope)
	heartbeatAuth := authConfig.AuthMiddleware([]string{"heartbeat:write"})
	mux.Handle("/api/v1/heartbeat", heartbeatAuth(http.HandlerFunc(handler.HandleHeartbeat)))

	// Health endpoint (no auth required)
	mux.HandleFunc("/api/v1/health", handler.HandleHealth)

	// Dashboard API endpoints (no auth required for now - can add read scope later)
	mux.HandleFunc("/api/v1/agents", handler.HandleGetAgents)
	mux.HandleFunc("/api/v1/agents/", handler.HandleGetAgent)
	mux.HandleFunc("/api/v1/alerts", handler.HandleGetAlerts)
	mux.HandleFunc("/api/v1/events", handler.HandleEventsSSE)

	// Serve static files from web/dist (if exists)
	fileServer := http.FileServer(http.Dir("./web/dist"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes are already handled above
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Serve index.html for root and index
			http.ServeFile(w, r, "./web/dist/index.html")
			return
		}

		// Check if requesting an API endpoint
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			// API route not found
			http.NotFound(w, r)
			return
		}

		// Try serving static file, fallback to index.html for SPA routing
		fileServer.ServeHTTP(w, r)
	})

	// Apply middleware
	var finalHandler http.Handler = mux

	// Apply CORS middleware if enabled
	if cfg.CORS.Enabled {
		corsConfig := &api.CORSConfig{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			DevMode:        cfg.CORS.DevMode,
		}
		finalHandler = api.CORSMiddleware(corsConfig)(finalHandler)
		if cfg.CORS.DevMode {
			log.Println("CORS enabled in development mode (allowing all origins)")
		} else {
			log.Printf("CORS enabled with allowed origins: %v", cfg.CORS.AllowedOrigins)
		}
	}

	// Apply logging middleware
	finalHandler = api.LoggingMiddleware(finalHandler)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Address(),
		Handler: finalHandler,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		if err := httpServer.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	// Start server
	log.Printf("Server listening on %s", cfg.Address())
	log.Printf("Endpoints:")
	log.Printf("  POST /api/v1/metrics/push  - Receive metrics from agents")
	log.Printf("  POST /api/v1/heartbeat     - Receive heartbeat from agents")
	log.Printf("  GET  /api/v1/health        - Health check")
	log.Printf("  GET  /api/v1/agents        - List all agents")
	log.Printf("  GET  /api/v1/agents/:name  - Get specific agent")
	log.Printf("  GET  /api/v1/alerts        - List all alerts")
	log.Printf("  GET  /api/v1/events        - Server-Sent Events stream")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("Server stopped")
}
