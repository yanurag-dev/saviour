.PHONY: help build run clean test deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the agent binary
	@mkdir -p bin
	go build -o bin/saviour-agent ./cmd/agent

run: build ## Build and run the agent with example config
	./bin/saviour-agent -config examples/agent.yaml

dev: build ## Build and run with test config
	./bin/saviour-agent -config agent.yaml

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f *.log

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Run formatting and vetting

all: deps lint build ## Download deps, lint, and build
