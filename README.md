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

### 1. Build the Agent

```bash
go mod tidy
go build -o bin/saviour-agent ./cmd/agent
```

### 2. Create Configuration

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

### 3. Run the Agent

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

### Roadmap
- [ ] Process monitoring
- [ ] Health checks (HTTP, TCP, ping)
- [ ] Central server with REST API
- [ ] Agent-to-server communication
- [ ] Time-series data storage
- [ ] Web dashboard
- [ ] Multi-server aggregation
- [ ] Alert channels (email, Slack, webhooks)

## Requirements

- Go 1.21 or higher
- Tested on: macOS, Linux (Windows support planned)

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
