package collector

import (
	"context"
	"fmt"
	"log"

	"github.com/anurag/saviour/internal/docker"
)

// DockerCollector collects Docker container metrics
type DockerCollector struct {
	client *docker.Client
	logger *log.Logger
}

// NewDockerCollector creates a new Docker collector
func NewDockerCollector(socketPath string, filterConfig docker.FilterConfig, logger *log.Logger) (*DockerCollector, error) {
	client, err := docker.NewClient(socketPath, filterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &DockerCollector{
		client: client,
		logger: logger,
	}, nil
}

// Collect gathers all container metrics
func (c *DockerCollector) Collect(ctx context.Context) ([]docker.ContainerInfo, error) {
	containers, err := c.client.GetAllContainerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect container info: %w", err)
	}

	return containers, nil
}

// Close closes the Docker client connection
func (c *DockerCollector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
