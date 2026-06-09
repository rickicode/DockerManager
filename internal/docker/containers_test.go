package docker

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	dmtypes "docker-manager/internal/types"
)

// ============================================
// Tests for toLower
// ============================================

func TestToLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"already lowercase", "hello", "hello"},
		{"uppercase", "HELLO", "hello"},
		{"mixed case", "HeLLo WoRLd", "hello world"},
		{"with numbers", "Test123", "test123"},
		{"with special chars", "Hello-World!", "hello-world!"},
		{"unicode not affected", "Café", "café"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLower(tt.input)
			if result != tt.expected {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================
// Tests for containsAny
// ============================================

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substrs  []string
		expected bool
	}{
		{"empty string", "", []string{"test"}, false},
		{"empty substrs", "hello", []string{}, false},
		{"exact match", "hello world", []string{"hello"}, true},
		{"case insensitive", "Hello World", []string{"hello"}, true},
		{"mixed case substr", "Hello World", []string{"WORLD"}, true},
		{"multiple substrs - one matches", "The quick brown fox", []string{"cat", "dog", "fox"}, true},
		{"multiple substrs - none match", "The quick brown fox", []string{"cat", "dog", "bird"}, false},
		{"substring in middle", "succeeded", []string{"ceed"}, true},
		{"Connected to pattern", "Connected to 192.168.1.1:80", []string{"Connected to"}, true},
		{"open pattern", "80/tcp open", []string{"open"}, true},
		{"nc output format succeeded", "Connection to localhost port 80 succeeded!", []string{"succeeded"}, true},
		{"nc output format failed", "nc: connection refused", []string{"succeeded", "open", "Connected to"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrs...)
			if result != tt.expected {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
			}
		})
	}
}

// ============================================
// Tests for ToContainerInfo
// ============================================

func TestToContainerInfo(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name     string
		summary  container.Summary
		expected dmtypes.ContainerInfo
	}{
		{
			name: "running container with ports",
			summary: container.Summary{
				ID:      "abc123def456ghi789jkl",
				Names:   []string{"/my-nginx"},
				Image:   "nginx:latest",
				Created: now,
				Ports: []container.Port{
					{PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
				},
				State:  container.StateRunning,
				Status: "Up 2 hours",
				Labels: map[string]string{"app": "web"},
			},
			expected: dmtypes.ContainerInfo{
				ID:      "abc123def456",
				Name:    "my-nginx",
				Image:   "nginx:latest",
				State:   "running",
				Status:  "Up 2 hours",
				Created: time.Unix(now, 0).Format(time.RFC3339),
				Ports: []dmtypes.PortMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
				Labels: map[string]string{"app": "web"},
			},
		},
		{
			name: "stopped container no ports",
			summary: container.Summary{
				ID:      "xyz987abc654",
				Names:   []string{"/old-container"},
				Image:   "alpine:latest",
				Created: now - 86400, // 1 day ago
				Ports:   []container.Port{},
				State:   container.StateExited,
				Status:  "Exited (0) 2 days ago",
			},
			expected: dmtypes.ContainerInfo{
				ID:      "xyz987abc654",
				Name:    "old-container",
				Image:   "alpine:latest",
				State:   "exited",
				Status:  "Exited (0) 2 days ago",
				Created: time.Unix(now-86400, 0).Format(time.RFC3339),
				Ports:   []dmtypes.PortMapping{},
				Labels:  nil,
			},
		},
		{
			name: "container with no name (empty names)",
			summary: container.Summary{
				ID:      "short",
				Names:   []string{},
				Image:   "busybox:latest",
				Created: now,
				Ports:   nil,
				State:   container.StateCreated,
				Status:  "Created",
			},
			expected: dmtypes.ContainerInfo{
				ID:      "short",
				Name:    "",
				Image:   "busybox:latest",
				State:   "created",
				Status:  "Created",
				Created: time.Unix(now, 0).Format(time.RFC3339),
				Ports:   []dmtypes.PortMapping{},
			},
		},
		{
			name: "multiple ports with UDP",
			summary: container.Summary{
				ID:      "longenoughid12345678",
				Names:   []string{"/multi-port"},
				Image:   "custom-app:1.0",
				Created: now,
				Ports: []container.Port{
					{PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
					{PrivatePort: 443, PublicPort: 8443, Type: "tcp"},
					{PrivatePort: 53, PublicPort: 5353, Type: "udp"},
				},
				State:  container.StateRunning,
				Status: "Up 5 hours",
			},
			expected: dmtypes.ContainerInfo{
				ID:      "longenoughid",
				Name:    "multi-port",
				Image:   "custom-app:1.0",
				State:   "running",
				Status:  "Up 5 hours",
				Created: time.Unix(now, 0).Format(time.RFC3339),
				Ports: []dmtypes.PortMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
					{HostPort: 8443, ContainerPort: 443, Protocol: "tcp"},
					{HostPort: 5353, ContainerPort: 53, Protocol: "udp"},
				},
			},
		},
		{
			name: "short ID truncation safeguard",
			summary: container.Summary{
				ID:      "abc",
				Names:   []string{"/short-id"},
				Image:   "test:latest",
				Created: now,
				Ports:   nil,
				State:   container.StateRunning,
				Status:  "Up 1 minute",
			},
			expected: dmtypes.ContainerInfo{
				ID:      "abc",
				Name:    "short-id",
				Image:   "test:latest",
				State:   "running",
				Status:  "Up 1 minute",
				Created: time.Unix(now, 0).Format(time.RFC3339),
				Ports:   []dmtypes.PortMapping{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToContainerInfo(tt.summary)

			// Check fields
			if result.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", result.Name, tt.expected.Name)
			}
			if result.Image != tt.expected.Image {
				t.Errorf("Image = %q, want %q", result.Image, tt.expected.Image)
			}
			if result.State != tt.expected.State {
				t.Errorf("State = %q, want %q", result.State, tt.expected.State)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status = %q, want %q", result.Status, tt.expected.Status)
			}
			if result.Created != tt.expected.Created {
				t.Errorf("Created = %q, want %q", result.Created, tt.expected.Created)
			}

			// Compare ports
			if len(result.Ports) != len(tt.expected.Ports) {
				t.Errorf("Ports count = %d, want %d", len(result.Ports), len(tt.expected.Ports))
			} else {
				for i, p := range result.Ports {
					if p != tt.expected.Ports[i] {
						t.Errorf("Port[%d] = %+v, want %+v", i, p, tt.expected.Ports[i])
					}
				}
			}
		})
	}
}

// ============================================
// Tests for ToContainerInfoDetailed
// ============================================

func TestToContainerInfoDetailed(t *testing.T) {
	hostPort := "8080"
	tests := []struct {
		name     string
		inspect  container.InspectResponse
		expected dmtypes.ContainerInfo
	}{
		{
			name: "full running container details",
			inspect: container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					ID:      "abc123def456ghi789jklmno",
					Created: "2024-01-15T10:00:00Z",
					Name:    "/my-web-app",
					Image:   "nginx:1.25",
					State: &container.State{
						Status:  container.StateRunning,
						Running: true,
					},
					HostConfig: &container.HostConfig{
						NetworkMode: container.NetworkMode("bridge"),
						RestartPolicy: container.RestartPolicy{
							Name: container.RestartPolicyAlways,
						},
					},
				},
				Config: &container.Config{
					Image: "nginx:1.25",
					Env:   []string{"NGINX_HOST=example.com", "NGINX_PORT=80"},
					Labels: map[string]string{
						"app": "web",
						"env": "prod",
					},
				},
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							nat.Port("80/tcp"): []nat.PortBinding{
								{HostPort: hostPort, HostIP: "0.0.0.0"},
							},
						},
					},
					DefaultNetworkSettings: container.DefaultNetworkSettings{
						IPAddress: "172.17.0.2",
					},
					Networks: map[string]*network.EndpointSettings{
						"bridge": {IPAddress: "172.17.0.2"},
					},
				},
				Mounts: []container.MountPoint{
					{
						Source:      "/host/html",
						Destination: "/usr/share/nginx/html",
						RW:          true,
						Mode:        "rprivate",
						Type:        mount.TypeBind,
					},
					{
						Source:      "/host/config",
						Destination: "/etc/nginx/conf.d",
						RW:          false,
						Mode:        "ro,rprivate",
						Type:        mount.TypeBind,
					},
				},
			},
			expected: dmtypes.ContainerInfo{
				ID:      "abc123def456",
				Name:    "my-web-app",
				Image:   "nginx:1.25",
				State:   "running",
				Status:  "running",
				Created: "2024-01-15T10:00:00Z",
				Ports: []dmtypes.PortMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
				IPAddress:   "172.17.0.2",
				NetworkMode: "bridge",
				Mounts: []dmtypes.VolumeMapping{
					{HostPath: "/host/html", ContainerPath: "/usr/share/nginx/html", ReadOnly: false},
					{HostPath: "/host/config", ContainerPath: "/etc/nginx/conf.d", ReadOnly: true},
				},
				Env: []string{"NGINX_HOST=example.com", "NGINX_PORT=80"},
				Labels: map[string]string{
					"app": "web",
					"env": "prod",
				},
				RestartPolicy: "always",
			},
		},
		{
			name: "stopped container with minimal info",
			inspect: container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					ID:      "xyz789abc456",
					Created: "2023-12-01T08:00:00Z",
					Name:    "/stopped-app",
					Image:   "alpine:3.19",
					State: &container.State{
						Status:  container.StateExited,
						Running: false,
					},
					HostConfig: &container.HostConfig{},
				},
				Config: &container.Config{
					Image: "alpine:3.19",
				},
				NetworkSettings: &container.NetworkSettings{},
				Mounts:          []container.MountPoint{},
			},
			expected: dmtypes.ContainerInfo{
				ID:      "xyz789abc456",
				Name:    "stopped-app",
				Image:   "alpine:3.19",
				State:   "exited",
				Status:  "exited",
				Created: "2023-12-01T08:00:00Z",
				Ports:   []dmtypes.PortMapping{},
				Mounts:  []dmtypes.VolumeMapping{},
				Env:     []string{},
			},
		},
		{
			name: "container with host networking",
			inspect: container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					ID:      "net123host456",
					Created: "2024-02-20T12:00:00Z",
					Name:    "/host-net-app",
					Image:   "myapp:latest",
					State: &container.State{
						Status: container.StateRunning,
					},
					HostConfig: &container.HostConfig{
						NetworkMode: container.NetworkMode("host"),
					},
				},
				Config: &container.Config{
					Image: "myapp:latest",
				},
				NetworkSettings: &container.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{},
				},
				Mounts: nil,
			},
			expected: dmtypes.ContainerInfo{
				ID:            "net123host45",
				Name:          "host-net-app",
				Image:         "myapp:latest",
				State:         "running",
				Status:        "running",
				Created:       "2024-02-20T12:00:00Z",
				Ports:         []dmtypes.PortMapping{},
				NetworkMode:   "host",
				Mounts:        []dmtypes.VolumeMapping{},
				Env:           []string{},
			},
		},
		{
			name: "nil Config and nil State",
			inspect: container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					ID:         "nilcfg1234567",
					Created:    "2024-03-01T00:00:00Z",
					Name:       "/no-config",
					Image:      "unknown:latest",
					State:      nil,
					HostConfig: nil,
				},
				Config:          nil,
				NetworkSettings: nil,
				Mounts:          nil,
			},
			expected: dmtypes.ContainerInfo{
				ID:      "nilcfg123456",
				Name:    "no-config",
				Image:   "unknown:latest",
				State:   "",
				Status:  "",
				Created: "2024-03-01T00:00:00Z",
				Ports:   []dmtypes.PortMapping{},
				Mounts:  []dmtypes.VolumeMapping{},
				Env:     []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToContainerInfoDetailed(tt.inspect)

			if result.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", result.Name, tt.expected.Name)
			}
			if result.Image != tt.expected.Image {
				t.Errorf("Image = %q, want %q", result.Image, tt.expected.Image)
			}
			if result.State != tt.expected.State {
				t.Errorf("State = %q, want %q", result.State, tt.expected.State)
			}
			if result.IPAddress != tt.expected.IPAddress {
				t.Errorf("IPAddress = %q, want %q", result.IPAddress, tt.expected.IPAddress)
			}
			if result.NetworkMode != tt.expected.NetworkMode {
				t.Errorf("NetworkMode = %q, want %q", result.NetworkMode, tt.expected.NetworkMode)
			}

			// Compare ports
			if len(result.Ports) != len(tt.expected.Ports) {
				t.Errorf("Ports count = %d, want %d", len(result.Ports), len(tt.expected.Ports))
			} else {
				for i, p := range result.Ports {
					if p != tt.expected.Ports[i] {
						t.Errorf("Port[%d] = %+v, want %+v", i, p, tt.expected.Ports[i])
					}
				}
			}

			// Compare mounts
			if len(result.Mounts) != len(tt.expected.Mounts) {
				t.Errorf("Mounts count = %d, want %d", len(result.Mounts), len(tt.expected.Mounts))
			} else {
				for i, m := range result.Mounts {
					if m != tt.expected.Mounts[i] {
						t.Errorf("Mount[%d] = %+v, want %+v", i, m, tt.expected.Mounts[i])
					}
				}
			}

			// Check labels
			if len(result.Labels) != len(tt.expected.Labels) {
				t.Errorf("Labels count = %d, want %d", len(result.Labels), len(tt.expected.Labels))
			}
		})
	}
}
