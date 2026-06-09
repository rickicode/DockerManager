package types

// ContainerConfig represents the configuration for creating a container
type ContainerConfig struct {
	Name          string            `json:"name"`
	Image         string            `json:"image" binding:"required"`
	Command       []string          `json:"command"`
	Env           map[string]string `json:"env"`
	Ports         []PortMapping     `json:"ports"`
	Volumes       []VolumeMapping   `json:"volumes"`
	NetworkMode   string            `json:"networkMode"`
	NetworkName   string            `json:"networkName"`
	RestartPolicy string            `json:"restartPolicy"`
	Labels        map[string]string `json:"labels"`
}

// PortMapping represents a port mapping between host and container
type PortMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

// VolumeMapping represents a volume/bind mount mapping
type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

// ContainerInfo represents container information for API responses
type ContainerInfo struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Status        string            `json:"status"`
	State         string            `json:"state"`
	Ports         []PortMapping     `json:"ports"`
	Created       string            `json:"created"`
	IPAddress     string            `json:"ipAddress"`
	NetworkMode   string            `json:"networkMode"`
	Mounts        []VolumeMapping   `json:"mounts"`
	Env           []string          `json:"env"`
	Labels        map[string]string `json:"labels"`
	RestartPolicy string            `json:"restartPolicy"`
}

// ImageInfo represents image information for API responses
type ImageInfo struct {
	ID       string            `json:"id"`
	RepoTags []string          `json:"repoTags"`
	Size     int64             `json:"size"`
	Created  string            `json:"created"`
	Labels   map[string]string `json:"labels"`
}

// NetworkConfig represents the configuration for creating a network
type NetworkConfig struct {
	Name    string            `json:"name" binding:"required"`
	Driver  string            `json:"driver"`
	Subnet  string            `json:"subnet"`
	Gateway string            `json:"gateway"`
	Labels  map[string]string `json:"labels"`
}

// NetworkInfo represents network information for API responses
type NetworkInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Subnet     string            `json:"subnet"`
	Gateway    string            `json:"gateway"`
	Containers int              `json:"containers"`
	Labels     map[string]string `json:"labels"`
}

// ComposeConfig represents a docker-compose configuration
type ComposeConfig struct {
	Content     string `json:"content" binding:"required"`
	ProjectName string `json:"projectName"`
}

// PortCheckRequest represents a port check request
type PortCheckRequest struct {
	Host string `json:"host" binding:"required"`
	Port int    `json:"port" binding:"required"`
}

// PortCheckResult represents the result of a port check
type PortCheckResult struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Open    bool   `json:"open"`
	Service string `json:"service,omitempty"`
}

// SystemInfo represents Docker system information
type SystemInfo struct {
	Version       string `json:"version"`
	APIVersion    string `json:"apiVersion"`
	Containers    int    `json:"containers"`
	Running       int    `json:"running"`
	Paused        int    `json:"paused"`
	Stopped       int    `json:"stopped"`
	Images        int    `json:"images"`
	OS            string `json:"os"`
	Architecture  string `json:"architecture"`
	KernelVersion string `json:"kernelVersion"`
	OSType        string `json:"osType"`
	ServerVersion string `json:"serverVersion"`
}

// TSDProxyStatus represents the status of the TSDProxy container
type TSDProxyStatus struct {
	Running   bool   `json:"running"`
	ContainerID string `json:"containerId,omitempty"`
	State     string `json:"state,omitempty"`
	Image     string `json:"image,omitempty"`
	Port      int    `json:"port,omitempty"`
}

// TSDProxyConfig represents TSDProxy configuration
type TSDProxyConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Tags         string `json:"tags"`
	Hostname     string `json:"hostname"`
	AutoApprove  bool   `json:"autoApprove"`
	DashboardPort int    `json:"dashboardPort"`
}

// TSDProxyService represents a container proxied by TSDProxy
type TSDProxyService struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Image         string `json:"image"`
	Hostname      string `json:"hostname"`
	Enabled       bool   `json:"enabled"`
	Funnel        bool   `json:"funnel"`
	ProxyPort     int    `json:"proxyPort"`
	State         string `json:"state"`
}

// ComposeService represents a service in a compose file
type ComposeService struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Ports       []string          `json:"ports"`
	Environment map[string]string `json:"environment"`
	Volumes     []string          `json:"volumes"`
	Networks    []string          `json:"networks"`
}

// ComposeFile represents a parsed docker-compose file
type ComposeFile struct {
	Version  string                        `yaml:"version,omitempty"`
	Services map[string]ComposeFileService `yaml:"services"`
	Networks map[string]interface{}        `yaml:"networks,omitempty"`
	Volumes  map[string]interface{}        `yaml:"volumes,omitempty"`
}

// ComposeFileService represents a service definition in a compose file
type ComposeFileService struct {
	Image       string            `yaml:"image"`
	Command     string            `yaml:"command,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment []string          `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
}
