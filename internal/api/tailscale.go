package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	dockerclient "docker-manager/internal/docker"
	dmtypes "docker-manager/internal/types"
	"docker-manager/internal/tsdproxy"
)

// TSDProxyStatus handles GET /api/tailscale/status
func TSDProxyStatus(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		status, err := mgr.GetStatus(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status})
	}
}

// TSDProxyDeploy handles POST /api/tailscale/deploy
func TSDProxyDeploy(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cfg dmtypes.TSDProxyConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mgr := tsdproxy.NewManager(docker)
		if err := mgr.Deploy(c.Request.Context(), cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy deployed successfully"})
	}
}

// TSDProxyStop handles POST /api/tailscale/stop
func TSDProxyStop(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		if err := mgr.Stop(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy stopped"})
	}
}

// TSDProxyStart handles POST /api/tailscale/start
func TSDProxyStart(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		if err := mgr.Start(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy started"})
	}
}

// TSDProxyRestart handles POST /api/tailscale/restart
func TSDProxyRestart(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		if err := mgr.Restart(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy restarted"})
	}
}

// TSDProxyRemove handles DELETE /api/tailscale/remove
func TSDProxyRemove(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		if err := mgr.Remove(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy removed"})
	}
}

// TSDProxyServices handles GET /api/tailscale/services
func TSDProxyServices(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)
		services, err := mgr.ListServices(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"services": services})
	}
}

// TSDProxyEnableContainer handles POST /api/tailscale/enable/:containerID
func TSDProxyEnableContainer(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := c.Param("containerID")
		var req struct {
			Hostname string `json:"hostname"`
			Funnel   bool   `json:"funnel"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mgr := tsdproxy.NewManager(docker)
		if err := mgr.EnableForContainer(c.Request.Context(), containerID, req.Hostname, req.Funnel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "TSDProxy labels noted — container recreation required"})
	}
}

// TSDProxyConfig handles GET/POST /api/tailscale/config
func TSDProxyConfig(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := tsdproxy.NewManager(docker)

		if c.Request.Method == "GET" {
			cfg, err := mgr.GetConfig(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"config": cfg})
			return
		}

		// POST
		var cfg dmtypes.TSDProxyConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Save config and restart if running
		status, _ := mgr.GetStatus(c.Request.Context())
		if status.Running {
			if err := mgr.Restart(c.Request.Context()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
	}
}
