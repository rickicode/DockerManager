package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
	"docker-manager/internal/types"
)

// ListNetworks handles GET /api/networks
func ListNetworks(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		networks, err := docker.ListNetworks(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		infos := make([]types.NetworkInfo, 0, len(networks))
		for _, n := range networks {
			infos = append(infos, dockerclient.ToNetworkInfo(n))
		}

		c.JSON(http.StatusOK, gin.H{"networks": infos})
	}
}

// GetNetwork handles GET /api/networks/:id
func GetNetwork(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		network, err := docker.GetNetwork(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"network": network})
	}
}

// CreateNetworkHandler handles POST /api/networks
func CreateNetworkHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var config types.NetworkConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := docker.CreateNetwork(c.Request.Context(), config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":      resp.ID,
			"warning": resp.Warning,
		})
	}
}

// RemoveNetworkHandler handles DELETE /api/networks/:id
func RemoveNetworkHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := docker.RemoveNetwork(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Network removed successfully"})
	}
}
