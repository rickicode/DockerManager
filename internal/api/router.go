package api

import (
	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
)

// SetupRouter configures all API routes
func SetupRouter(docker *dockerclient.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		// System
		api.GET("/system/info", GetSystemInfo(docker))
		api.POST("/port-check", CheckPortHandler(docker))

		// Containers
		api.GET("/containers", ListContainers(docker))
		api.GET("/containers/:id", GetContainer(docker))
		api.GET("/containers/:id/logs", GetContainerLogs(docker))
		api.POST("/containers", CreateContainerHandler(docker))
		api.POST("/containers/:id/start", StartContainer(docker))
		api.POST("/containers/:id/stop", StopContainer(docker))
		api.POST("/containers/:id/restart", RestartContainer(docker))
		api.DELETE("/containers/:id", RemoveContainer(docker))

		// Images
		api.GET("/images", ListImages(docker))
		api.POST("/images/pull", PullImageHandler(docker))
		api.DELETE("/images/:id", RemoveImageHandler(docker))

		// Networks
		api.GET("/networks", ListNetworks(docker))
		api.GET("/networks/:id", GetNetwork(docker))
		api.POST("/networks", CreateNetworkHandler(docker))
		api.DELETE("/networks/:id", RemoveNetworkHandler(docker))

		// Docker Compose
		api.POST("/compose/deploy", DeployComposeHandler(docker))
		api.POST("/compose/parse", ParseComposeHandler())

		// Tailscale / TSDProxy
		api.GET("/tailscale/status", TSDProxyStatus(docker))
		api.POST("/tailscale/deploy", TSDProxyDeploy(docker))
		api.POST("/tailscale/stop", TSDProxyStop(docker))
		api.POST("/tailscale/start", TSDProxyStart(docker))
		api.POST("/tailscale/restart", TSDProxyRestart(docker))
		api.DELETE("/tailscale/remove", TSDProxyRemove(docker))
		api.GET("/tailscale/services", TSDProxyServices(docker))
		api.POST("/tailscale/enable/:containerID", TSDProxyEnableContainer(docker))
		api.GET("/tailscale/config", TSDProxyConfig(docker))
		api.POST("/tailscale/config", TSDProxyConfig(docker))
	}

	return r
}
