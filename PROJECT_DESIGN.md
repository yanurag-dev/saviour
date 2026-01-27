# Saviour - Server Monitoring Tool

## Project Overview
Saviour is a lightweight, open-source server monitoring tool built in Go, designed for internal teams to monitor infrastructure health, performance, and availability. It provides real-time insights into server metrics, service health, and system performance with minimal overhead.

## Goals
- **Simple to deploy**: Single binary, minimal dependencies
- **Resource efficient**: Low CPU and memory footprint
- **Real-time monitoring**: Sub-second metric collection and alerting
- **Extensible**: Plugin architecture for custom metrics and checks
- **Open source friendly**: Well-documented, easy to contribute to

## Core Features (MVP)

### 1. System Metrics Collection
- **CPU**: Usage per core, load average, utilization percentages
- **Memory**: Total, used, available, swap usage
- **Disk**: Usage per mount point, I/O statistics, read/write rates
- **Network**: Bandwidth usage, packet statistics, connection counts
- **System Info**: Uptime, OS details, kernel version

### 2. Process Monitoring
- Track specific processes by name or PID
- CPU and memory usage per process
- Process state (running, sleeping, zombie)
- Auto-restart capabilities for critical processes
- Process count thresholds

### 3. Service Health Checks
- **HTTP/HTTPS**: Endpoint availability, response time, status codes
- **TCP**: Port connectivity checks
- **Ping**: ICMP reachability tests
- **Custom scripts**: Exit code based health checks
- Configurable check intervals and timeouts

### 4. Alerting System
- **Threshold-based alerts**: CPU > 80%, disk > 90%, etc.
- **Multiple channels**:
  - Email (SMTP)
  - Slack webhooks
  - Discord webhooks
  - PagerDuty
  - Custom webhooks
- Alert severity levels (info, warning, critical)
- Alert grouping and deduplication
- Cooldown periods to prevent alert storms

### 5. Data Storage
- Time-series database for historical metrics (embedded or external)
- Configurable retention periods
- Data aggregation for long-term storage efficiency
- SQLite for embedded mode, support for external DBs (PostgreSQL, InfluxDB)

### 6. Multi-Server Architecture
- **Agent mode**: Lightweight agent on each server, reports to central server
- **Agentless mode**: SSH-based monitoring (optional)
- Central server aggregates metrics from all agents
- Auto-discovery of new agents

### 7. REST API
- Query current and historical metrics
- Manage monitored servers and services
- Configure alerts and thresholds
- Health check endpoints
- Authentication and API key support

### 8. Web Dashboard
- Real-time metrics visualization
- Server overview page (all servers at a glance)
- Individual server detail pages
- Alert history and management
- Configuration management UI
- Dark/light theme support

## Advanced Features (Future Roadmap)

### Phase 2
- **Log aggregation**: Collect and search logs from monitored servers
- **Container monitoring**: Docker and Kubernetes integration
- **Cloud integration**: AWS, GCP, Azure metrics
- **Anomaly detection**: ML-based threshold learning
- **Custom dashboards**: User-defined metric combinations

### Phase 3
- **Distributed tracing**: Application performance monitoring
- **Network topology**: Visualize server relationships and dependencies
- **Incident management**: Built-in ticketing and runbook execution
- **RBAC**: Role-based access control for teams
- **High availability**: Multi-master setup for central server

### Phase 4
- **Plugin marketplace**: Community-contributed plugins
- **Mobile app**: iOS/Android monitoring app
- **Predictive analytics**: Forecast resource usage trends
- **Compliance reporting**: Generate audit reports

## Technical Architecture

### Components
```
┌─────────────────────────────────────────────┐
│           Web Dashboard (React/Go)          │
│  ┌────────────┐  ┌─────────────────────┐  │
│  │  Frontend  │  │    REST API (Go)    │  │
│  └────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────┘
                     │
                     │ (HTTP/gRPC)
                     │
┌────────────────────┴────────────────────────┐
│         Central Server (Collector)          │
│  ┌──────────────────────────────────────┐  │
│  │ Metric Aggregator │ Alert Manager    │  │
│  │ Time-series DB    │ Configuration    │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
          │              │              │
          │              │              │
   ┌──────┴──────┐ ┌────┴─────┐ ┌─────┴──────┐
   │   Agent 1   │ │ Agent 2  │ │  Agent N   │
   │   (Server)  │ │ (Server) │ │  (Server)  │
   └─────────────┘ └──────────┘ └────────────┘
```

### Technology Stack
- **Language**: Go 1.21+
- **Web Framework**: Chi/Gin (lightweight HTTP routers)
- **Database**: SQLite (embedded), PostgreSQL (production), InfluxDB (time-series)
- **Frontend**: HTML/CSS/JS (or lightweight React if needed)
- **Communication**: HTTP/gRPC for agent-server communication
- **Configuration**: YAML-based configuration files
- **Metrics Collection**:
  - `gopsutil` (cross-platform system metrics)
  - Custom collectors for specific checks

### Project Structure
```
saviour/
├── cmd/
│   ├── agent/          # Agent binary
│   ├── server/         # Central server binary
│   └── cli/            # CLI tool for management
├── internal/
│   ├── agent/          # Agent logic
│   ├── server/         # Server logic
│   ├── collector/      # Metric collectors
│   ├── storage/        # Database abstractions
│   ├── alerting/       # Alert system
│   ├── api/            # REST API handlers
│   └── config/         # Configuration parsing
├── pkg/
│   ├── metrics/        # Shared metric types
│   └── protocol/       # Agent-server protocol
├── web/                # Dashboard frontend
├── docs/               # Documentation
├── examples/           # Example configs
└── scripts/            # Build and deployment scripts
```

## Deployment Models

### 1. Standalone Mode
- Single server monitors itself
- Embedded SQLite database
- Perfect for small teams or single-server setups

### 2. Distributed Mode
- Central server + multiple agents
- Agents report to central server
- Horizontal scaling for large infrastructures

### 3. Docker/Kubernetes
- Containerized deployment
- Helm charts for Kubernetes
- Sidecar pattern for application monitoring

## Configuration Example

```yaml
# agent.yaml
agent:
  name: "web-server-01"
  server_url: "https://monitoring.company.com"
  api_key: "your-api-key"
  collect_interval: 10s

metrics:
  system: true
  processes:
    - name: "nginx"
      restart_on_failure: true
    - name: "postgres"
  disk_mounts:
    - "/"
    - "/data"

health_checks:
  - name: "web-app"
    type: "http"
    url: "http://localhost:8080/health"
    interval: 30s
    timeout: 5s
  - name: "postgres"
    type: "tcp"
    host: "localhost"
    port: 5432
    interval: 60s

alerts:
  cpu_threshold: 80
  memory_threshold: 85
  disk_threshold: 90
```

## Use Cases

### Internal Team
- Monitor production, staging, and development servers
- Alert on-call engineers when services are down
- Track resource usage for capacity planning
- Quick troubleshooting with historical data

### Open Source Community
- Self-hosted alternative to commercial monitoring tools
- Customizable for specific infrastructure needs
- Learn about Go and monitoring system architecture
- Contribute plugins for different services and platforms

## Success Metrics
- **Performance**: Handle 1000+ agents with sub-second latency
- **Reliability**: 99.9% uptime for central server
- **Ease of use**: < 5 minutes from download to first metrics
- **Community**: Active contributor base, documented plugin API

## Next Steps
1. Set up project structure and repository
2. Implement basic agent with system metrics collection
3. Build central server with REST API
4. Create simple web dashboard
5. Add alerting system
6. Write comprehensive documentation
7. Create demo environment
8. Open source release with contribution guidelines

---

**Status**: Design Document
**Version**: 1.0
**Last Updated**: 2026-01-27
