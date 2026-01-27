package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// Client wraps the Docker client with our custom methods
type Client struct {
	cli    *client.Client
	filter FilterConfig
}

// NewClient creates a new Docker client
func NewClient(socketPath string, filterConfig FilterConfig) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	// Override socket path if provided
	if socketPath != "" {
		opts = append(opts, client.WithHost("unix://"+socketPath))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{
		cli:    cli,
		filter: filterConfig,
	}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// Ping tests the connection to Docker daemon
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// ListContainers returns all containers matching the filter criteria
func (c *Client) ListContainers(ctx context.Context) ([]types.Container, error) {
	opts := container.ListOptions{
		All: true, // Include stopped containers to detect state changes
	}

	// Apply filters if not monitoring all
	if !c.filter.MonitorAll {
		args := filters.NewArgs()

		// Filter by labels
		for _, label := range c.filter.Labels {
			args.Add("label", label)
		}

		opts.Filters = args
	}

	containers, err := c.cli.ContainerList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Post-filter by name and image patterns (Docker API doesn't support wildcards)
	if !c.filter.MonitorAll {
		containers = c.filterByPatterns(containers)
	}

	return containers, nil
}

// filterByPatterns applies name and image pattern matching
func (c *Client) filterByPatterns(containers []types.Container) []types.Container {
	if len(c.filter.Names) == 0 && len(c.filter.Images) == 0 {
		return containers
	}

	filtered := []types.Container{}
	for _, container := range containers {
		match := false

		// Check name patterns
		if len(c.filter.Names) > 0 {
			for _, name := range container.Names {
				// Remove leading slash from container name
				name = strings.TrimPrefix(name, "/")
				for _, pattern := range c.filter.Names {
					if matched, _ := filepath.Match(pattern, name); matched {
						match = true
						break
					}
				}
				if match {
					break
				}
			}
		}

		// Check image patterns
		if !match && len(c.filter.Images) > 0 {
			for _, pattern := range c.filter.Images {
				if matched, _ := filepath.Match(pattern, container.Image); matched {
					match = true
					break
				}
			}
		}

		if match {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

// InspectContainer gets detailed information about a container
func (c *Client) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	return inspect, nil
}

// GetContainerStats retrieves resource usage statistics for a container
func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false) // stream=false for single snapshot
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("container %s is not running", containerID)
		}
		return nil, fmt.Errorf("failed to decode stats for container %s: %w", containerID, err)
	}

	return &v, nil
}

// GetContainerInfo gets comprehensive information about a container
func (c *Client) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error) {
	// Inspect container for details
	inspect, err := c.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	info := &ContainerInfo{
		ID:      inspect.ID[:12], // Short ID
		Name:    strings.TrimPrefix(inspect.Name, "/"),
		Image:   inspect.Config.Image,
		ImageID: inspect.Image[:12],
		Labels:  inspect.Config.Labels,

		State:        inspect.State.Status,
		Status:       inspect.State.Status,
		ExitCode:     inspect.State.ExitCode,
		OOMKilled:    inspect.State.OOMKilled,
		RestartCount: inspect.RestartCount,
	}

	// Parse created timestamp
	if created, err := time.Parse(time.RFC3339Nano, inspect.Created); err == nil {
		info.Created = created
	}

	// Parse timestamps
	if startedAt, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
		info.StartedAt = startedAt
	}
	if finishedAt, err := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt); err == nil {
		info.FinishedAt = finishedAt
	}

	// Health status
	if inspect.State.Health != nil {
		info.Health = inspect.State.Health.Status
	} else {
		info.Health = "none"
	}

	// Get stats only if container is running
	if inspect.State.Running {
		stats, err := c.GetContainerStats(ctx, containerID)
		if err == nil {
			info.CPUPercent = calculateCPUPercent(stats)
			info.MemoryUsage = stats.MemoryStats.Usage
			info.MemoryLimit = stats.MemoryStats.Limit
			if stats.MemoryStats.Limit > 0 {
				info.MemoryPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
			}

			// Network I/O
			for _, network := range stats.Networks {
				info.NetworkRxBytes += network.RxBytes
				info.NetworkTxBytes += network.TxBytes
			}

			// Block I/O
			for _, blkio := range stats.BlkioStats.IoServiceBytesRecursive {
				if blkio.Op == "read" || blkio.Op == "Read" {
					info.BlockReadBytes += blkio.Value
				} else if blkio.Op == "write" || blkio.Op == "Write" {
					info.BlockWriteBytes += blkio.Value
				}
			}

			// PIDs
			info.PIDs = stats.PidsStats.Current
		}
	}

	return info, nil
}

// GetAllContainerInfo retrieves info for all monitored containers
func (c *Client) GetAllContainerInfo(ctx context.Context) ([]ContainerInfo, error) {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]ContainerInfo, 0, len(containers))
	for _, container := range containers {
		info, err := c.GetContainerInfo(ctx, container.ID)
		if err != nil {
			// Log error but continue with other containers
			continue
		}
		infos = append(infos, *info)
	}

	return infos, nil
}

// calculateCPUPercent calculates CPU usage percentage from stats
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	// CPU calculation based on Docker's algorithm
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)

	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	return 0.0
}
