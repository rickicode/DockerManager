package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client wraps the Docker client with additional functionality
type Client struct {
	*client.Client
}

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Client{cli}, nil
}

// MustNewClient creates a new Docker client or panics
func MustNewClient() *Client {
	c, err := NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return c
}

// PullImageIfNotExists pulls an image if it doesn't exist locally
func (c *Client) PullImageIfNotExists(ctx context.Context, imageName string) error {
	_, err := c.ImageInspect(ctx, imageName)
	if err == nil {
		return nil // Image exists
	}

	fmt.Printf("Pulling image: %s...\n", imageName)
	reader, err := c.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}
	return nil
}
