package tsdproxy

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	dockerclient "docker-manager/internal/docker"
	dmtypes "docker-manager/internal/types"
)

const (
	tsdproxyImage    = "almeidapaulopt/tsdproxy:2"
	tsdproxyLabel    = "tsdproxy.enable"
	tsdproxyName     = "dockermanager-tsdproxy"
	tsdproxyDataVol  = "dockermanager-tsdproxy-data"
	tsdproxyConfVol  = "dockermanager-tsdproxy-config"
	defaultDashPort  = 8090
)

// Manager handles TSDProxy lifecycle and configuration
type Manager struct {
	docker *dockerclient.Client
}

// NewManager creates a new TSDProxy manager
func NewManager(d *dockerclient.Client) *Manager {
	return &Manager{docker: d}
}

// GetStatus returns the current status of TSDProxy
func (m *Manager) GetStatus(ctx context.Context) (*dmtypes.TSDProxyStatus, error) {
	// Look for our managed container
	containers, err := m.docker.ListContainers(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == tsdproxyName {
				port := 0
				for _, p := range c.Ports {
					if p.PrivatePort == defaultDashPort {
						port = int(p.PublicPort)
					}
				}
				return &dmtypes.TSDProxyStatus{
					Running:     c.State == "running",
					ContainerID: c.ID[:12],
					State:       string(c.State),
					Image:       c.Image,
					Port:        port,
				}, nil
			}
		}
	}

	return &dmtypes.TSDProxyStatus{Running: false}, nil
}

// Deploy creates and starts the TSDProxy container
func (m *Manager) Deploy(ctx context.Context, cfg dmtypes.TSDProxyConfig) error {
	// Check if already exists
	exists, err := m.containerExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("TSDProxy container already exists, remove it first or restart")
	}

	// Ensure image is pulled
	if err := m.pullImage(ctx); err != nil {
		return fmt.Errorf("failed to pull TSDProxy image: %w", err)
	}

	// Ensure volumes exist
	m.ensureVolume(ctx, tsdproxyDataVol)
	m.ensureVolume(ctx, tsdproxyConfVol)

	// Write config file to the config volume
	if err := m.writeConfig(ctx, cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Determine dashboard port
	dashPort := cfg.DashboardPort
	if dashPort == 0 {
		dashPort = defaultDashPort
	}

	// Create container
	portKey := nat.Port(fmt.Sprintf("%d/tcp", dashPort))
	containerConfig := &container.Config{
		Image: tsdproxyImage,
		Labels: map[string]string{
			"managed-by": "dockermanager",
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			portKey: []nat.PortBinding{
				{HostPort: fmt.Sprintf("%d", dashPort)},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
			{
				Type:   mount.TypeVolume,
				Source: tsdproxyDataVol,
				Target: "/data",
			},
			{
				Type:   mount.TypeVolume,
				Source: tsdproxyConfVol,
				Target: "/config",
			},
		},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
	}

	resp, err := m.docker.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, tsdproxyName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := m.docker.StartContainer(ctx, resp.ID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

// Stop stops the TSDProxy container
func (m *Manager) Stop(ctx context.Context) error {
	id, err := m.findContainerID(ctx)
	if err != nil {
		return err
	}
	return m.docker.StopContainer(ctx, id)
}

// Start starts the TSDProxy container
func (m *Manager) Start(ctx context.Context) error {
	id, err := m.findContainerID(ctx)
	if err != nil {
		return err
	}
	return m.docker.StartContainer(ctx, id)
}

// Restart restarts the TSDProxy container
func (m *Manager) Restart(ctx context.Context) error {
	id, err := m.findContainerID(ctx)
	if err != nil {
		return err
	}
	return m.docker.RestartContainer(ctx, id)
}

// Remove removes the TSDProxy container
func (m *Manager) Remove(ctx context.Context) error {
	id, err := m.findContainerID(ctx)
	if err != nil {
		return err
	}
	return m.docker.RemoveContainer(ctx, id, true)
}

// ListServices lists all containers with TSDProxy labels
func (m *Manager) ListServices(ctx context.Context) ([]dmtypes.TSDProxyService, error) {
	containers, err := m.docker.ListContainers(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var services []dmtypes.TSDProxyService
	for _, c := range containers {
		labels := c.Labels
		if labels[tsdproxyLabel] == "true" {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}

			hostname := labels["tsdproxy.name"]
			if hostname == "" {
				hostname = name
			}

			funnel := labels["tsdproxy.funnel"] == "true"

			proxyPort := 0
			if p, ok := labels["tsdproxy.proxyport"]; ok {
				fmt.Sscanf(p, "%d", &proxyPort)
			}

			services = append(services, dmtypes.TSDProxyService{
				ContainerID:   c.ID[:12],
				ContainerName: name,
				Image:         c.Image,
				Hostname:      hostname,
				Enabled:       true,
				Funnel:        funnel,
				ProxyPort:     proxyPort,
				State:         string(c.State),
			})
		}
	}

	return services, nil
}

// EnableForContainer adds TSDProxy labels to a container
func (m *Manager) EnableForContainer(ctx context.Context, containerID string, hostname string, funnel bool) error {
	inspected, err := m.docker.GetContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	labels := inspected.Config.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[tsdproxyLabel] = "true"
	if hostname != "" {
		labels["tsdproxy.name"] = hostname
	} else {
		name := strings.TrimPrefix(inspected.Name, "/")
		labels["tsdproxy.name"] = name
	}
	if funnel {
		labels["tsdproxy.funnel"] = "true"
	}

	// Docker doesn't support updating labels on running containers
	// We need to note this limitation
	return fmt.Errorf("label updates require container recreation — use the compose approach or recreate with labels")
}

// GetConfig returns the current TSDProxy config (placeholder - reads from volume)
func (m *Manager) GetConfig(ctx context.Context) (*dmtypes.TSDProxyConfig, error) {
	// Default config
	return &dmtypes.TSDProxyConfig{
		Tags:          "tag:tsdproxy",
		Hostname:      "tsdproxy",
		AutoApprove:   true,
		DashboardPort: defaultDashPort,
	}, nil
}

// --- internal helpers ---

func (m *Manager) containerExists(ctx context.Context) (bool, error) {
	id, err := m.findContainerID(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return id != "", nil
}

func (m *Manager) findContainerID(ctx context.Context) (string, error) {
	containers, err := m.docker.ListContainers(ctx, true)
	if err != nil {
		return "", err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == tsdproxyName {
				return c.ID, nil
			}
		}
	}

	return "", fmt.Errorf("TSDProxy container not found")
}

func (m *Manager) pullImage(ctx context.Context) error {
	fmt.Printf("Pulling TSDProxy image: %s...\n", tsdproxyImage)
	reader, err := m.docker.ImagePull(ctx, tsdproxyImage, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

func (m *Manager) ensureVolume(ctx context.Context, name string) {
	// Try to create volume, ignore if exists
	// Docker SDK doesn't have a direct "ensure volume" — we let Docker handle it
	_ = name
}

func (m *Manager) writeConfig(ctx context.Context, cfg dmtypes.TSDProxyConfig) error {
	// Generate YAML config
	tags := cfg.Tags
	if tags == "" {
		tags = "tag:tsdproxy"
	}
	hostname := cfg.Hostname
	if hostname == "" {
		hostname = "tsdproxy"
	}

	dashPort := cfg.DashboardPort
	if dashPort == 0 {
		dashPort = defaultDashPort
	}

	autoApprove := "true"
	if !cfg.AutoApprove {
		autoApprove = "false"
	}

	configYAML := fmt.Sprintf(`defaultProxyProvider: default

docker:
  local:
    host: unix:///var/run/docker.sock
    targetHostname: host.docker.internal
    defaultProxyProvider: default

tailscale:
  providers:
    default:
      clientId: "%s"
      clientSecret: "%s"
      tags: "%s"
      services: true
      hostname: "%s"
      autoApproveDevices: %s
      preventDuplicates: true
  dataDir: /data/

http:
  hostname: 0.0.0.0
  port: %d

dashboard:
  adminAllowLocalhost: true

log:
  level: info
  json: false
  proxyAccessLog: true
`, cfg.ClientID, cfg.ClientSecret, tags, hostname, autoApprove, dashPort)

	// Write to a temp file that can be mounted
	tmpPath := "/tmp/tsdproxy-config.yaml"
	if err := os.WriteFile(tmpPath, []byte(configYAML), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
