package docker

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"

	dmtypes "docker-manager/internal/types"
)

// DeployCompose parses a docker-compose YAML and creates/starts containers for each service
func (c *Client) DeployCompose(ctx context.Context, cfg dmtypes.ComposeConfig) ([]string, error) {
	var composeFile dmtypes.ComposeFile
	if err := yaml.Unmarshal([]byte(cfg.Content), &composeFile); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	if len(composeFile.Services) == 0 {
		return nil, fmt.Errorf("no services defined in compose file")
	}

	projectName := cfg.ProjectName
	if projectName == "" {
		projectName = "compose"
	}

	var createdContainers []string

	// Create networks first
	for netName := range composeFile.Networks {
		_, err := c.CreateNetwork(ctx, dmtypes.NetworkConfig{
			Name:   fmt.Sprintf("%s_%s", projectName, netName),
			Driver: "bridge",
		})
		if err != nil {
			fmt.Printf("Warning: could not create network %s: %v\n", netName, err)
		}
	}

	// Create and start services
	for serviceName, svc := range composeFile.Services {
		containerName := fmt.Sprintf("%s_%s_1", projectName, serviceName)

		// Parse ports
		var portMappings []dmtypes.PortMapping
		for _, p := range svc.Ports {
			hostPort := 0
			containerPort := 0
			protocol := "tcp"
			if _, err := fmt.Sscanf(p, "%d:%d/%s", &hostPort, &containerPort, &protocol); err != nil {
				if _, err := fmt.Sscanf(p, "%d:%d", &hostPort, &containerPort); err != nil {
					fmt.Sscanf(p, "%d", &containerPort)
				}
			}
			portMappings = append(portMappings, dmtypes.PortMapping{
				HostPort:      hostPort,
				ContainerPort: containerPort,
				Protocol:      protocol,
			})
		}

		// Parse environment
		env := make(map[string]string)
		for _, e := range svc.Environment {
			key, val := parseEnvVar(e)
			if key != "" {
				env[key] = val
			}
		}

		// Parse volumes
		var volumes []dmtypes.VolumeMapping
		for _, v := range svc.Volumes {
			hostPath := ""
			containerPath := ""
			readOnly := false
			if _, err := fmt.Sscanf(v, "%s:%s:ro", &hostPath, &containerPath); err == nil {
				readOnly = true
			} else if _, err := fmt.Sscanf(v, "%s:%s", &hostPath, &containerPath); err != nil {
				containerPath = v
			}
			volumes = append(volumes, dmtypes.VolumeMapping{
				HostPath:      hostPath,
				ContainerPath: containerPath,
				ReadOnly:      readOnly,
			})
		}

		// Determine network name
		networkName := ""
		if len(svc.Networks) > 0 {
			networkName = fmt.Sprintf("%s_%s", projectName, svc.Networks[0])
		}

		networkMode := ""
		if len(svc.Networks) > 0 && len(composeFile.Networks) == 0 {
			networkMode = "bridge"
		}

		// Create the container
		resp, err := c.CreateContainer(ctx, dmtypes.ContainerConfig{
			Name:          containerName,
			Image:         svc.Image,
			Command:       splitCommand(svc.Command),
			Env:           env,
			Ports:         portMappings,
			Volumes:       volumes,
			NetworkMode:   networkMode,
			NetworkName:   networkName,
			RestartPolicy: svc.Restart,
			Labels:        svc.Labels,
		})
		if err != nil {
			fmt.Printf("Warning: could not create service %s: %v\n", serviceName, err)
			continue
		}

		// Start the container
		if err := c.StartContainer(ctx, resp.ID); err != nil {
			fmt.Printf("Warning: could not start service %s: %v\n", serviceName, err)
		}

		createdContainers = append(createdContainers, containerName)
	}

	if len(createdContainers) == 0 {
		return nil, fmt.Errorf("no containers were created from compose file")
	}

	return createdContainers, nil
}

// ParseComposeFile parses a docker-compose YAML and returns service info
func ParseComposeFile(content string) (*dmtypes.ComposeFile, error) {
	var composeFile dmtypes.ComposeFile
	if err := yaml.Unmarshal([]byte(content), &composeFile); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}
	return &composeFile, nil
}

func parseEnvVar(e string) (string, string) {
	for i := 0; i < len(e); i++ {
		if e[i] == '=' {
			return e[:i], e[i+1:]
		}
	}
	return e, ""
}

func splitCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
	var result []string
	current := ""
	inQuote := false
	for _, ch := range cmd {
		if ch == '"' || ch == '\'' {
			inQuote = !inQuote
			continue
		}
		if ch == ' ' && !inQuote {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			continue
		}
		current += string(ch)
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
