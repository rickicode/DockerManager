package docker

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	dmtypes "docker-manager/internal/types"
)

// ListContainers returns all containers (optionally filtered by status)
func (c *Client) ListContainers(ctx context.Context, all bool) ([]container.Summary, error) {
	containers, err := c.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	return containers, nil
}

// GetContainer returns detailed info about a single container
func (c *Client) GetContainer(ctx context.Context, id string) (*container.InspectResponse, error) {
	inspected, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", id, err)
	}
	return &inspected, nil
}

// GetContainerLogs returns logs from a container
func (c *Client) GetContainerLogs(ctx context.Context, id string, tail string) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}
	if tail == "" {
		options.Tail = "100"
	}

	reader, err := c.ContainerLogs(ctx, id, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for container %s: %w", id, err)
	}
	defer reader.Close()

	// Docker log stream uses 8-byte headers for multiplexing stdout/stderr.
	// We strip these headers to get clean log text.
	// For non-follow mode, the entire log comes as a single frame, so we just
	// strip the 8-byte header from each chunk.
	var logBuf bytes.Buffer
	tmpBuf := make([]byte, 1024*64)
	for {
		n, err := reader.Read(tmpBuf)
		if n > 8 {
			// Strip 8-byte Docker multiplexing header
			logBuf.Write(tmpBuf[8:n])
		} else if n > 0 {
			logBuf.Write(tmpBuf[:n])
		}
		if err != nil {
			break
		}
	}

	return logBuf.String(), nil
}

// CreateContainer creates a new container with the given configuration
func (c *Client) CreateContainer(ctx context.Context, cfg dmtypes.ContainerConfig) (*container.CreateResponse, error) {
	// Pull image if needed
	if err := c.PullImageIfNotExists(ctx, cfg.Image); err != nil {
		return nil, err
	}

	// Prepare port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for _, p := range cfg.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		portKey := nat.Port(fmt.Sprintf("%d/%s", p.ContainerPort, proto))
		exposedPorts[portKey] = struct{}{}

		if p.HostPort > 0 {
			portBindings[portKey] = []nat.PortBinding{
				{HostPort: fmt.Sprintf("%d", p.HostPort)},
			}
		}
	}

	// Prepare environment variables
	envVars := []string{}
	for k, v := range cfg.Env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Prepare volumes/bind mounts
	hostMounts := []mount.Mount{}
	for _, v := range cfg.Volumes {
		hostMounts = append(hostMounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   v.HostPath,
			Target:   v.ContainerPath,
			ReadOnly: v.ReadOnly,
		})
	}

	// Prepare restart policy
	restartPolicy := container.RestartPolicy{}
	switch cfg.RestartPolicy {
	case "always":
		restartPolicy.Name = container.RestartPolicyAlways
	case "unless-stopped":
		restartPolicy.Name = container.RestartPolicyUnlessStopped
	case "on-failure":
		restartPolicy.Name = container.RestartPolicyOnFailure
	case "no":
		restartPolicy.Name = container.RestartPolicyDisabled
	}

	// Build container config
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Cmd:          cfg.Command,
		Env:          envVars,
		ExposedPorts: exposedPorts,
		Labels:       cfg.Labels,
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		Mounts:        hostMounts,
		RestartPolicy: restartPolicy,
	}

	// Handle network mode
	networkingConfig := &network.NetworkingConfig{}
	if cfg.NetworkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(cfg.NetworkMode)
		if cfg.NetworkName != "" {
			networkingConfig.EndpointsConfig = map[string]*network.EndpointSettings{
				cfg.NetworkName: {},
			}
		}
	}

	resp, err := c.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &resp, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, id string) error {
	timeout := 30
	return c.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	timeout := 30
	return c.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer removes a container (optionally force)
func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return c.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

// ToContainerInfo converts Docker container Summary to our API type
func ToContainerInfo(s container.Summary) dmtypes.ContainerInfo {
	ports := []dmtypes.PortMapping{}
	for _, p := range s.Ports {
		ports = append(ports, dmtypes.PortMapping{
			HostPort:      int(p.PublicPort),
			ContainerPort: int(p.PrivatePort),
			Protocol:      p.Type,
		})
	}

	name := ""
	if len(s.Names) > 0 {
		name = s.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
	}

	id := s.ID
	if len(id) > 12 {
		id = id[:12]
	}

	return dmtypes.ContainerInfo{
		ID:      id,
		Name:    name,
		Image:   s.Image,
		Status:  s.Status,
		State:   string(s.State),
		Ports:   ports,
		Created: time.Unix(s.Created, 0).Format(time.RFC3339),
		Labels:  s.Labels,
	}
}

// ToContainerInfoDetailed converts detailed container inspect to our API type
func ToContainerInfoDetailed(insp container.InspectResponse) dmtypes.ContainerInfo {
	ports := []dmtypes.PortMapping{}
	if insp.NetworkSettings != nil {
		for portKey, bindings := range insp.NetworkSettings.Ports {
			containerPort := int(portKey.Int())
			proto := portKey.Proto()
			hostPort := 0
			if len(bindings) > 0 {
				fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
			}
			ports = append(ports, dmtypes.PortMapping{
				HostPort:      hostPort,
				ContainerPort: containerPort,
				Protocol:      proto,
			})
		}
	}

	volMounts := []dmtypes.VolumeMapping{}
	for _, m := range insp.Mounts {
		volMounts = append(volMounts, dmtypes.VolumeMapping{
			HostPath:      m.Source,
			ContainerPath: m.Destination,
			ReadOnly:      m.RW == false,
		})
	}

	ipAddress := ""
	networkMode := ""
	if insp.NetworkSettings != nil {
		for _, net := range insp.NetworkSettings.Networks {
			ipAddress = net.IPAddress
			break
		}
	}

	if insp.HostConfig != nil {
		networkMode = string(insp.HostConfig.NetworkMode)
	}

	restartPolicy := ""
	if insp.HostConfig != nil {
		restartPolicy = string(insp.HostConfig.RestartPolicy.Name)
	}

	name := insp.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	state := ""
	if insp.State != nil {
		state = string(insp.State.Status)
	}

	var labels map[string]string
	env := []string{}
	if insp.Config != nil {
		env = insp.Config.Env
		labels = insp.Config.Labels
	}

	id := insp.ID
	if len(id) > 12 {
		id = id[:12]
	}

	return dmtypes.ContainerInfo{
		ID:            id,
		Name:          name,
		Image:         insp.Image,
		Status:        state,
		State:         state,
		Ports:         ports,
		Created:       insp.Created,
		IPAddress:     ipAddress,
		NetworkMode:   networkMode,
		Mounts:        volMounts,
		Env:           env,
		Labels:        labels,
		RestartPolicy: restartPolicy,
	}
}

// CheckPort checks if a port is open on a host using a Docker container
func (c *Client) CheckPort(ctx context.Context, host string, port int) (bool, string, error) {
	containerName := fmt.Sprintf("port-check-%d-%d", port, time.Now().UnixNano())

	// Try to pull alpine first, ignore errors
	pullReader, pullErr := c.ImagePull(ctx, "alpine:latest", image.PullOptions{})
	if pullErr == nil {
		buf := make([]byte, 4096)
		for {
			_, err := pullReader.Read(buf)
			if err != nil {
				break
			}
		}
		pullReader.Close()
	}

	resp, err := c.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sh", "-c", fmt.Sprintf("nc -zv -w3 %s %d 2>&1 || true", host, port)},
	}, nil, nil, nil, containerName)
	if err != nil {
		return false, "", fmt.Errorf("failed to create port-check container: %w", err)
	}

	// Use context.Background() for cleanup to avoid using a cancelled context
	defer c.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	if err := c.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return false, "", fmt.Errorf("failed to start port-check container: %w", err)
	}

	// Wait for completion using the channels
	resultC, errC := c.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errC:
		if err != nil {
			return false, "", err
		}
	case <-resultC:
	case <-ctx.Done():
		return false, "", ctx.Err()
	}

	// Get logs (use background context for reading since container may have stopped)
	logCtx := context.Background()
	reader, err := c.ContainerLogs(logCtx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return false, "", err
	}
	defer reader.Close()

	var logBuf bytes.Buffer
	tmpBuf := make([]byte, 4096)
	for {
		n, err := reader.Read(tmpBuf)
		if n > 0 {
			data := tmpBuf[:n]
			if len(data) > 8 {
				logBuf.Write(data[8:])
			}
		}
		if err != nil {
			break
		}
	}
	output := logBuf.String()

	// Parse output - if nc shows "succeeded" or similar, port is open
	open := false
	if containsAny(output, "succeeded", "open", "Connected to") {
		open = true
	}

	return open, "", nil
}

func containsAny(s string, substrs ...string) bool {
	lower := toLower(s)
	for _, sub := range substrs {
		if len(sub) > 0 && len(lower) > 0 {
			lowerSub := toLower(sub)
			for i := 0; i <= len(lower)-len(lowerSub); i++ {
				match := true
				for j := 0; j < len(lowerSub); j++ {
					if lower[i+j] != lowerSub[j] {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
