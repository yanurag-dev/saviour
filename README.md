# Saviour - Server Monitoring Tool

A lightweight, open-source server monitoring tool built in Go, designed for internal teams to monitor infrastructure health, performance, and availability.

## Features (MVP)

✅ **System Metrics Collection**
- CPU usage (overall and per-core)
- Memory usage (RAM and swap)
- Disk usage (per mount point)
- Network I/O statistics
- System information (OS, uptime, kernel)

✅ **Real-time Monitoring**
- Configurable collection intervals
- Live metrics display with emoji-rich output
- JSON output for integration

✅ **Alert Thresholds**
- CPU usage alerts
- Memory usage alerts
- Disk usage alerts
- Configurable thresholds per metric

✅ **YAML Configuration**
- Simple, readable configuration
- Sensible defaults
- Per-agent customization

## Quick Start

### Option 1: Docker (Recommended)

The easiest way to run Saviour is with Docker:

```bash
# Build and run with docker-compose
make docker-run

# Or manually
docker build -t saviour-agent:latest .
docker run -d \
  --name saviour-agent \
  --network host \
  --cap-add SYS_ADMIN \
  --cap-add NET_ADMIN \
  -v $(pwd)/agent.yaml:/etc/saviour/agent.yaml:ro \
  saviour-agent:latest
```

View logs:
```bash
make docker-logs
# Or: docker logs -f saviour-agent
```

Stop:
```bash
make docker-stop
# Or: docker-compose down
```

### Option 2: Build from Source

#### 1. Build the Agent

```bash
go mod tidy
go build -o bin/saviour-agent ./cmd/agent
```

#### 2. Create Configuration

Create an `agent.yaml` file:

```yaml
agent:
  name: "my-server"
  collect_interval: 10s

metrics:
  system: true
  disk_mounts:
    - "/"

alerts:
  cpu_threshold: 80.0
  memory_threshold: 85.0
  disk_threshold: 90.0
```

#### 3. Run the Agent

```bash
./bin/saviour-agent -config agent.yaml
```

## Example Output

```
[saviour-agent] 📊 Metrics collected at 2026-01-27T20:45:01+05:30
[saviour-agent] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[saviour-agent] 🖥️  System: my-server (darwin darwin)
[saviour-agent]    Uptime: 11d 6h 26m
[saviour-agent] 💻 CPU Usage: 21.88%
[saviour-agent]    Load Avg: 9.83 (1m) | 6.16 (5m) | 8.07 (15m)
[saviour-agent] 🧠 Memory: 83.04% used (6.6 GiB / 8.0 GiB)
[saviour-agent]    Swap: 90.60% used (11.8 GiB / 13.0 GiB)
[saviour-agent] 💾 Disk Usage:
[saviour-agent]    /: 89.72% used (204.8 GiB / 228.3 GiB)
[saviour-agent] 🌐 Network: ↑ 8.3 GiB sent | ↓ 88.0 GiB received
```

When thresholds are exceeded:

```
[saviour-agent] ⚠️  ALERT: Memory usage (86.85%) exceeds threshold (85.00%)
```

## Project Structure

```
saviour/
├── cmd/
│   └── agent/          # Agent binary
├── internal/
│   ├── agent/          # Agent logic
│   ├── collector/      # Metric collectors
│   └── config/         # Configuration parsing
├── pkg/
│   └── metrics/        # Shared metric types
├── examples/           # Example configs
└── bin/                # Compiled binaries
```

## Configuration Reference

See `examples/agent.yaml` for a complete configuration example with all available options.

### Agent Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `agent.name` | string | hostname | Agent identifier |
| `agent.collect_interval` | duration | 10s | How often to collect metrics |
| `agent.server_url` | string | - | Central server URL (future use) |
| `agent.api_key` | string | - | API key for authentication (future use) |

### Metrics Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `metrics.system` | bool | false | Enable system metrics collection |
| `metrics.disk_mounts` | []string | all | Specific mount points to monitor |

### Alert Thresholds

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `alerts.cpu_threshold` | float | 0 | CPU usage percentage (0-100) |
| `alerts.memory_threshold` | float | 0 | Memory usage percentage (0-100) |
| `alerts.disk_threshold` | float | 0 | Disk usage percentage (0-100) |

## Development Status

**Current Phase**: MVP Agent (✅ Complete)

### Completed
- [x] Project structure
- [x] System metrics collection (CPU, memory, disk, network)
- [x] YAML configuration
- [x] Alert thresholds
- [x] Pretty-print output
- [x] JSON export
- [x] Docker support with multi-stage builds
- [x] Docker Compose deployment

### Roadmap
- [ ] Process monitoring
- [ ] Health checks (HTTP, TCP, ping)
- [ ] Central server with REST API
- [ ] Agent-to-server communication
- [ ] Time-series data storage
- [ ] Web dashboard
- [ ] Multi-server aggregation
- [ ] Alert channels (email, Slack, webhooks)

## Docker Deployment

### Docker Image Features

- **Multi-stage build** - Minimal final image (~10MB)
- **Scratch-based** - Maximum security, no unnecessary packages
- **Static binary** - No runtime dependencies
- **Non-root user** - Runs as UID 65534 for security
- **Host metrics** - Access to host system metrics via network_mode: host

### Docker Configuration

The Docker image expects configuration at `/etc/saviour/agent.yaml`. Mount your config:

```bash
docker run -d \
  --name saviour-agent \
  --network host \
  --cap-add SYS_ADMIN \
  --cap-add NET_ADMIN \
  -v /path/to/your/agent.yaml:/etc/saviour/agent.yaml:ro \
  saviour-agent:latest
```

### Docker Compose

For production deployments, use docker-compose:

```yaml
version: '3.8'

services:
  agent:
    image: saviour-agent:latest
    container_name: saviour-agent
    restart: unless-stopped
    network_mode: host
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
    volumes:
      - ./agent.yaml:/etc/saviour/agent.yaml:ro
```

Run with: `docker-compose up -d`

### Makefile Commands

```bash
make docker-build    # Build Docker image
make docker-run      # Build and run with docker-compose
make docker-stop     # Stop containers
make docker-clean    # Stop and remove image
make docker-logs     # View container logs
make docker-rebuild  # Clean rebuild
```

## Requirements

### Native Build
- Go 1.21 or higher
- Tested on: macOS, Linux (Windows support planned)

### Docker
- Docker 20.10+ or Docker Desktop
- Docker Compose v2+ (optional, for docker-compose.yml)

## Dependencies

- [gopsutil](https://github.com/shirou/gopsutil) - Cross-platform system metrics
- [yaml.v3](https://github.com/go-yaml/yaml) - YAML parsing

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! This project is in early development. See PROJECT_DESIGN.md for the full vision.

---

**Status**: MVP Agent Complete
**Version**: 0.1.0
**Last Updated**: 2026-01-27
