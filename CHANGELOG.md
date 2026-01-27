# Changelog

All notable changes to Saviour will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Phase 4 (Planned)
- Web dashboard with HTMX/React
- Multi-agent overview
- Container search and filtering
- Alert history and acknowledgment
- Historical metrics with optional SQLite
- High availability support
- Additional notification channels (Slack, PagerDuty, Email)

---

## [0.3.0] - 2026-01-28

### Phase 3: Central Server & Alerting - **COMPLETE** ✅

#### Added
- **Central Server** (`cmd/server/main.go`)
  - HTTP API server for metrics ingestion
  - POST `/api/v1/metrics/push` - Receive metrics from agents
  - POST `/api/v1/heartbeat` - Heartbeat tracking
  - GET `/api/v1/health` - Health check endpoint
  
- **In-Memory State Store** (`internal/server/state.go`)
  - Thread-safe state management with RWMutex
  - Agent state tracking (online/offline)
  - Container state change detection
  - Alert storage and management
  - Deep copy snapshots to prevent data races

- **Alert Detection Engine** (`internal/alerting/engine.go`)
  - Periodic check loop (configurable interval)
  - System alerts: CPU, memory, disk thresholds
  - Container alerts: stopped, unhealthy, high resources
  - Offline agent detection via heartbeat timeout
  - Alert deduplication with time-based window
  - State-change detection for containers

- **Google Chat Integration** (`internal/alerting/googlechat.go`)
  - Rich card-based messages
  - Severity-based icons (🚨 critical, ⚠️ warning, ℹ️ info)
  - Dashboard link buttons
  - Thread grouping for related alerts
  - Console notifier for testing

- **Authentication Middleware** (`internal/api/auth.go`)
  - Bearer token authentication
  - Scope-based permissions (metrics:write, heartbeat:write, etc.)
  - Multiple API key support
  - Configurable CORS with whitelist
  - Request logging

- **Security Hardening**
  - Request size limits (10MB max)
  - Gzip bomb protection with `http.MaxBytesReader`
  - CORS whitelist (no wildcards in production)
  - Configuration validation (fail-fast on invalid config)
  - Ticker validation (prevents panic on invalid durations)

- **EC2 Metadata Integration** (`internal/agent/ec2.go`)
  - IMDSv2 client for AWS metadata
  - Auto-detects EC2 environment
  - Fetches instance ID, type, region, AZ
  - Graceful fallback if not on EC2

- **Comprehensive Documentation**
  - Complete README.md rewrite (12KB)
  - USER_GUIDE.md with installation to troubleshooting (24KB)
  - ARCHITECTURE.md (system design)
  - DEPLOYMENT.md (production setup)
  - This CHANGELOG.md

#### Changed
- State store getters now return deep copies (thread-safety)
- UpdateAgent preserves ActiveAlerts across updates
- CORS middleware now configurable (dev/prod modes)
- Server configuration enhanced with validation

#### Fixed
- Payload structure mismatch between agent and server
- Data races in state store (concurrent access)
- Alert engine panic on invalid CheckInterval
- ActiveAlerts being dropped on metric updates

#### Security
- ✅ No wildcard CORS in production
- ✅ DoS protection (request size limits)
- ✅ Thread-safe state access
- ✅ Config validation
- ✅ No panic conditions

---

## [0.2.0] - 2026-01-28

### Phase 2: Agent Push Mechanism - **COMPLETE** ✅

#### Added
- **HTTP Sender** (`internal/agent/sender.go`)
  - HTTPS POST to central server
  - Bearer token authentication
  - Gzip compression for payloads >1KB
  - Exponential backoff retry logic (3 attempts)
  - Configurable timeouts and intervals
  
- **Push Configuration**
  - `push_interval`: How often to push metrics (default: 20s)
  - `heartbeat_interval`: Heartbeat frequency (default: 10s)
  - `push_timeout`: HTTP request timeout (default: 10s)
  - `retry_attempts`: Max retries (default: 3)
  - `retry_backoff`: Initial backoff (default: 2s)

- **Multiple Tickers**
  - Collection ticker (15s) - collect metrics
  - Push ticker (20s) - send to server
  - Heartbeat ticker (10s) - keep-alive signal
  - Independent operation (no blocking)

- **Mock Server** (`cmd/mockserver/main.go`)
  - Testing endpoint for agent development
  - Logs received metrics and heartbeats

- **Progress Tracking** (`PROGRESS.md`)
  - Detailed development progress
  - Phase completion tracking
  - Feature checklist

#### Changed
- Agent now pushes metrics instead of console output
- Metrics include timestamp and agent name
- Separate heartbeat mechanism

#### Performance
- Compression reduces payload: 140KB → ~14KB (10x reduction)
- Configurable intervals for different use cases
- Retry logic prevents data loss on network issues

---

## [0.1.0] - 2026-01-27

### Phase 1: Docker Container Monitoring - **COMPLETE** ✅

#### Added
- **Docker SDK Integration**
  - Container discovery and filtering
  - Per-container metrics collection
  - Health status tracking
  - OOM detection

- **Container Metrics** (`internal/docker/client.go`)
  - Identity: ID, name, image, labels
  - State: running, exited, paused, restarting
  - Health: healthy, unhealthy, starting, none
  - Resources: CPU %, memory usage/limit/%, network I/O, disk I/O
  - Lifecycle: created, started, exit code, OOM killed
  - Reliability: restart count, uptime, PIDs

- **Container Filtering**
  - Monitor all containers (default)
  - Filter by Docker labels
  - Filter by container names (glob patterns)
  - Filter by image names (glob patterns)
  - Exclude stopped containers (optional)

- **Docker-Specific Alerts**
  - Container stopped (running → exited)
  - Container unhealthy
  - OOM kill detection
  - CPU threshold per container
  - Memory threshold per container
  - Restart count threshold

- **Pattern Matching** (`internal/agent/agent.go`)
  - Glob pattern support (*, ?) for container names
  - Per-container alert overrides
  - Example: "api-*" matches api-v1, api-v2, api-gateway

- **Configuration Enhancements**
  - `monitor_all` defaults to true when no filters specified
  - Docker socket configurable (default: /var/run/docker.sock)
  - Container alert overrides with pattern matching

- **Timeout & Cleanup**
  - 5-second timeout for Docker daemon ping
  - Proper client cleanup on initialization failure
  - Resource leak prevention

- **Partial Results**
  - GetAllContainerInfo returns partial data on errors
  - First error captured and returned
  - Resilient: one failed container doesn't block others

#### Changed
- Go version updated to 1.24
- Enhanced configuration structure
- Improved error handling in agent initialization
- PROJECT_DESIGN.md updated with current scope

---

## [0.0.1] - 2026-01-27

### Phase 0: Project Foundation - **COMPLETE** ✅

#### Added
- **Project Structure**
  - `cmd/` - Agent, server, CLI entry points
  - `internal/` - Core business logic
  - `pkg/` - Shared libraries
  - `examples/` - Configuration examples

- **System Metrics Collection** (`internal/collector/system.go`)
  - CPU: Usage per core, load average, utilization percentages
  - Memory: Total, used, available, swap usage
  - Disk: Usage per mount point, I/O statistics
  - Network: Bandwidth usage, packet statistics
  - System Info: Hostname, OS, platform, uptime

- **Configuration Management** (`internal/config/config.go`)
  - YAML-based configuration
  - Environment variable support
  - Sensible defaults
  - Validation on load

- **Local Alert Thresholds**
  - CPU, memory, disk threshold detection
  - Console output for exceeded thresholds

- **Agent Main Loop**
  - Ticker-based collection cycle
  - Configurable intervals
  - Graceful shutdown

- **Docker Support**
  - Dockerfile for agent
  - docker-compose.yml for easy deployment
  - Multi-stage builds

- **Build Automation**
  - Makefile with common tasks
  - Build, run, docker commands

- **Documentation**
  - README.md - Project overview
  - PROJECT_DESIGN.md - Detailed design document

#### Dependencies
- `github.com/shirou/gopsutil/v3` - System metrics
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/docker/docker` - Docker SDK (Phase 1)

---

## Version History

| Version | Date | Phase | Status |
|---------|------|-------|--------|
| 0.3.0 | 2026-01-28 | Phase 3 | ✅ Complete |
| 0.2.0 | 2026-01-28 | Phase 2 | ✅ Complete |
| 0.1.0 | 2026-01-27 | Phase 1 | ✅ Complete |
| 0.0.1 | 2026-01-27 | Phase 0 | ✅ Complete |

**Current Version**: v0.3.0  
**Current Status**: 75% of MVP Complete (3/4 phases)  
**Next Release**: v0.4.0 (Phase 4 - Web Dashboard)

---

## Upgrade Guide

### From 0.2.0 to 0.3.0

#### Breaking Changes
- None - backward compatible

#### New Requirements
- Server must be deployed
- Agents must be configured with `server_url` and `api_key`

#### Migration Steps

1. **Deploy Central Server**
   ```bash
   # Copy server binary
   cp bin/saviour-server /usr/local/bin/
   
   # Create server config
   cp examples/server.yaml /etc/saviour/server.yaml
   vim /etc/saviour/server.yaml  # Configure API keys
   
   # Start server
   systemctl start saviour-server
   ```

2. **Update Agent Configuration**
   ```yaml
   # Add to agent.yaml
   agent:
     server_url: "http://your-server:8080"
     api_key: "your-api-key-here"
     push_interval: 20s
     heartbeat_interval: 10s
   ```

3. **Restart Agents**
   ```bash
   systemctl restart saviour-agent
   ```

4. **Verify Connection**
   ```bash
   curl http://your-server:8080/api/v1/health
   # Should show agents_online: N
   ```

---

## Contributors

- **Anurag Yadav** - Initial work and core development

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
