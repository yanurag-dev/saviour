package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/anurag/saviour/internal/agent"
	"github.com/anurag/saviour/internal/config"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "agent.yaml", "path to configuration file")
	flag.Parse()

	// Set up logger
	logger := log.New(os.Stdout, "[saviour-agent] ", log.LstdFlags)

	// Load configuration
	logger.Printf("Loading configuration from: %s", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}

	// Create agent
	a, err := agent.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Run agent
	logger.Println("Starting Saviour Agent...")
	if err := a.Run(ctx); err != nil && err != context.Canceled {
		logger.Fatalf("Agent error: %v", err)
	}

	logger.Println("Agent stopped")
}
