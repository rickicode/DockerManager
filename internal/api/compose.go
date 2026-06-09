package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
	"docker-manager/internal/types"
)

// DeployComposeHandler handles POST /api/compose/deploy
func DeployComposeHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var config types.ComposeConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		containers, err := docker.DeployCompose(c.Request.Context(), config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Compose deployment started",
			"containers": containers,
		})
	}
}

// ParseComposeHandler handles POST /api/compose/parse
func ParseComposeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		composeFile, err := dockerclient.ParseComposeFile(req.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		services := []types.ComposeService{}
		for name, svc := range composeFile.Services {
			env := make(map[string]string)
			for _, e := range svc.Environment {
				key, val := parseEnvVar(e)
				if key != "" {
					env[key] = val
				}
			}

			services = append(services, types.ComposeService{
				Name:        name,
				Image:       svc.Image,
				Ports:       svc.Ports,
				Environment: env,
				Volumes:     svc.Volumes,
				Networks:    svc.Networks,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"version":  composeFile.Version,
			"services": services,
		})
	}
}

func parseEnvVar(e string) (string, string) {
	for i := 0; i < len(e); i++ {
		if e[i] == '=' {
			return e[:i], e[i+1:]
		}
	}
	return e, ""
}
