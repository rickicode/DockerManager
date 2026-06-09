package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
	"docker-manager/internal/types"
)

// ListContainers handles GET /api/containers
func ListContainers(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		all := c.DefaultQuery("all", "false") == "true"

		containers, err := docker.ListContainers(c.Request.Context(), all)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		infos := make([]types.ContainerInfo, 0, len(containers))
		for _, container := range containers {
			infos = append(infos, dockerclient.ToContainerInfo(container))
		}

		c.JSON(http.StatusOK, gin.H{"containers": infos})
	}
}

// GetContainer handles GET /api/containers/:id
func GetContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		container, err := docker.GetContainer(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		info := dockerclient.ToContainerInfoDetailed(*container)
		c.JSON(http.StatusOK, gin.H{"container": info})
	}
}

// GetContainerLogs handles GET /api/containers/:id/logs
func GetContainerLogs(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		tail := c.DefaultQuery("tail", "100")

		logs, err := docker.GetContainerLogs(c.Request.Context(), id, tail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"logs": logs})
	}
}

// CreateContainerHandler handles POST /api/containers
func CreateContainerHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var config types.ContainerConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := docker.CreateContainer(c.Request.Context(), config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       resp.ID,
			"warnings": resp.Warnings,
		})
	}
}

// StartContainer handles POST /api/containers/:id/start
func StartContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := docker.StartContainer(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Container started successfully"})
	}
}

// StopContainer handles POST /api/containers/:id/stop
func StopContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := docker.StopContainer(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Container stopped successfully"})
	}
}

// RestartContainer handles POST /api/containers/:id/restart
func RestartContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := docker.RestartContainer(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Container restarted successfully"})
	}
}

// RemoveContainer handles DELETE /api/containers/:id
func RemoveContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		force := c.DefaultQuery("force", "false") == "true"

		if err := docker.RemoveContainer(c.Request.Context(), id, force); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Container removed successfully"})
	}
}
