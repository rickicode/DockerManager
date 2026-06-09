package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"

	dmtypes "docker-manager/internal/types"
)

// ListNetworks returns all Docker networks
func (c *Client) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	networks, err := c.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	return networks, nil
}

// GetNetwork returns detailed info about a network
func (c *Client) GetNetwork(ctx context.Context, id string) (*network.Inspect, error) {
	inspected, err := c.NetworkInspect(ctx, id, network.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network %s: %w", id, err)
	}
	return &inspected, nil
}

// CreateNetwork creates a new Docker network
func (c *Client) CreateNetwork(ctx context.Context, cfg dmtypes.NetworkConfig) (*network.CreateResponse, error) {
	ipam := &network.IPAM{}
	if cfg.Subnet != "" {
		ipam.Config = []network.IPAMConfig{
			{
				Subnet:  cfg.Subnet,
				Gateway: cfg.Gateway,
			},
		}
	}

	driver := cfg.Driver
	if driver == "" {
		driver = "bridge"
	}

	resp, err := c.NetworkCreate(ctx, cfg.Name, network.CreateOptions{
		Driver: driver,
		IPAM:   ipam,
		Labels: cfg.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return &resp, nil
}

// RemoveNetwork removes a Docker network
func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return c.NetworkRemove(ctx, id)
}

// ConnectContainerToNetwork connects a container to a network
func (c *Client) ConnectContainerToNetwork(ctx context.Context, containerID, networkID string) error {
	return c.NetworkConnect(ctx, networkID, containerID, nil)
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (c *Client) DisconnectContainerFromNetwork(ctx context.Context, containerID, networkID string, force bool) error {
	return c.NetworkDisconnect(ctx, networkID, containerID, force)
}

// ToNetworkInfo converts Docker network summary to our API type
func ToNetworkInfo(n network.Summary) dmtypes.NetworkInfo {
	subnet := ""
	gateway := ""
	if len(n.IPAM.Config) > 0 {
		subnet = n.IPAM.Config[0].Subnet
		gateway = n.IPAM.Config[0].Gateway
	}

	id := n.ID
	if len(id) > 12 {
		id = id[:12]
	}

	return dmtypes.NetworkInfo{
		ID:         id,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Subnet:     subnet,
		Gateway:    gateway,
		Containers: len(n.Containers),
		Labels:     n.Labels,
	}
}
