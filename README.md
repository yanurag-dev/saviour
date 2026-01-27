# Saviour - Internal Monitoring & Alerting Platform

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/yanurag-dev/saviour)

**Saviour** is a lightweight, push-based monitoring and alerting platform designed for tracking Docker containers and system health across multiple EC2 instances. It provides centralized monitoring, intelligent alerting via Google Chat, and real-time health visibility.

---

## 🎯 Key Features

### ✅ **Phase 1: Docker Container Monitoring** (Complete)
- 🐳 **Container Metrics**: CPU, memory, network I/O, disk I/O, restart counts
- 🏥 **Health Tracking**: Container state changes, health checks, OOM detection
- 🎯 **Flexible Filtering**: Monitor by labels, names, images, or all containers
- 📊 **Per-Container Alerts**: Customizable thresholds with pattern matching

### ✅ **Phase 2: Agent Push Mechanism** (Complete)
- 🔐 **Secure Communication**: HTTPS with Bearer token authentication
- 🗜️ **Compression**: Automatic gzip compression (10MB+ → ~14KB)
- 🔄 **Retry Logic**: Exponential backoff with configurable attempts
- 💓 **Heartbeat**: Separate heartbeat mechanism for offline detection
- ⚡ **Independent Intervals**: Collect (15s), push (20s), heartbeat (10s)

### ✅ **Phase 3: Central Server & Alerting** (Complete)
- 🏢 **In-Memory State Store**: Fast, thread-safe state management
- 🔔 **Alert Detection Engine**: System and container threshold monitoring
- 🚫 **Deduplication**: Time-based alert deduplication (prevents spam)
- 💬 **Google Chat Integration**: Rich card notifications with severity icons
- 🔒 **Security Hardened**: Request limits, CORS whitelist, config validation
- ☁️ **EC2 Auto-Discovery**: Automatic instance metadata fetching (IMDSv2)

### 🔜 **Phase 4: Web Dashboard** (Planned)
- 📈 Multi-agent overview
- 🔍 Container search and filtering
- 📜 Alert history and acknowledgment
- 📊 Real-time metrics visualization

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│         Saviour Core (Central Server)                   │
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

**Push-Based Design Benefits**:
- ✅ No firewall configuration needed (agents only need outbound HTTPS)
- ✅ Real-time metrics delivery
- ✅ Simple agent deployment
- ✅ Automatic offline detection via heartbeat

---

## 🚀 Quick Start

### Prerequisites
- Go 1.24+
- Docker (optional, for containerized deployment)
- Docker socket access (for container monitoring)

### 1. Build from Source

```bash
# Clone repository
git clone https://github.com/yanurag-dev/saviour.git
cd saviour

# Build agent and server
make build

# Or build individually
go build -o bin/saviour-agent ./cmd/agent
go build -o bin/saviour-server ./cmd/server
```

### 2. Configure & Run Server

```bash
# Copy example configuration
cp examples/server.yaml server.yaml

# Edit configuration (add API keys, configure alerts)
vim server.yaml

# Start server
./bin/saviour-server -config server.yaml
```

Server will start on `http://0.0.0.0:8080` with endpoints:
- `POST /api/v1/metrics/push` - Receive metrics from agents
- `POST /api/v1/heartbeat` - Heartbeat tracking
- `GET /api/v1/health` - Health check

### 3. Configure & Run Agent

```bash
# Copy example configuration
cp examples/agent-docker.yaml agent.yaml

# Edit configuration (add server URL and API key)
vim agent.yaml

# Start agent
./bin/saviour-agent -config agent.yaml
```

Agent will:
- Collect system and Docker metrics every 15s
- Push metrics to server every 20s
- Send heartbeat every 10s

---

## 📋 Configuration

### Server Configuration (`server.yaml`)

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
  deduplication_enabled: true
  deduplication_window: 5m
  system_cpu_threshold: 80.0
  system_memory_threshold: 85.0
  system_disk_threshold: 90.0

google_chat:
  enabled: true
  webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
  dashboard_url: "https://saviour.company.com"

cors:
  enabled: false  # Enable for dashboard
  dev_mode: false
  allowed_origins:
    - "https://dashboard.company.com"
```

### Agent Configuration (`agent.yaml`)

```yaml
agent:
  name: "ec2-prod-web-01"  # Or "auto" for EC2 instance ID
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"
  
  collect_interval: 15s
  push_interval: 20s
  heartbeat_interval: 10s
  
  push_timeout: 10s
  retry_attempts: 3
  retry_backoff: 2s

metrics:
  system: true
  
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    monitor_all: true
    
    filters:
      labels: ["monitor=true"]
      names: ["api-*", "web-*"]
      images: ["mycompany/*"]
    
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

alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

See [examples/](examples/) for more configuration examples.

---

## 🔒 Security Features

### Authentication
- **Bearer Token**: All agent requests require valid API key
- **Scope-Based**: Different scopes for metrics, heartbeat, and dashboard access
- **Multiple Keys**: Support for different key sets (agents, dashboards, admin)

### Request Protection
- **Size Limits**: 10MB maximum request size
- **Gzip Bomb Protection**: http.MaxBytesReader prevents DoS
- **CORS Whitelist**: Configurable allowed origins (no wildcards in production)

### Configuration Validation
- **Fail-Fast**: Invalid config prevents server startup
- **Duration Validation**: All durations must be > 0
- **Threshold Validation**: Percentages must be 0-100
- **Required Fields**: API keys, scopes validated on startup

### Thread Safety
- **Deep Copies**: All state getters return snapshots
- **Data Race Free**: Proper mutex locking throughout
- **Concurrent Safe**: Multiple goroutines can safely access state

---

## 📊 Monitoring Capabilities

### System Metrics
- **CPU**: Usage percentage, per-core usage, load averages
- **Memory**: Total, used, available, swap usage and percentages
- **Disk**: Per-mount usage, I/O statistics, inode usage
- **Network**: Bytes sent/received, packet counts, errors, drops

### Container Metrics (Per Container)
- **Identity**: ID, name, image, labels
- **State**: Running, exited, paused, restarting, with status
- **Health**: Docker health check status (healthy, unhealthy, starting)
- **Resources**: CPU %, memory usage/limit/%, network I/O, disk I/O
- **Lifecycle**: Created time, started time, exit code, OOM killed flag
- **Reliability**: Restart count, uptime, process count (PIDs)

### Alert Types

#### System Alerts
- 🔴 **High CPU**: CPU usage > threshold
- 🔴 **High Memory**: Memory usage > threshold
- 🔴 **High Disk**: Disk usage > threshold
- 🔴 **Agent Offline**: No heartbeat for > timeout

#### Container Alerts
- 💀 **Container Stopped**: Running → Exited/Dead
- 🏥 **Container Unhealthy**: Health check failing
- ⚠️ **High CPU**: Container CPU > threshold
- 🚨 **High Memory**: Container memory > threshold
- 💥 **OOM Killed**: Container killed by OOM
- 🔄 **Restart Loop**: Too many restarts in time window

---

## 📦 Deployment Options

### Binary Deployment

```bash
# On each EC2 instance
curl -L https://github.com/yanurag-dev/saviour/releases/latest/download/saviour-agent -o /usr/local/bin/saviour-agent
chmod +x /usr/local/bin/saviour-agent

# Create systemd service
sudo tee /etc/systemd/system/saviour-agent.service > /dev/null <<EOF
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
EOF

sudo systemctl enable saviour-agent
sudo systemctl start saviour-agent
```

### Docker Deployment

```bash
# Agent as Docker container
docker run -d \
  --name saviour-agent \
  --restart unless-stopped \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /etc/saviour/agent.yaml:/etc/saviour/agent.yaml:ro \
  -e SAVIOUR_API_KEY=your-api-key-here \
  saviour-agent:latest
```

### Docker Compose

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
    depends_on:
      - saviour-server
```

---

## 🧪 Development

### Prerequisites
- Go 1.24+
- Docker (for testing container monitoring)
- Make

### Build Commands

```bash
# Build both agent and server
make build

# Build agent only
make build-agent

# Build server only
make build-server

# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
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

---

## 📚 Documentation

- **[USER_GUIDE.md](USER_GUIDE.md)** - Comprehensive user guide
- **[PROJECT_DESIGN.md](PROJECT_DESIGN.md)** - Detailed design document
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Production deployment guide
- **[PROGRESS.md](PROGRESS.md)** - Development progress tracking
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

---

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Built with Go standard library and minimal dependencies
- Docker SDK for container monitoring
- gopsutil for system metrics
- Google Chat for alerting

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yanurag-dev/saviour/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yanurag-dev/saviour/discussions)
- **Email**: support@saviour.dev

---

**Made with ❤️ for reliable infrastructure monitoring**
