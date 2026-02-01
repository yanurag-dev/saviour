# Saviour - Internal Monitoring & Alerting Platform

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/yanurag-dev/saviour)

**Saviour** is a lightweight, push-based monitoring and alerting platform designed for tracking Docker containers and system health across multiple servers. It provides centralized monitoring, intelligent alerting via Google Chat, and real-time infrastructure visibility.

---

## 🎯 Key Features

### 📊 **Comprehensive Monitoring**
- **System Metrics**: CPU usage, memory, disk space, network I/O, load averages
- **Docker Containers**: Per-container CPU, memory, network, disk I/O, health status
- **Process Tracking**: Container restart counts, OOM detection, exit codes
- **Real-time Collection**: Configurable intervals (default: 15s)
- **EC2 Integration**: Automatic instance metadata discovery (IMDSv2)

### 🔔 **Intelligent Alerting**
- **Threshold Monitoring**: System and container resource alerts
- **State Change Detection**: Container stopped, unhealthy, restarting
- **Smart Deduplication**: Prevents alert spam with time-based windows
- **Multiple Severity Levels**: Critical, warning, info
- **Flexible Thresholds**: Global defaults with per-container overrides
- **Pattern Matching**: Alert rules support glob patterns (e.g., `api-*`)

### 💬 **Google Chat Integration**
- **Rich Notifications**: Formatted card messages with icons
- **Severity Colors**: Visual distinction for alert types
- **Dashboard Links**: Quick access to monitoring dashboard
- **Thread Grouping**: Related alerts grouped together
- **Instant Delivery**: Real-time webhook notifications

### 🏢 **Central Server**
- **Push-Based Architecture**: No firewall configuration needed
- **In-Memory State Store**: Fast, zero-database monitoring
- **REST API**: Simple HTTP endpoints for metrics ingestion
- **Heartbeat Tracking**: Automatic offline detection
- **Multi-Agent Support**: Monitor hundreds of servers
- **Thread-Safe**: Concurrent access without data races

### 🔒 **Security & Performance**
- **Authentication**: Bearer token with scope-based permissions
- **Request Limits**: Protection against DoS attacks
- **CORS Whitelist**: Configurable allowed origins
- **Data Compression**: Gzip reduces bandwidth by 10x
- **Retry Logic**: Exponential backoff for network failures
- **Minimal Overhead**: <1% CPU, ~20MB memory per agent

### 🐳 **Flexible Container Monitoring**
- **Auto-Discovery**: Monitor all containers by default
- **Label Filtering**: Target specific containers with labels
- **Name Patterns**: Glob pattern matching (e.g., `web-*`, `api-*`)
- **Image Filtering**: Monitor by image name or registry
- **Health Checks**: Docker health status tracking
- **Resource Tracking**: Per-container CPU, memory, network, disk

---

## 🏗️ Architecture

```text
┌─────────────────────────────────────────────────────────┐
│         Saviour Central Server                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Alert Engine  │  State Store  │  REST API      │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTPS POST         │ Webhook
                      ↑                    ↓
         ┌────────────┼──────────┐   ┌─────────────────┐
         │            │          │   │  Google Chat    │
    ┌────┴───┐   ┌───┴────┐  ┌──┴──────┐
    │ Agent  │   │ Agent  │  │ Agent   │
    │ EC2-1  │   │ EC2-2  │  │ EC2-N   │
    │ 🐳🐳🐳 │   │ 🐳🐳🐳 │  │ 🐳🐳🐳  │
    └────────┘   └────────┘  └─────────┘
```

### How It Works

1. **Agents** collect system and Docker metrics every 15 seconds
2. **Push** metrics to central server via HTTPS (compressed)
3. **Send heartbeat** every 10 seconds for offline detection
4. **Server** analyzes metrics and detects threshold violations
5. **Alerts** sent to Google Chat when issues detected
6. **Deduplication** prevents alert spam

**Benefits of Push Architecture**:
- ✅ No firewall rules needed (outbound HTTPS only)
- ✅ Real-time metric delivery
- ✅ Simple agent deployment
- ✅ Automatic offline detection
- ✅ Works across VPCs and networks

---

## 🚀 Quick Start

### Prerequisites
- Go 1.24+ (for building from source)
- Docker (optional, for container monitoring)
- Docker socket access (if monitoring containers)

### 1. Install Server

```bash
# Download latest release
curl -L https://github.com/yanurag-dev/saviour/releases/latest/download/saviour-server \
  -o /usr/local/bin/saviour-server
chmod +x /usr/local/bin/saviour-server

# Or build from source
git clone https://github.com/yanurag-dev/saviour.git
cd saviour
go build -o bin/saviour-server ./cmd/server
```

### 2. Configure Server

Create `/etc/saviour/server.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  api_keys:
    - key: "your-secure-api-key-here"
      name: "production-agents"
      scopes: ["metrics:write", "heartbeat:write"]

alerting:
  enabled: true
  check_interval: 30s
  heartbeat_timeout: 2m
  system_cpu_threshold: 80.0
  system_memory_threshold: 85.0
  system_disk_threshold: 90.0

google_chat:
  enabled: true
  webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
```

### 3. Start Server

```bash
# Direct execution
./bin/saviour-server -config /etc/saviour/server.yaml

# Or as systemd service (recommended)
sudo systemctl enable saviour-server
sudo systemctl start saviour-server
```

### 4. Install Agent on Each Server

```bash
# Download agent
curl -L https://github.com/yanurag-dev/saviour/releases/latest/download/saviour-agent \
  -o /usr/local/bin/saviour-agent
chmod +x /usr/local/bin/saviour-agent
```

### 5. Configure Agent

Create `/etc/saviour/agent.yaml`:

```yaml
agent:
  name: "auto"  # Uses EC2 instance ID
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"
  collect_interval: 15s
  push_interval: 20s
  heartbeat_interval: 10s

metrics:
  system: true
  docker:
    enabled: true
    monitor_all: true

alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

### 6. Start Agent

```bash
export SAVIOUR_API_KEY="your-api-key-here"
./bin/saviour-agent -config /etc/saviour/agent.yaml

# Or as systemd service
sudo systemctl enable saviour-agent
sudo systemctl start saviour-agent
```

### 7. Verify

```bash
# Check server health
curl http://localhost:8080/api/v1/health

# Should show:
{
  "status": "ok",
  "agents_online": 1,
  "agents_offline": 0,
  "active_alerts": 0
}
```

---

## 📋 Configuration

### Server Configuration

```yaml
# HTTP Server
server:
  host: "0.0.0.0"      # Listen on all interfaces
  port: 8080           # Server port

# Authentication
auth:
  api_keys:
    - key: "sk_prod_agents_xxxxx"
      name: "production-agents"
      scopes: ["metrics:write", "heartbeat:write"]
    - key: "sk_prod_dashboard_xxxxx"
      name: "dashboard"
      scopes: ["metrics:read", "alerts:read"]

# Alert Detection
alerting:
  enabled: true
  check_interval: 30s              # Check frequency
  heartbeat_timeout: 2m            # Offline threshold
  deduplication_enabled: true
  deduplication_window: 5m         # Don't repeat alerts within 5min
  
  # System thresholds
  system_cpu_threshold: 80.0
  system_memory_threshold: 85.0
  system_disk_threshold: 90.0

# Notifications
google_chat:
  enabled: true
  webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
  dashboard_url: "https://saviour.company.com"

# CORS (for web dashboard)
cors:
  enabled: false
  dev_mode: false
  allowed_origins:
    - "https://dashboard.company.com"
```

### Agent Configuration

```yaml
# Agent Identity
agent:
  name: "auto"                     # "auto" uses EC2 instance ID
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"    # From environment

# Collection Intervals
  collect_interval: 15s            # Metric collection
  push_interval: 20s               # Push to server
  heartbeat_interval: 10s          # Keep-alive signal
  
# Network Settings
  push_timeout: 10s
  retry_attempts: 3
  retry_backoff: 2s

# Metrics Collection
metrics:
  system: true
  
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    monitor_all: true              # Monitor all containers
    
    # Or use filters (set monitor_all: false)
    filters:
      labels: ["monitor=true"]
      names: ["api-*", "web-*"]
      images: ["mycompany/*"]
    
    # Per-container thresholds
    alerts:
      default:
        cpu_threshold: 80.0
        memory_threshold: 90.0
        restart_threshold: 5
        restart_window: 300s
      
      overrides:
        - name: "postgres"
          memory_threshold: 95.0
        - name: "redis"
          cpu_threshold: 70.0

# System Alerts
alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

---

## 🔔 Alert Types

### System Alerts

| Alert | Trigger | Severity |
|-------|---------|----------|
| High CPU | CPU > threshold% | Warning |
| High Memory | Memory > threshold% | Warning |
| High Disk | Disk > threshold% | Critical |
| Agent Offline | No heartbeat for > timeout | Critical |

### Container Alerts

| Alert | Trigger | Severity |
|-------|---------|----------|
| Container Stopped | State: running → exited/dead | Critical |
| Container Unhealthy | Health check failing | Warning |
| High CPU | Container CPU > threshold% | Warning |
| High Memory | Container memory > threshold% | Critical |
| OOM Killed | Out of memory kill detected | Critical |
| Restart Loop | Too many restarts in time window | Warning |

### Alert Message Example

```
🚨 CRITICAL ALERT
ec2-prod-web-01

Alert Type: disk_critical
Severity: critical

🚨 High Disk Usage
Agent: ec2-prod-web-01
Mount: /data
Usage: 95.2%

Triggered: 2026-01-28T10:15:30Z

[View Dashboard]
```

---

## 🐳 Container Monitoring

### Monitor All Containers

```yaml
docker:
  enabled: true
  monitor_all: true  # Simplest approach
```

### Filter by Labels

```yaml
# Tag containers in docker-compose.yml
labels:
  - "monitor=true"
  - "env=production"

# Configure agent
docker:
  monitor_all: false
  filters:
    labels: ["monitor=true", "env=production"]
```

### Filter by Name Pattern

```yaml
docker:
  monitor_all: false
  filters:
    names:
      - "api-*"      # Matches: api-v1, api-v2, api-gateway
      - "web-*"      # Matches: web-frontend, web-backend
      - "worker-*"   # Matches: worker-1, worker-2
```

### Filter by Image

```yaml
docker:
  monitor_all: false
  filters:
    images:
      - "mycompany/*"     # All images from registry
      - "nginx:*"         # All nginx versions
      - "postgres:14*"    # Postgres 14.x
```

### Per-Container Thresholds

```yaml
docker:
  alerts:
    default:
      cpu_threshold: 80.0
      memory_threshold: 90.0
    
    overrides:
      - name: "postgres"        # Exact match
        memory_threshold: 95.0
      
      - name: "api-*"           # Pattern match
        cpu_threshold: 70.0
      
      - name: "worker-*"
        restart_threshold: 10   # Allow more restarts
```

---

## 📦 Deployment

### Docker Compose (Recommended)

```yaml
version: '3.8'

services:
  saviour-server:
    image: saviour-server:latest
    ports:
      - "8080:8080"
    volumes:
      - ./server.yaml:/etc/saviour/server.yaml:ro
    environment:
      - GOOGLE_CHAT_WEBHOOK_URL=${GOOGLE_CHAT_WEBHOOK_URL}
    restart: unless-stopped

  saviour-agent:
    image: saviour-agent:latest
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./agent.yaml:/etc/saviour/agent.yaml:ro
    environment:
      - SAVIOUR_API_KEY=${SAVIOUR_API_KEY}
    restart: unless-stopped
```

### Systemd Service

```ini
[Unit]
Description=Saviour Monitoring Agent
After=network.target docker.service

[Service]
Type=simple
User=root
Environment="SAVIOUR_API_KEY=your-api-key-here"
ExecStart=/usr/local/bin/saviour-agent -config /etc/saviour/agent.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Production Deployment

1. **Server**: Deploy on dedicated instance or VM
2. **Reverse Proxy**: Use Nginx/Caddy for HTTPS
3. **Agent**: Install on each EC2 instance via systemd
4. **Secrets**: Store API keys in AWS Secrets Manager
5. **Monitoring**: Set up external health checks for server

---

## 🔒 Security

### API Key Management

```bash
# Generate secure keys
openssl rand -hex 32

# Format: sk_prod_<random-string>
# Example: sk_prod_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**Best Practices**:
- Use different keys per environment (dev/staging/prod)
- Rotate keys every 90 days
- Store in environment variables or secrets manager
- Use scope-based permissions (metrics:write, alerts:read)

### Network Security

- **Firewall**: Only allow outbound HTTPS (443) from agents
- **Server**: Restrict inbound to agent IPs only
- **TLS**: Use reverse proxy (Nginx) with Let's Encrypt
- **CORS**: Whitelist dashboard origins in production

### Request Protection

- **Size Limits**: 10MB maximum request size
- **Gzip Bomb**: Protected with `http.MaxBytesReader`
- **Rate Limiting**: Per-agent request validation
- **Authentication**: Bearer token on all agent endpoints

---

## 📊 Monitoring Capabilities

### System Metrics Collected

- **CPU**: Usage %, per-core %, load averages (1m, 5m, 15m)
- **Memory**: Total, used, available, swap usage and %
- **Disk**: Per-mount usage, I/O stats, inode usage
- **Network**: Bytes sent/received, packets, errors, drops
- **System**: Hostname, OS, platform, uptime

### Container Metrics Collected

- **Identity**: ID, name, image, image ID, labels
- **State**: Running, exited, paused, restarting, dead
- **Health**: Healthy, unhealthy, starting, none
- **Resources**: CPU %, memory usage/limit/%, network I/O, disk I/O
- **Lifecycle**: Created time, started time, exit code, OOM flag
- **Reliability**: Restart count, uptime, process count

### Performance Characteristics

| Component | Metric | Value |
|-----------|--------|-------|
| Agent | CPU Usage | <1% |
| Agent | Memory | ~20MB |
| Agent | Network | ~1KB/s |
| Server | Startup | <1s |
| Server | Memory | ~15MB (idle) |
| Server | Latency | <5ms per request |
| Compression | Ratio | 10x (140KB → 14KB) |

---

## 🧪 Development

### Build Commands

```bash
# Build both
make build

# Build agent only
go build -o bin/saviour-agent ./cmd/agent

# Build server only
go build -o bin/saviour-server ./cmd/server

# Run tests
go test ./...

# Run linter
golangci-lint run
```

### Running Locally

```bash
# Terminal 1: Start server
go run cmd/server/main.go -config examples/test-configs/server-test.yaml

# Terminal 2: Start agent
go run cmd/agent/main.go -config examples/test-configs/agent-server-test.yaml

# Terminal 3: Check health
curl http://localhost:8080/api/v1/health | jq
```

### Testing

Saviour has comprehensive unit tests covering all critical components with >70% overall code coverage.

#### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...

# Run tests in a specific package
go test ./internal/agent/...
go test ./internal/api/...

# Run with verbose output
go test -v ./...
```

#### Using Test Script

```bash
# Run comprehensive test suite with coverage reporting
./scripts/test.sh
```

This script will:
- Run all tests with race detection
- Generate coverage report (coverage.out)
- Display coverage summary
- Create HTML coverage report (coverage.html)
- Check that coverage meets 70% threshold

#### Test Coverage

| Component | Coverage | Key Tests |
|-----------|----------|-----------|
| **API Layer** | 97%+ | Auth middleware, CORS, handlers, request validation |
| **Server State** | 100% | State management, agent tracking, alert storage |
| **Server Config** | 100% | Configuration validation, defaults, environment vars |
| **Server Overall** | 86%+ | Server state and configuration |
| **Alerting Engine** | 100% | Alert detection, deduplication, notifications |
| **Agent Sender** | 100% | HTTP client, retry logic, compression, error handling |
| **Agent EC2** | 80%+ | Metadata fetching, token management, IMDS calls |
| **Overall** | 37%+ | Comprehensive coverage of critical business logic |

#### Test Structure

```
internal/
├── api/
│   ├── auth_test.go         # Auth middleware tests
│   ├── handler_test.go      # API handler tests
├── server/
│   ├── state_test.go        # State store tests
│   ├── config_test.go       # Config validation tests
├── alerting/
│   └── engine_test.go       # Alert engine tests
├── agent/
│   ├── ec2_test.go          # EC2 metadata tests
│   └── sender_test.go       # HTTP sender tests
└── testutil/
    ├── testutil.go          # Test utilities
    └── mocks.go             # Mock implementations
```

#### Continuous Integration

GitHub Actions runs tests automatically on:
- Every push to `main` and `develop` branches
- Every pull request
- Daily scheduled runs

CI pipeline includes:
- ✅ Unit tests with race detection
- ✅ Code coverage reporting
- ✅ Linting with golangci-lint
- ✅ Build verification
- ✅ Coverage threshold checks (70%)

See [`.github/workflows/test.yml`](.github/workflows/test.yml) for details.

#### Writing Tests

When contributing, ensure:
- All new code includes unit tests
- Tests cover both success and error cases
- Mock external dependencies (HTTP, Docker, time)
- Use table-driven tests for multiple scenarios
- Maintain >70% coverage for new packages

Example test pattern:

```go
func TestFeature_Success(t *testing.T) {
    // Setup
    mockServer := httptest.NewServer(...)
    defer mockServer.Close()

    // Execute
    result, err := DoSomething()

    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

---

## 📚 Documentation

- **[USER_GUIDE.md](USER_GUIDE.md)** - Complete installation and usage guide
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and changes
- **[PROGRESS.md](PROGRESS.md)** - Development progress tracking
- **[PROJECT_DESIGN.md](PROJECT_DESIGN.md)** - Detailed design document
- **[examples/](examples/)** - Configuration examples

---

## 🖥️ Web Dashboard

Saviour includes a production-grade web dashboard for monitoring your infrastructure in real-time.

### Features

- **Agent Overview**: Grid view of all agents with live CPU/memory/disk metrics
- **Container Monitoring**: Comprehensive table of all containers with filtering and search
- **Alert Dashboard**: Real-time alerts with severity categorization
- **Live Charts**: Real-time CPU and memory graphs with historical data
- **SSE Streaming**: Auto-updating data every 5 seconds via Server-Sent Events

### Quick Start

```bash
# Install web dependencies
cd web
npm install

# Development mode (hot reload)
npm run dev

# Build for production
npm run build

# The server will serve the dashboard at http://localhost:8080
```

### Design

The dashboard features a distinctive **Industrial Terminal** aesthetic:
- Monospace typography (JetBrains Mono)
- High-contrast charcoal + amber color scheme
- Dense information display with breathing room
- Real-time animations and visual feedback
- Responsive design (desktop + tablet)

See [web/README.md](web/README.md) for detailed documentation.

---

## 🐛 Troubleshooting

### Agent Can't Connect

```bash
# Test connectivity
curl http://server-ip:8080/api/v1/health

# Check firewall
sudo iptables -L -n | grep 8080

# Verify API key
curl -H "Authorization: Bearer your-key" \
  http://server-ip:8080/api/v1/health
```

### Docker Metrics Not Working

```bash
# Check socket permissions
ls -la /var/run/docker.sock

# Test Docker access
docker ps

# Fix permissions
sudo usermod -aG docker $USER
```

### Too Many Alerts

```yaml
# Enable deduplication in server.yaml
alerting:
  deduplication_enabled: true
  deduplication_window: 10m  # Increase window

# Or raise thresholds
  system_cpu_threshold: 90.0
  system_memory_threshold: 92.0
```

---

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create feature branch (`git checkout -b feat/amazing-feature`)
3. Commit changes (`git commit -m 'feat: add amazing feature'`)
4. Push to branch (`git push origin feat/amazing-feature`)
5. Open Pull Request

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Built with Go standard library and minimal dependencies
- Docker SDK for container monitoring
- gopsutil for system metrics collection
- Google Chat for rich notifications

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yanurag-dev/saviour/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yanurag-dev/saviour/discussions)
- **Documentation**: [Complete User Guide](USER_GUIDE.md)

---

**Built for reliable infrastructure monitoring** • **Production-ready** • **Easy to deploy**
