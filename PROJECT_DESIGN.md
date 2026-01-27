# Saviour - Internal Monitoring & Alerting Platform

## Project Overview
Saviour is a lightweight, internal monitoring tool designed to track health and performance of multiple EC2 instances running Docker containers. It provides centralized monitoring, intelligent alerting via Google Chat, and a unified web dashboard for infrastructure visibility.

## Design Philosophy
- **Push-based architecture**: Agents push data, never pulled
- **Simple deployment**: Single binary for agents, on-prem server
- **Minimal network requirements**: Only outbound HTTPS (port 443) from EC2
- **State-change alerting**: Alert only on state changes, prevent spam
- **Status-focused UI**: Simple, clear health indicators over complex graphs

---

## System Architecture

```
┌─────────────────────────────────────────────────────────┐
│         Saviour Core (On-Prem VM / Cloud)               │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Web Dashboard  │  REST API  │  Alert Manager   │  │
│  │  (HTMX/React)   │  (Go/Chi)  │  (State Engine)  │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │         In-Memory State Store (or SQLite)        │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────┘
                      │                    │
                      │ HTTPS POST         │ Webhook POST
                      │ (port 443)         ↓
                      ↑              ┌─────────────────┐
         ┌────────────┼──────────┐   │  Google Chat    │
         │            │          │   └─────────────────┘
    ┌────┴───┐   ┌───┴────┐  ┌──┴──────┐
    │ Agent  │   │ Agent  │  │ Agent   │
    │ EC2-1  │   │ EC2-2  │  │ EC2-N   │
    │        │   │        │  │         │
    │ ┌────┐ │   │ ┌────┐ │  │ ┌────┐ │
    │ │ 🐳 │ │   │ │ 🐳 │ │  │ │ 🐳 │ │
    │ │ 🐳 │ │   │ │ 🐳 │ │  │ │ 🐳 │ │
    │ │ 🐳 │ │   │ │ 🐳 │ │  │ │ 🐳 │ │
    └────────┘   └────────┘  └─────────┘
```

### Communication Flow
1. **Agent → Server**: HTTPS POST every 30s with metrics + heartbeat
2. **Server → Google Chat**: Webhook POST on state change alerts
3. **User → Dashboard**: HTTPS to view status via web browser

### Network Requirements
- **EC2 agents**: Outbound HTTPS (443) only - no inbound ports needed
- **Central server**: Public HTTPS endpoint (can be on-prem with public IP/domain)
- **No VPN required**: Agents and server don't need to be in same network

---

## Core Features (MVP)

### 1. Agent (Deployed on EC2 Instances)

#### 1.1 System Metrics Collection ✅ (Implemented)
- **CPU**: Usage per core, load average, utilization percentages
- **Memory**: Total, used, available, swap usage
- **Disk**: Usage per mount point, I/O statistics
- **Network**: Bandwidth usage, packet statistics, connection counts
- **System Info**: Hostname, OS, platform, uptime

#### 1.2 Docker Container Monitoring 🔴 (In Progress)
**Per-container metrics**:
- Container ID, name, image
- State (running, stopped, exited, paused, restarting)
- Health status (healthy, unhealthy, starting) from Docker health checks
- CPU usage percentage (per container)
- Memory usage (current, limit, percentage)
- Network I/O (bytes sent/received per container)
- Disk I/O (read/write bytes)
- Restart count
- OOM kill detection
- Container uptime
- Exit code (when stopped)

**Container discovery**:
- Monitor all containers by default
- Optional filtering by:
  - Container name pattern
  - Image name pattern
  - Docker labels
  - Container state

**Configuration**:
```yaml
metrics:
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    monitor_all: true
    
    # Optional filters
    filters:
      labels:
        - "monitor=true"
        - "env=production"
      names:
        - "api-*"
        - "web-*"
      images:
        - "mycompany/*"
    
    # Configurable thresholds
    alerts:
      default:
        cpu_threshold: 80.0
        memory_threshold: 90.0
        restart_threshold: 5      # Alert if >5 restarts
        restart_window: 300s      # in 5 minutes
      
      # Per-container overrides
      overrides:
        - name: "postgres"
          memory_threshold: 95.0
        - name: "redis"
          cpu_threshold: 70.0
```

#### 1.3 EC2 Metadata Integration 🟡
**Auto-discovery**:
- EC2 instance ID (use as agent name if not specified)
- Instance type (t3.large, m5.xlarge, etc.)
- Region and Availability Zone
- AMI ID
- Instance tags (for grouping/filtering)
- Private/Public IP addresses

**Configuration**:
```yaml
ec2:
  enabled: true
  auto_name: true           # Use instance-id as agent name
  include_tags: true        # Include instance tags in metrics
  metadata_endpoint: "http://169.254.169.254/latest/meta-data/"
```

#### 1.4 Push Mechanism 🔴 (To Implement)
**Metrics push**:
- HTTPS POST to central server
- Configurable push interval (default: 30s)
- Token-based authentication
- Retry logic with exponential backoff
- Batch metrics to reduce network calls
- Compression (gzip) for large payloads

**Heartbeat**:
- Separate heartbeat endpoint
- Sent every 30s (independent of metrics)
- Minimal payload (agent name + timestamp)
- Server detects missing heartbeat → triggers alert

**Configuration**:
```yaml
agent:
  name: "auto"                    # Or explicit name
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"  # From environment variable
  
  collect_interval: 30s           # How often to collect metrics
  push_interval: 30s              # How often to push to server
  heartbeat_interval: 30s         # Heartbeat frequency
  
  push_timeout: 10s               # HTTP request timeout
  retry_attempts: 3               # Retry on failure
  retry_backoff: 2s               # Initial backoff duration
```

**Payload structure**:
```json
{
  "agent_name": "i-1234567890abcdef0",
  "timestamp": "2026-01-27T21:30:00Z",
  "ec2_metadata": {
    "instance_id": "i-1234567890abcdef0",
    "instance_type": "t3.large",
    "region": "us-east-1",
    "availability_zone": "us-east-1a",
    "tags": {
      "env": "production",
      "team": "backend"
    }
  },
  "system": {
    "cpu": {...},
    "memory": {...},
    "disk": [...],
    "network": {...},
    "uptime": 123456
  },
  "containers": [
    {
      "id": "a1b2c3d4e5f6",
      "name": "nginx",
      "image": "nginx:1.21",
      "state": "running",
      "health": "healthy",
      "cpu_percent": 12.5,
      "memory_usage": 134217728,
      "memory_limit": 536870912,
      "restart_count": 0,
      "uptime": 86400
    }
  ]
}
```

#### 1.5 Deployment Modes ✅
**Binary on EC2 host**:
```bash
# Direct Docker socket access
./saviour-agent -config /etc/saviour/agent.yaml
```

**As Docker container**:
```bash
# Mount Docker socket from host
docker run -d \
  --name saviour-agent \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /etc/saviour/agent.yaml:/etc/saviour/agent.yaml:ro \
  -e SAVIOUR_API_KEY=sk_xxxxx \
  saviour-agent:latest
```

---

### 2. Central Server (Saviour Core)

#### 2.1 Metrics Ingestion API 🔴
**Endpoints**:
```
POST   /api/v1/metrics/push      # Agents push metrics here
POST   /api/v1/heartbeat          # Heartbeat endpoint
GET    /api/v1/health             # Server health check
```

**Authentication**:
- Token-based (Bearer token in Authorization header)
- Multiple API keys with different scopes
- API key management via config or database

**Rate limiting**:
- Per-agent rate limits to prevent abuse
- Configurable thresholds

#### 2.2 State Management 🔴
**In-memory state store** (or SQLite for persistence):
- Latest metrics from each agent
- Last heartbeat timestamp
- Current alert states
- Container inventory
- Historical state for trend detection

**State schema**:
```go
type ServerState struct {
    AgentName        string
    EC2InstanceID    string
    LastSeen         time.Time
    Status           string  // online, offline, degraded
    
    // Latest metrics
    SystemMetrics    SystemMetrics
    Containers       []ContainerState
    
    // Alert states
    ActiveAlerts     []Alert
}

type ContainerState struct {
    ID               string
    Name             string
    Image            string
    State            string
    PreviousState    string  // For state change detection
    LastStateChange  time.Time
    RestartCount     int
    AlertState       string  // ok, warning, critical
}
```

#### 2.3 Alert Detection Engine 🔴
**Threshold-based alerts**:
- System metrics (CPU, memory, disk)
- Container metrics (per-container thresholds)
- Container state changes
- Restart count threshold

**State-change detection**:
- Container: running → exited (CRITICAL)
- Container: healthy → unhealthy (WARNING)
- Server: online → offline (missing heartbeat) (CRITICAL)
- Disk: <90% → >90% (WARNING)

**Alert rules**:
```yaml
alerting:
  enabled: true
  check_interval: 30s
  
  # Prevent alert spam
  deduplication:
    enabled: true
    window: 5m  # Don't send same alert twice within 5 minutes
  
  # Alert rules
  rules:
    # System alerts
    - name: "disk_critical"
      condition: "system.disk.used_percent > 90"
      duration: 2m          # Must be sustained for 2 minutes
      severity: "critical"
      message: "🚨 Disk Alert\nEC2: {{.agent_name}}\nUsage: {{.disk_percent}}%\nMount: {{.mount_point}}"
    
    - name: "memory_high"
      condition: "system.memory.used_percent > 85"
      duration: 5m
      severity: "warning"
      message: "⚠️ High Memory\nEC2: {{.agent_name}}\nUsage: {{.memory_percent}}%"
    
    # Container alerts
    - name: "container_stopped"
      condition: "container.state_changed AND container.state == 'exited'"
      severity: "critical"
      message: "💀 Container Stopped\nEC2: {{.agent_name}}\nContainer: {{.container_name}}\nExit Code: {{.exit_code}}"
    
    - name: "container_unhealthy"
      condition: "container.health == 'unhealthy'"
      duration: 3m
      severity: "warning"
      message: "🏥 Container Unhealthy\nEC2: {{.agent_name}}\nContainer: {{.container_name}}"
    
    - name: "container_restarting"
      condition: "container.restart_count > 5 in 5m"
      severity: "warning"
      message: "🔄 Container Restarting\nEC2: {{.agent_name}}\nContainer: {{.container_name}}\nRestarts: {{.restart_count}} in 5 minutes"
    
    - name: "container_oom"
      condition: "container.oom_killed == true"
      severity: "critical"
      message: "💥 OOM Kill\nEC2: {{.agent_name}}\nContainer: {{.container_name}}"
    
    # Heartbeat alerts
    - name: "agent_offline"
      condition: "heartbeat_missing > 2m"
      severity: "critical"
      message: "🔴 Agent Offline\nEC2: {{.agent_name}}\nLast Seen: {{.last_seen}}"
```

#### 2.4 Google Chat Integration 🔴
**Webhook sender**:
- Rich card-based messages
- Color-coded by severity (red=critical, yellow=warning)
- Clickable links to dashboard
- Grouped notifications (multiple alerts in one message)

**Message format**:
```json
{
  "cards": [{
    "header": {
      "title": "🚨 CRITICAL ALERT",
      "subtitle": "ec2-prod-web-01",
      "imageStyle": "IMAGE"
    },
    "sections": [{
      "widgets": [
        {
          "keyValue": {
            "topLabel": "Alert Type",
            "content": "Disk Space Critical"
          }
        },
        {
          "keyValue": {
            "topLabel": "Disk Usage",
            "content": "92%",
            "contentMultiline": false
          }
        },
        {
          "keyValue": {
            "topLabel": "Mount Point",
            "content": "/var/lib/docker"
          }
        },
        {
          "keyValue": {
            "topLabel": "Time",
            "content": "2026-01-27 21:30:45 UTC"
          }
        }
      ]
    }],
    "buttons": [{
      "textButton": {
        "text": "VIEW DASHBOARD",
        "onClick": {
          "openLink": {
            "url": "https://saviour.company.com/servers/ec2-prod-web-01"
          }
        }
      }
    }]
  }]
}
```

**Configuration**:
```yaml
alerting:
  google_chat:
    enabled: true
    webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
    timeout: 10s
    retry_attempts: 3
    
    # Severity filtering
    min_severity: "warning"  # Send warning and critical only
    
    # Message formatting
    include_dashboard_link: true
    dashboard_url: "https://saviour.company.com"
```

#### 2.5 REST API (for Dashboard) 🔴
**Endpoints**:
```
GET    /api/v1/servers                    # List all servers
GET    /api/v1/servers/:id                # Server details
GET    /api/v1/servers/:id/containers     # Containers on server
GET    /api/v1/containers                 # All containers
GET    /api/v1/containers/:id             # Container details
GET    /api/v1/alerts                     # Active alerts
GET    /api/v1/alerts/history             # Alert history
GET    /api/v1/metrics/current            # Current metrics (all servers)
GET    /api/v1/metrics/history            # Historical metrics (time range)
```

**Response examples**:
```json
// GET /api/v1/servers
{
  "servers": [
    {
      "agent_name": "ec2-prod-web-01",
      "ec2_instance_id": "i-1234567890abcdef0",
      "status": "online",
      "last_seen": "2026-01-27T21:30:45Z",
      "uptime": 123456,
      "cpu_percent": 45.2,
      "memory_percent": 72.8,
      "disk_percent": 65.4,
      "container_count": 8,
      "active_alerts": 1
    }
  ],
  "summary": {
    "total": 12,
    "online": 11,
    "offline": 1,
    "degraded": 0
  }
}

// GET /api/v1/alerts
{
  "alerts": [
    {
      "id": "alert-123",
      "agent_name": "ec2-prod-web-01",
      "severity": "critical",
      "alert_type": "disk_critical",
      "message": "Disk usage at 92%",
      "triggered_at": "2026-01-27T21:25:00Z",
      "state": "active"
    }
  ]
}
```

#### 2.6 Data Storage 🟡
**Initial**: In-memory state (ephemeral)
**Future**: SQLite or PostgreSQL

**Schema** (for future persistence):
```sql
-- Server registry
CREATE TABLE servers (
  id UUID PRIMARY KEY,
  agent_name VARCHAR(255) UNIQUE,
  ec2_instance_id VARCHAR(50),
  ec2_region VARCHAR(20),
  status VARCHAR(20),
  last_seen_at TIMESTAMP,
  created_at TIMESTAMP
);

-- Current state (latest metrics)
CREATE TABLE server_state (
  server_id UUID PRIMARY KEY REFERENCES servers(id),
  cpu_percent FLOAT,
  memory_percent FLOAT,
  disk_percent FLOAT,
  container_count INT,
  updated_at TIMESTAMP
);

-- Container inventory
CREATE TABLE containers (
  id VARCHAR(64) PRIMARY KEY,
  server_id UUID REFERENCES servers(id),
  name VARCHAR(255),
  image VARCHAR(255),
  state VARCHAR(20),
  health VARCHAR(20),
  restart_count INT,
  last_seen_at TIMESTAMP
);

-- Alerts
CREATE TABLE alerts (
  id UUID PRIMARY KEY,
  server_id UUID REFERENCES servers(id),
  container_id VARCHAR(64),
  severity VARCHAR(20),
  alert_type VARCHAR(50),
  message TEXT,
  triggered_at TIMESTAMP,
  resolved_at TIMESTAMP
);
```

---

### 3. Web Dashboard 🟡

#### 3.1 Technology
**Option 1**: HTMX + Server-side rendered HTML (Recommended for MVP)
- No build step
- Simple, fast
- Progressive enhancement

**Option 2**: React SPA (Future)
- More interactive
- Better UX
- Requires build process

#### 3.2 Pages

**Overview Dashboard** (`/`):
```
┌─────────────────────────────────────────────────────┐
│  Saviour Monitoring                                  │
├─────────────────────────────────────────────────────┤
│  Servers: 11 🟢 online  |  1 🔴 offline              │
│  Containers: 156 running  |  3 stopped  |  2 ⚠️       │
│  Active Alerts: 3                                    │
├─────────────────────────────────────────────────────┤
│                                                      │
│  🔴 ec2-prod-db-01      Last seen: 5m ago           │
│  🟢 ec2-prod-web-01     CPU: 45%  Mem: 72%          │
│  🟢 ec2-prod-web-02     CPU: 38%  Mem: 65%          │
│  🟡 ec2-prod-api-01     1 alert active              │
│                                                      │
│  Recent Alerts:                                      │
│  🚨 ec2-prod-db-01 offline (5m ago)                 │
│  ⚠️  ec2-prod-api-01 high memory (12m ago)          │
│                                                      │
└─────────────────────────────────────────────────────┘
```

**Server Detail Page** (`/servers/:id`):
```
ec2-prod-web-01 (i-1234567890abcdef0)
Region: us-east-1  |  Type: t3.large  |  Uptime: 23d 14h

System Health
  CPU: 45.2%   Memory: 72.8%   Disk: 65.4%

Containers (8 running)
┌─────────────┬────────┬─────────┬────────┬────────┐
│ Name        │ State  │ CPU     │ Memory │ Alerts │
├─────────────┼────────┼─────────┼────────┼────────┤
│ nginx       │ 🟢 Run │  12%    │ 128 MB │   -    │
│ api-v1      │ 🟢 Run │  45%    │ 512 MB │   -    │
│ postgres    │ 🟢 Run │  28%    │ 2.1 GB │   -    │
│ redis       │ 🟡 Unh │   8%    │  64 MB │   1    │
└─────────────┴────────┴─────────┴────────┴────────┘
```

**Container List** (`/containers`):
```
All Containers Across Servers

Filter: [All] [Running] [Stopped] [Unhealthy]
Search: [______________________]

┌─────────────┬──────────────┬────────┬─────────┬────────┐
│ Container   │ Server       │ State  │ CPU     │ Memory │
├─────────────┼──────────────┼────────┼─────────┼────────┤
│ postgres    │ ec2-prod-db  │ 🟢 Run │  65%    │ 3.2 GB │
│ api-v1      │ ec2-prod-api │ 🟢 Run │  45%    │ 512 MB │
│ nginx       │ ec2-prod-web │ 🟢 Run │  12%    │ 128 MB │
│ worker-1    │ ec2-prod-api │ 🔴 Exit│   -     │   -    │
└─────────────┴──────────────┴────────┴─────────┴────────┘
```

**Alerts Page** (`/alerts`):
```
Active Alerts (3)

🚨 CRITICAL: ec2-prod-db-01 offline
   Last seen: 5 minutes ago
   [ACKNOWLEDGE] [VIEW SERVER]

⚠️ WARNING: High memory on ec2-prod-api-01
   Memory: 92% (threshold: 85%)
   Duration: 12 minutes
   [ACKNOWLEDGE] [VIEW SERVER]

Alert History (Last 24h)
┌──────────────────────────────────────────────────┐
│ 21:25  🚨  ec2-prod-db-01 offline     [Resolved] │
│ 20:15  ⚠️   Disk 92% on ec2-prod-web   [Resolved] │
│ 19:30  💀  nginx container stopped     [Resolved] │
└──────────────────────────────────────────────────┘
```

#### 3.3 Auto-refresh
- Dashboard auto-refreshes every 30s
- Real-time updates via WebSocket (future)
- Manual refresh button

---

## Implementation Status

### ✅ Completed
- [x] Project structure
- [x] Agent scaffolding
- [x] System metrics collection (CPU, memory, disk, network)
- [x] YAML configuration
- [x] Local alert threshold detection
- [x] Docker support (Dockerfile, docker-compose)
- [x] Makefile build automation

### 🔴 In Progress / Planned

#### Phase 1: Docker Container Monitoring (Week 1)
- [ ] Docker client integration
- [ ] Container discovery and filtering
- [ ] Container metrics collection
- [ ] Container state tracking
- [ ] Add containers to metrics payload

#### Phase 2: Agent Push Mechanism (Week 1-2)
- [ ] HTTP client for server push
- [ ] Heartbeat mechanism
- [ ] Token authentication
- [ ] Retry logic with backoff
- [ ] Payload compression
- [ ] EC2 metadata integration

#### Phase 3: Central Server Core (Week 2-3)
- [ ] HTTP server setup (Chi router)
- [ ] `/api/v1/metrics/push` endpoint
- [ ] `/api/v1/heartbeat` endpoint
- [ ] Token authentication middleware
- [ ] In-memory state store
- [ ] Alert detection engine
- [ ] State-change detection
- [ ] Google Chat webhook integration

#### Phase 4: Web Dashboard (Week 3-4)
- [ ] Overview page
- [ ] Server list page
- [ ] Server detail page
- [ ] Container list page
- [ ] Alerts page
- [ ] Auto-refresh mechanism

### 🟡 Future Enhancements
- [ ] Historical metrics (time-series storage)
- [ ] Metric graphs and charts
- [ ] Log aggregation from containers
- [ ] Service health checks (HTTP/TCP/ping)
- [ ] Container auto-restart capability
- [ ] Multi-user authentication (login)
- [ ] Alert acknowledgment
- [ ] Alert routing rules
- [ ] Notification channels (email, Slack, PagerDuty)
- [ ] Mobile app
- [ ] Kubernetes support
- [ ] Cloud integration (AWS CloudWatch, etc.)

---

## Configuration Reference

### Agent Configuration
```yaml
# /etc/saviour/agent.yaml

# Agent identity
agent:
  name: "auto"                         # "auto" = use EC2 instance-id
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"       # From environment variable
  
  # Timing
  collect_interval: 30s                # How often to collect metrics
  push_interval: 30s                   # How often to push to server
  heartbeat_interval: 30s              # Heartbeat frequency
  
  # Network
  push_timeout: 10s                    # HTTP timeout
  retry_attempts: 3                    # Retry on failure
  retry_backoff: 2s                    # Initial backoff duration
  compression: true                    # Gzip compression

# EC2 metadata
ec2:
  enabled: true
  auto_name: true                      # Use instance-id as agent name
  include_tags: true                   # Include instance tags

# System metrics
metrics:
  system: true
  disk_mounts:
    - "/"
    - "/var/lib/docker"
  
  # Docker container monitoring
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    monitor_all: true
    
    # Optional filters
    filters:
      labels:
        - "monitor=true"
      names:
        - "api-*"
        - "web-*"
    
    # Container alert thresholds
    alerts:
      default:
        cpu_threshold: 80.0
        memory_threshold: 90.0
        restart_threshold: 5
        restart_window: 300s
      
      overrides:
        - name: "postgres"
          memory_threshold: 95.0

# System alert thresholds
alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

### Server Configuration
```yaml
# /etc/saviour/server.yaml

# Server settings
server:
  listen: "0.0.0.0:8080"
  public_url: "https://saviour.company.com"
  
  # TLS (optional, use reverse proxy recommended)
  tls:
    enabled: false
    cert_file: "/etc/saviour/tls/cert.pem"
    key_file: "/etc/saviour/tls/key.pem"

# Authentication
auth:
  api_keys:
    - key: "sk_prod_xxxxxxxxxxxxx"
      name: "production-agents"
      scopes: ["metrics:write", "heartbeat:write"]
    
    - key: "sk_dash_xxxxxxxxxxxxx"
      name: "dashboard"
      scopes: ["metrics:read", "alerts:read"]

# Alerting
alerting:
  enabled: true
  check_interval: 30s
  
  # Deduplication
  deduplication:
    enabled: true
    window: 5m
  
  # Google Chat
  google_chat:
    enabled: true
    webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
    timeout: 10s
    retry_attempts: 3
    min_severity: "warning"
  
  # Alert rules
  rules:
    - name: "disk_critical"
      condition: "system.disk.used_percent > 90"
      duration: 2m
      severity: "critical"
      message: "🚨 Disk Alert\nEC2: {{.agent_name}}\nUsage: {{.disk_percent}}%"
    
    - name: "container_stopped"
      condition: "container.state_changed AND container.state == 'exited'"
      severity: "critical"
      message: "💀 Container Stopped\nEC2: {{.agent_name}}\nContainer: {{.container_name}}"
    
    - name: "agent_offline"
      condition: "heartbeat_missing > 2m"
      severity: "critical"
      message: "🔴 Agent Offline\nEC2: {{.agent_name}}\nLast Seen: {{.last_seen}}"

# Storage (optional, in-memory by default)
storage:
  type: "memory"  # or "sqlite", "postgres"
  
  # SQLite config
  sqlite:
    path: "/var/lib/saviour/saviour.db"
  
  # PostgreSQL config
  postgres:
    host: "localhost"
    port: 5432
    database: "saviour"
    user: "saviour"
    password: "${DB_PASSWORD}"
```

---

## Deployment Guide

### Agent Deployment (EC2)

**1. Binary installation**:
```bash
# Download binary
wget https://github.com/your-org/saviour/releases/latest/download/saviour-agent-linux-amd64
chmod +x saviour-agent-linux-amd64
sudo mv saviour-agent-linux-amd64 /usr/local/bin/saviour-agent

# Create config
sudo mkdir -p /etc/saviour
sudo vi /etc/saviour/agent.yaml

# Create systemd service
sudo cat > /etc/systemd/system/saviour-agent.service <<EOF
[Unit]
Description=Saviour Monitoring Agent
After=network.target docker.service

[Service]
Type=simple
User=root
Environment="SAVIOUR_API_KEY=sk_prod_xxxxx"
ExecStart=/usr/local/bin/saviour-agent -config /etc/saviour/agent.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable saviour-agent
sudo systemctl start saviour-agent
sudo systemctl status saviour-agent
```

**2. Docker container deployment**:
```bash
docker run -d \
  --name saviour-agent \
  --restart unless-stopped \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /etc/saviour/agent.yaml:/etc/saviour/agent.yaml:ro \
  -e SAVIOUR_API_KEY=sk_prod_xxxxx \
  saviour-agent:latest
```

### Server Deployment (On-Prem)

**Docker Compose** (Recommended):
```yaml
version: '3.8'

services:
  saviour-server:
    image: saviour-server:latest
    container_name: saviour-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./server.yaml:/etc/saviour/server.yaml:ro
      - saviour-data:/var/lib/saviour
    environment:
      - GOOGLE_CHAT_WEBHOOK_URL=${GOOGLE_CHAT_WEBHOOK_URL}
      - DB_PASSWORD=${DB_PASSWORD}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  saviour-data:
```

**Reverse Proxy (Nginx)**:
```nginx
server {
    listen 443 ssl http2;
    server_name saviour.company.com;
    
    ssl_certificate /etc/letsencrypt/live/saviour.company.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/saviour.company.com/privkey.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## Technology Stack

### Agent
- **Language**: Go 1.21+
- **System metrics**: gopsutil v3
- **Docker client**: docker/docker/client
- **Configuration**: gopkg.in/yaml.v3
- **HTTP client**: net/http (stdlib)

### Server
- **Language**: Go 1.21+
- **HTTP router**: Chi v5
- **Database**: In-memory (MVP), SQLite/PostgreSQL (future)
- **Template engine**: html/template (MVP), HTMX
- **Authentication**: Token-based (custom middleware)

### Dashboard
- **Frontend**: HTMX + TailwindCSS (MVP)
- **Future**: React + TypeScript

---

## Success Metrics
- **Deployment time**: < 10 minutes per EC2 agent
- **Agent overhead**: < 50MB RAM, < 1% CPU
- **Alert latency**: < 60 seconds from state change to Google Chat
- **Server capacity**: Handle 100+ agents with < 500MB RAM
- **Uptime**: 99.9% server availability

---

## Security Considerations
- Token-based authentication for all API calls
- HTTPS only for agent-server communication
- Read-only Docker socket access for agent
- No inbound ports on EC2 (only outbound HTTPS)
- API key rotation support
- Rate limiting on server endpoints

---

**Status**: Design Document  
**Version**: 2.0  
**Last Updated**: 2026-01-27  
**Implementation Status**: 40% Complete (Agent foundation done, server pending)
