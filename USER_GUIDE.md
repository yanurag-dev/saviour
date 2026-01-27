# Saviour User Guide

Complete guide for installing, configuring, and using Saviour monitoring platform.

---

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Server Setup](#server-setup)
4. [Agent Setup](#agent-setup)
5. [Configuration Guide](#configuration-guide)
6. [Alert Configuration](#alert-configuration)
7. [Google Chat Integration](#google-chat-integration)
8. [Monitoring Containers](#monitoring-containers)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)
11. [FAQ](#faq)

---

## Introduction

### What is Saviour?

Saviour is a push-based monitoring platform designed for EC2 instances running Docker containers. It provides:

- **Real-time monitoring** of system and container metrics
- **Intelligent alerting** with deduplication
- **Google Chat notifications** for critical events
- **Centralized dashboard** for multiple servers
- **Minimal configuration** with sensible defaults

### Architecture Overview

```
EC2 Instances (Agents) → Central Server → Google Chat Alerts
         ↓                       ↓
  Collect Metrics        Store & Analyze
  Push via HTTPS         Detect Thresholds
  Send Heartbeat         Send Notifications
```

**Key Concepts**:
- **Agent**: Runs on each EC2 instance, collects and pushes metrics
- **Server**: Central service that receives metrics, detects alerts, sends notifications
- **Heartbeat**: Lightweight signal to detect offline agents
- **Alert**: Triggered when metrics exceed thresholds or state changes occur

---

## Installation

### Method 1: Binary Installation (Recommended)

#### Server Installation

```bash
# Download latest release
curl -L https://github.com/yanurag-dev/saviour/releases/latest/download/saviour-server \
  -o /usr/local/bin/saviour-server

# Make executable
chmod +x /usr/local/bin/saviour-server

# Verify installation
saviour-server --version
```

#### Agent Installation

```bash
# On each EC2 instance
curl -L https://github.com/yanurag-dev/saviour/releases/latest/download/saviour-agent \
  -o /usr/local/bin/saviour-agent

# Make executable
chmod +x /usr/local/bin/saviour-agent

# Verify installation
saviour-agent --version
```

### Method 2: Build from Source

```bash
# Install Go 1.24+
# Clone repository
git clone https://github.com/yanurag-dev/saviour.git
cd saviour

# Build
make build

# Binaries will be in bin/
ls -lh bin/
```

### Method 3: Docker Images

```bash
# Pull images
docker pull ghcr.io/yanurag-dev/saviour-server:latest
docker pull ghcr.io/yanurag-dev/saviour-agent:latest
```

---

## Server Setup

### Step 1: Create Configuration File

```bash
# Create config directory
sudo mkdir -p /etc/saviour

# Create server configuration
sudo vim /etc/saviour/server.yaml
```

**Minimal Configuration** (`/etc/saviour/server.yaml`):

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  api_keys:
    # Generate secure random keys
    - key: "sk_prod_a1b2c3d4e5f6g7h8i9j0"
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
  enabled: false  # Enable after setting up webhook
  webhook_url: ""
```

### Step 2: Generate API Keys

```bash
# Generate secure random API keys
openssl rand -hex 32
# Output: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6

# Use format: sk_prod_<random-string>
# Example: sk_prod_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**Security Best Practices**:
- Use different keys for different environments (dev, staging, prod)
- Rotate keys regularly (every 90 days)
- Store keys in environment variables, not in config files
- Use key management service (AWS Secrets Manager, HashiCorp Vault)

### Step 3: Start Server

#### Option A: Direct Execution

```bash
# Start server
saviour-server -config /etc/saviour/server.yaml

# Server will log:
# 2026/01/28 10:00:00 Starting Saviour Server on 0.0.0.0:8080
# 2026/01/28 10:00:00 Server listening on 0.0.0.0:8080
```

#### Option B: Systemd Service

```bash
# Create systemd service file
sudo tee /etc/systemd/system/saviour-server.service > /dev/null <<'EOF'
[Unit]
Description=Saviour Monitoring Server
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=saviour
Group=saviour
WorkingDirectory=/opt/saviour
ExecStart=/usr/local/bin/saviour-server -config /etc/saviour/server.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=saviour-server

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/saviour

[Install]
WantedBy=multi-user.target
EOF

# Create user
sudo useradd -r -s /bin/false saviour

# Create directories
sudo mkdir -p /opt/saviour /var/log/saviour
sudo chown saviour:saviour /opt/saviour /var/log/saviour

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable saviour-server
sudo systemctl start saviour-server

# Check status
sudo systemctl status saviour-server
```

#### Option C: Docker

```bash
docker run -d \
  --name saviour-server \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /etc/saviour/server.yaml:/etc/saviour/server.yaml:ro \
  -e GOOGLE_CHAT_WEBHOOK_URL=${GOOGLE_CHAT_WEBHOOK_URL} \
  ghcr.io/yanurag-dev/saviour-server:latest
```

### Step 4: Verify Server is Running

```bash
# Check health endpoint
curl http://localhost:8080/api/v1/health

# Expected output:
# {
#   "status": "ok",
#   "agents_online": 0,
#   "agents_offline": 0,
#   "active_alerts": 0
# }
```

### Step 5: Configure Firewall (if applicable)

```bash
# AWS Security Group: Allow inbound on port 8080 from agent IPs
# Or use SSH tunnel for secure access

# If using reverse proxy (recommended):
# - Nginx/Caddy in front of server
# - Serve on port 443 with TLS
# - Configure agents to use https://saviour.company.com
```

---

## Agent Setup

### Step 1: Create Agent Configuration

```bash
# Create config directory
sudo mkdir -p /etc/saviour

# Create agent configuration
sudo vim /etc/saviour/agent.yaml
```

**Basic Configuration** (`/etc/saviour/agent.yaml`):

```yaml
agent:
  name: "auto"  # Use EC2 instance ID automatically
  server_url: "http://saviour-server:8080"  # Or https://saviour.company.com
  api_key: "${SAVIOUR_API_KEY}"  # From environment variable
  
  collect_interval: 15s
  push_interval: 20s
  heartbeat_interval: 10s

metrics:
  system: true
  
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    monitor_all: true  # Monitor all containers by default

alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

### Step 2: Set API Key

```bash
# Set environment variable
export SAVIOUR_API_KEY="sk_prod_your-api-key-here"

# Or add to systemd service file (see below)
# Or add to ~/.bashrc for persistent setting
echo 'export SAVIOUR_API_KEY="sk_prod_your-api-key-here"' >> ~/.bashrc
```

### Step 3: Start Agent

#### Option A: Direct Execution

```bash
# Start agent
saviour-agent -config /etc/saviour/agent.yaml

# Agent will log metrics collection:
# [saviour-agent] 2026/01/28 10:05:00 Starting Saviour Agent...
# [saviour-agent] 2026/01/28 10:05:00 Agent 'i-1234567890abcdef0' starting...
```

#### Option B: Systemd Service (Recommended for Production)

```bash
# Create systemd service
sudo tee /etc/systemd/system/saviour-agent.service > /dev/null <<'EOF'
[Unit]
Description=Saviour Monitoring Agent
After=network.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/saviour
Environment="SAVIOUR_API_KEY=sk_prod_your-api-key-here"
ExecStart=/usr/local/bin/saviour-agent -config /etc/saviour/agent.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=saviour-agent

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable saviour-agent
sudo systemctl start saviour-agent

# Check status
sudo systemctl status saviour-agent

# View logs
sudo journalctl -u saviour-agent -f
```

#### Option C: Docker Container

```bash
docker run -d \
  --name saviour-agent \
  --restart unless-stopped \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /etc/saviour/agent.yaml:/etc/saviour/agent.yaml:ro \
  -e SAVIOUR_API_KEY=sk_prod_your-api-key-here \
  ghcr.io/yanurag-dev/saviour-agent:latest
```

### Step 4: Verify Agent is Connected

```bash
# On server, check health endpoint
curl http://localhost:8080/api/v1/health

# Should show:
# {
#   "status": "ok",
#   "agents_online": 1,  # ← Agent is connected
#   "agents_offline": 0,
#   "active_alerts": 0
# }
```

---

## Configuration Guide

### Server Configuration Reference

```yaml
# Server HTTP settings
server:
  host: "0.0.0.0"      # Listen address (0.0.0.0 = all interfaces)
  port: 8080           # Listen port

# Authentication settings
auth:
  api_keys:
    - key: "sk_prod_agents_key"
      name: "production-agents"
      scopes: ["metrics:write", "heartbeat:write"]
    
    - key: "sk_prod_dashboard_key"
      name: "dashboard"
      scopes: ["metrics:read", "alerts:read"]

# Alert detection settings
alerting:
  enabled: true
  check_interval: 30s              # How often to check metrics
  heartbeat_timeout: 2m            # Mark offline after this
  
  # Deduplication prevents alert spam
  deduplication_enabled: true
  deduplication_window: 5m         # Don't repeat same alert within 5min
  
  # System-level thresholds
  system_cpu_threshold: 80.0       # Alert if CPU > 80%
  system_memory_threshold: 85.0    # Alert if memory > 85%
  system_disk_threshold: 90.0      # Alert if disk > 90%

# Google Chat webhook integration
google_chat:
  enabled: true
  webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
  dashboard_url: "https://saviour.company.com"

# CORS settings (for web dashboard)
cors:
  enabled: false                   # Enable when deploying dashboard
  dev_mode: false                  # true = allow all origins
  allowed_origins:
    - "https://dashboard.company.com"
```

### Agent Configuration Reference

```yaml
# Agent identification and server connection
agent:
  name: "auto"                     # "auto" = use EC2 instance ID
  server_url: "https://saviour.company.com"
  api_key: "${SAVIOUR_API_KEY}"    # From environment variable
  
  # Collection and push intervals
  collect_interval: 15s            # How often to collect metrics
  push_interval: 20s               # How often to push to server
  heartbeat_interval: 10s          # Heartbeat frequency
  
  # Retry settings
  push_timeout: 10s                # HTTP request timeout
  retry_attempts: 3                # Max retries on failure
  retry_backoff: 2s                # Initial backoff duration

# Metrics collection settings
metrics:
  system: true                     # Collect system metrics
  
  docker:
    enabled: true
    socket: "/var/run/docker.sock"
    
    # Monitoring mode (choose one)
    monitor_all: true              # Monitor all containers
    
    # OR use filters (monitor_all must be false)
    filters:
      labels:
        - "monitor=true"           # Only containers with this label
        - "env=production"
      names:
        - "api-*"                  # Pattern matching
        - "web-*"
      images:
        - "mycompany/*"            # Image pattern
    
    # Container-specific alert settings
    alerts:
      default:
        cpu_threshold: 80.0
        memory_threshold: 90.0
        restart_threshold: 5       # Alert if >5 restarts
        restart_window: 300s       # Within 5 minutes
      
      # Per-container overrides
      overrides:
        - name: "postgres"
          memory_threshold: 95.0   # Higher threshold for DB
        - name: "redis"
          cpu_threshold: 70.0      # Lower threshold for cache

# System-level alerts
alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

---

## Alert Configuration

### Understanding Alert Types

#### System Alerts

| Alert Type | Trigger Condition | Severity |
|------------|-------------------|----------|
| **high_cpu** | CPU > threshold% | Warning |
| **high_memory** | Memory > threshold% | Warning |
| **high_disk** | Disk > threshold% | Critical |
| **agent_offline** | No heartbeat for > timeout | Critical |

#### Container Alerts

| Alert Type | Trigger Condition | Severity |
|------------|-------------------|----------|
| **container_stopped** | State: running → exited/dead | Critical |
| **container_unhealthy** | Health status = unhealthy | Warning |
| **container_cpu_high** | CPU > threshold% | Warning |
| **container_memory_high** | Memory > threshold% | Critical |
| **container_oom** | OOM killed flag = true | Critical |
| **container_restarting** | Restarts > threshold in window | Warning |

### Configuring Thresholds

#### Server-Side (Global)

```yaml
# In server.yaml
alerting:
  system_cpu_threshold: 80.0       # 80% CPU
  system_memory_threshold: 85.0    # 85% memory
  system_disk_threshold: 90.0      # 90% disk
```

#### Agent-Side (Per-Agent)

```yaml
# In agent.yaml
alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

#### Per-Container Overrides

```yaml
# In agent.yaml
docker:
  alerts:
    default:
      cpu_threshold: 80.0
      memory_threshold: 90.0
    
    overrides:
      - name: "postgres"          # Exact name
        memory_threshold: 95.0
      
      - name: "api-*"             # Pattern (all api containers)
        cpu_threshold: 70.0
      
      - name: "worker-*"
        restart_threshold: 10     # Allow more restarts for workers
        restart_window: 600s      # Over 10 minutes
```

### Alert Deduplication

Prevents alert spam by not sending the same alert repeatedly.

```yaml
# In server.yaml
alerting:
  deduplication_enabled: true
  deduplication_window: 5m    # Don't repeat alert within 5 minutes
```

**How it works**:
1. Alert triggered: "Disk 95% on /data"
2. Alert sent to Google Chat
3. Same condition still true 30s later
4. Alert NOT sent (within 5min window)
5. After 5 minutes, if still true, alert sent again

---

## Google Chat Integration

### Step 1: Create Google Chat Webhook

1. Open Google Chat
2. Go to the space where you want alerts
3. Click space name → **Manage webhooks**
4. Click **+ Add webhook**
5. Name: "Saviour Alerts"
6. Avatar URL: (optional)
7. Click **SAVE**
8. Copy the webhook URL

### Step 2: Configure Server

```yaml
# In server.yaml
google_chat:
  enabled: true
  webhook_url: "https://chat.googleapis.com/v1/spaces/AAAA.../messages?key=..."
  dashboard_url: "https://saviour.company.com"  # Optional
```

**Using Environment Variable** (Recommended):

```bash
# Set environment variable
export GOOGLE_CHAT_WEBHOOK_URL="https://chat.googleapis.com/v1/spaces/..."

# In server.yaml, reference it:
google_chat:
  enabled: true
  webhook_url: "${GOOGLE_CHAT_WEBHOOK_URL}"
```

### Step 3: Test Alert

```bash
# Restart server to apply configuration
sudo systemctl restart saviour-server

# Trigger a test alert by using high CPU
stress-ng --cpu 8 --timeout 60s

# Or manually trigger by exceeding disk threshold
dd if=/dev/zero of=/tmp/testfile bs=1G count=10
```

### Alert Message Format

Alerts appear as rich cards in Google Chat:

```
🚨 CRITICAL ALERT
ec2-prod-web-01

Alert Type: disk_critical
Severity: critical

🚨 High Disk Usage
Agent: ec2-prod-web-01
Mount: /data
Usage: 95.2%

Triggered: 2026-01-28T10:15:30+00:00

[View Dashboard]
```

---

## Monitoring Containers

### Monitor All Containers

Simplest approach - monitor everything:

```yaml
# In agent.yaml
docker:
  enabled: true
  monitor_all: true
```

### Filter by Labels

Best for production - label containers explicitly:

```yaml
# In docker-compose.yml or docker run
labels:
  - "monitor=true"
  - "env=production"
  - "team=backend"

# In agent.yaml
docker:
  monitor_all: false
  filters:
    labels:
      - "monitor=true"
      - "env=production"
```

### Filter by Name Pattern

Monitor specific services:

```yaml
# In agent.yaml
docker:
  monitor_all: false
  filters:
    names:
      - "api-*"        # api-v1, api-v2, api-gateway
      - "web-*"        # web-frontend, web-backend
      - "worker-*"     # worker-1, worker-2, worker-service
```

### Filter by Image

Monitor all containers from specific images:

```yaml
# In agent.yaml
docker:
  monitor_all: false
  filters:
    images:
      - "mycompany/*"              # All images from mycompany registry
      - "nginx:*"                  # All nginx versions
      - "postgres:14*"             # Postgres 14.x versions
```

### Combining Filters

All filters are OR'd together - container matches if ANY filter matches:

```yaml
docker:
  monitor_all: false
  filters:
    labels: ["monitor=true"]
    names: ["critical-*"]
    images: ["mycompany/api:*"]

# Monitors containers that have:
# - Label "monitor=true" OR
# - Name starting with "critical-" OR
# - Image matching "mycompany/api:*"
```

### Per-Container Alert Overrides

Different thresholds for different containers:

```yaml
docker:
  alerts:
    default:
      cpu_threshold: 80.0
      memory_threshold: 90.0
      restart_threshold: 5
    
    overrides:
      # Database needs more memory
      - name: "postgres"
        memory_threshold: 95.0
      
      # Cache is CPU intensive
      - name: "redis"
        cpu_threshold: 60.0
      
      # Workers can restart more often
      - name: "worker-*"
        restart_threshold: 10
        restart_window: 600s
      
      # Critical API - strict thresholds
      - name: "api-critical"
        cpu_threshold: 70.0
        memory_threshold: 85.0
        restart_threshold: 2
```

---

## Troubleshooting

### Server Issues

#### Server won't start

```bash
# Check logs
sudo journalctl -u saviour-server -n 50

# Common issues:
# 1. Port already in use
sudo lsof -i :8080

# 2. Invalid configuration
saviour-server -config /etc/saviour/server.yaml

# 3. Permission issues
ls -la /etc/saviour/server.yaml
```

#### No agents connecting

```bash
# Check server is listening
netstat -tuln | grep 8080

# Check firewall
sudo iptables -L -n | grep 8080

# Test from agent machine
curl http://server-ip:8080/api/v1/health
```

### Agent Issues

#### Agent can't connect to server

```bash
# Check logs
sudo journalctl -u saviour-agent -n 50

# Test connectivity
curl -v http://server-ip:8080/api/v1/health

# Test with API key
curl -H "Authorization: Bearer sk_prod_your-key" \
  http://server-ip:8080/api/v1/health
```

#### Docker metrics not collected

```bash
# Check Docker socket permissions
ls -la /var/run/docker.sock

# Test Docker access
docker ps

# If permission denied:
sudo usermod -aG docker $USER
# Or run agent as root (systemd service already does this)
```

#### High CPU usage from agent

```bash
# Check collection interval
grep collect_interval /etc/saviour/agent.yaml

# Reduce collection frequency
collect_interval: 30s  # Instead of 15s
push_interval: 60s     # Instead of 20s
```

### Alert Issues

#### Not receiving alerts

```bash
# Check server logs
sudo journalctl -u saviour-server | grep -i alert

# Verify alerting is enabled
grep -A 5 "alerting:" /etc/saviour/server.yaml

# Test Google Chat webhook
curl -X POST "your-webhook-url" \
  -H "Content-Type: application/json" \
  -d '{"text": "Test message from Saviour"}'
```

#### Too many alerts (spam)

```bash
# Enable deduplication
# In server.yaml:
alerting:
  deduplication_enabled: true
  deduplication_window: 5m  # Or longer: 15m, 30m

# Or increase thresholds
system_cpu_threshold: 90.0    # From 80.0
system_memory_threshold: 92.0  # From 85.0
```

---

## Best Practices

### Security

1. **Use Strong API Keys**
   ```bash
   # Generate secure random keys
   openssl rand -hex 32
   ```

2. **Store Keys Securely**
   - Use environment variables
   - Use secrets management (AWS Secrets Manager, Vault)
   - Never commit keys to git

3. **Use HTTPS in Production**
   - Put server behind reverse proxy (Nginx, Caddy)
   - Use Let's Encrypt for free TLS certificates

4. **Restrict Network Access**
   - Use security groups to limit access
   - Only allow agent IPs to reach server
   - Use VPC for internal communication

### Performance

1. **Adjust Collection Intervals**
   ```yaml
   # High-frequency (more data, more load)
   collect_interval: 10s
   push_interval: 15s
   
   # Low-frequency (less data, less load)
   collect_interval: 30s
   push_interval: 60s
   ```

2. **Filter Containers**
   - Don't monitor all containers if not needed
   - Use labels to mark important containers
   - Exclude system containers

3. **Tune Alert Thresholds**
   - Start with defaults (80%, 85%, 90%)
   - Adjust based on your infrastructure
   - Use overrides for special cases

### Reliability

1. **Use Systemd for Auto-Restart**
   ```ini
   [Service]
   Restart=always
   RestartSec=10
   ```

2. **Monitor the Monitor**
   - Set up external monitoring for Saviour server
   - Use AWS CloudWatch, Datadog, or similar
   - Alert if server becomes unreachable

3. **Regular Backups**
   ```bash
   # Backup configurations
   tar -czf saviour-config-backup.tar.gz /etc/saviour/
   ```

### Operational

1. **Log Rotation**
   ```bash
   # Configure journald
   sudo vim /etc/systemd/journald.conf
   ```
   ```ini
   [Journal]
   SystemMaxUse=1G
   MaxFileSec=7day
   ```

2. **Health Checks**
   ```bash
   # Add to cron
   */5 * * * * curl -f http://localhost:8080/api/v1/health || systemctl restart saviour-server
   ```

3. **Gradual Rollout**
   - Deploy to dev environment first
   - Test for 24 hours
   - Deploy to staging
   - Finally deploy to production

---

## FAQ

### General

**Q: Do I need to install an agent on every EC2 instance?**  
A: Yes, each instance needs an agent to collect and push metrics.

**Q: Can I monitor non-Docker workloads?**  
A: Yes! System metrics (CPU, memory, disk) work without Docker. Just disable Docker monitoring.

**Q: How much resource does the agent use?**  
A: Very minimal:
- CPU: <1%
- Memory: ~20MB
- Network: ~1KB/s (compressed)

**Q: What happens if the server goes down?**  
A: Agents will retry with exponential backoff. Metrics are not queued (fire-and-forget).

### Configuration

**Q: Can I use the same API key for all agents?**  
A: Yes, but it's better to use different keys per environment (dev, staging, prod) for security.

**Q: How do I change thresholds without restarting?**  
A: Currently requires restart. Hot-reload is planned for Phase 4.

**Q: Can I disable certain alerts?**  
A: Yes, set threshold to 0 or 100:
```yaml
system_cpu_threshold: 0  # Disables CPU alerts
```

### Alerts

**Q: Why am I getting duplicate alerts?**  
A: Enable deduplication:
```yaml
alerting:
  deduplication_enabled: true
  deduplication_window: 5m
```

**Q: Can I send alerts to Slack/PagerDuty/Email?**  
A: Currently only Google Chat is supported. Other integrations planned for Phase 4.

**Q: How do I acknowledge alerts?**  
A: Manual acknowledgment UI is planned for Phase 4. Currently alerts auto-resolve when condition clears.

### Deployment

**Q: Can I run multiple servers for high availability?**  
A: Not yet. HA is planned for Phase 4 (with load balancer).

**Q: What's the maximum number of agents supported?**  
A: Tested with 100 agents. Should handle 1000+ with proper server resources.

**Q: Do I need a database?**  
A: No! Saviour uses in-memory state store. Optional SQLite support planned for Phase 4.

### Troubleshooting

**Q: Agent says "connection refused"**  
A: Check firewall, server is running, and URL is correct:
```bash
curl http://server-ip:8080/api/v1/health
```

**Q: Container metrics show 0 for everything**  
A: Docker socket permission issue:
```bash
sudo chmod 666 /var/run/docker.sock
# Or add user to docker group
```

**Q: Memory keeps growing on server**  
A: This is a known issue. Restart server weekly until fix is deployed:
```bash
# Add to cron
0 2 * * 0 systemctl restart saviour-server
```

---

## Next Steps

1. ✅ **Complete Setup**: Server and agents running
2. 📊 **Monitor Metrics**: Check `/api/v1/health` endpoint
3. 🔔 **Configure Alerts**: Set up Google Chat webhook
4. 🎯 **Tune Thresholds**: Adjust based on your infrastructure
5. 📈 **Phase 4**: Wait for web dashboard release!

---

**Need Help?**
- GitHub Issues: https://github.com/yanurag-dev/saviour/issues
- Documentation: https://docs.saviour.dev
- Email: support@saviour.dev

---

**Made with ❤️ for reliable infrastructure**
