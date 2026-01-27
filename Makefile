.PHONY: help build run clean test deps docker-build docker-run docker-stop docker-clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $1, $2}'

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

# Docker targets

docker-build: ## Build Docker image
	docker build -t saviour-agent:latest .

docker-run: docker-build ## Build and run with docker-compose
	docker-compose up -d

docker-stop: ## Stop Docker containers
	docker-compose down

docker-clean: docker-stop ## Stop containers and remove images
	docker rmi saviour-agent:latest || true

docker-logs: ## Show Docker container logs
	docker-compose logs -f agent

docker-shell: ## Open shell in running container (for debugging)
	docker exec -it saviour-agent /bin/sh || echo "Container not running or shell not available"

docker-rebuild: docker-clean docker-build ## Clean and rebuild Docker image
