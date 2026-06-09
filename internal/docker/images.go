package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/image"

	dmtypes "docker-manager/internal/types"
)

// ListImages returns all images
func (c *Client) ListImages(ctx context.Context) ([]image.Summary, error) {
	images, err := c.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return images, nil
}

// PullImage pulls a Docker image
func (c *Client) PullImage(ctx context.Context, imageName string) error {
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

// RemoveImage removes an image
func (c *Client) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := c.ImageRemove(ctx, id, image.RemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", id, err)
	}
	return nil
}

// ToImageInfo converts Docker image summary to our API type
func ToImageInfo(img image.Summary) dmtypes.ImageInfo {
	id := img.ID
	if len(id) > 19 {
		id = id[7:19]
	} else if len(id) > 12 {
		id = id[:12]
	}

	return dmtypes.ImageInfo{
		ID:       id,
		RepoTags: img.RepoTags,
		Size:     img.Size,
		Created:  time.Unix(img.Created, 0).Format(time.RFC3339),
		Labels:   img.Labels,
	}
}
