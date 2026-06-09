package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
)

// ListImages handles GET /api/images
func ListImages(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		images, err := docker.ListImages(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		infos := make([]interface{}, 0, len(images))
		for _, img := range images {
			infos = append(infos, dockerclient.ToImageInfo(img))
		}

		c.JSON(http.StatusOK, gin.H{"images": infos})
	}
}

// PullImageHandler handles POST /api/images/pull
func PullImageHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Image string `json:"image" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := docker.PullImage(c.Request.Context(), req.Image); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Image pulled successfully"})
	}
}

// RemoveImageHandler handles DELETE /api/images/:id
func RemoveImageHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		force := c.DefaultQuery("force", "false") == "true"

		if err := docker.RemoveImage(c.Request.Context(), id, force); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Image removed successfully"})
	}
}
