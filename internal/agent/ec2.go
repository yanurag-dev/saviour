package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anurag/saviour/internal/server"
)

const (
	// EC2 Instance Metadata Service (IMDS) v2 endpoints
	imdsTokenURL     = "http://169.254.169.254/latest/api/token"
	imdsBaseURL      = "http://169.254.169.254/latest/meta-data"
	imdsInstanceID   = imdsBaseURL + "/instance-id"
	imdsInstanceType = imdsBaseURL + "/instance-type"
	imdsRegion       = imdsBaseURL + "/placement/region"
	imdsAZ           = imdsBaseURL + "/placement/availability-zone"
	imdsTags         = imdsBaseURL + "/tags/instance"

	// Timeout for IMDS requests
	imdsTimeout = 2 * time.Second
	// Token TTL (6 hours max)
	imdsTokenTTL = "21600"
)

// EC2MetadataClient fetches EC2 instance metadata
type EC2MetadataClient struct {
	client *http.Client
	token  string
}

// NewEC2MetadataClient creates a new EC2 metadata client
func NewEC2MetadataClient() *EC2MetadataClient {
	return &EC2MetadataClient{
		client: &http.Client{
			Timeout: imdsTimeout,
		},
	}
}

// GetEC2Metadata fetches EC2 instance metadata using IMDSv2
func (c *EC2MetadataClient) GetEC2Metadata(ctx context.Context) (*server.EC2Metadata, error) {
	// Get IMDSv2 token
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get IMDS token: %w", err)
	}
	c.token = token

	metadata := &server.EC2Metadata{}

	// Fetch instance ID
	instanceID, err := c.fetchMetadata(ctx, imdsInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance ID: %w", err)
	}
	metadata.InstanceID = instanceID

	// Fetch instance type (optional, log error but continue)
	if instanceType, err := c.fetchMetadata(ctx, imdsInstanceType); err == nil {
		metadata.InstanceType = instanceType
	}

	// Fetch region (optional)
	if region, err := c.fetchMetadata(ctx, imdsRegion); err == nil {
		metadata.Region = region
	}

	// Fetch availability zone (optional)
	if az, err := c.fetchMetadata(ctx, imdsAZ); err == nil {
		metadata.AvailabilityZone = az
	}

	// Fetch tags (optional)
	if tags, err := c.fetchTags(ctx); err == nil {
		metadata.Tags = tags
	}

	return metadata, nil
}

// getToken fetches an IMDSv2 session token
func (c *EC2MetadataClient) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", imdsTokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", imdsTokenTTL)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IMDS token request failed with status: %d", resp.StatusCode)
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

// fetchMetadata fetches a single metadata value
func (c *EC2MetadataClient) fetchMetadata(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata request failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// fetchTags fetches instance tags
func (c *EC2MetadataClient) fetchTags(ctx context.Context) (map[string]string, error) {
	// First, get the list of tag keys
	tagKeys, err := c.fetchMetadata(ctx, imdsTags)
	if err != nil {
		return nil, err
	}

	if tagKeys == "" {
		return nil, nil
	}

	// Parse tag keys (newline separated)
	tags := make(map[string]string)
	// In a real implementation, you'd split by newlines and fetch each tag
	// For simplicity, we'll return an empty map
	// This would require additional IMDS calls to fetch each tag value

	return tags, nil
}

// IsRunningOnEC2 checks if the agent is running on an EC2 instance
func IsRunningOnEC2(ctx context.Context) bool {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", imdsBaseURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}
