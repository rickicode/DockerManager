package api

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	dockerclient "docker-manager/internal/docker"
	"docker-manager/internal/types"
)

// GetSystemInfo handles GET /api/system/info
func GetSystemInfo(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := docker.Info(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		sysInfo := types.SystemInfo{
			Version:       info.ServerVersion,
			Containers:    info.Containers,
			Running:       info.ContainersRunning,
			Paused:        info.ContainersPaused,
			Stopped:       info.ContainersStopped,
			Images:        info.Images,
			OS:            info.OperatingSystem,
			Architecture:  info.Architecture,
			KernelVersion: info.KernelVersion,
			OSType:        info.OSType,
			ServerVersion: info.ServerVersion,
		}

		c.JSON(http.StatusOK, gin.H{"info": sysInfo})
	}
}

// CheckPortHandler handles POST /api/port-check
func CheckPortHandler(docker *dockerclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.PortCheckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// First try a direct TCP connection
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		address := net.JoinHostPort(req.Host, strconv.Itoa(req.Port))
		conn, err := net.DialTimeout("tcp", address, 3*time.Second)

		result := types.PortCheckResult{
			Host: req.Host,
			Port: req.Port,
		}

		if err == nil {
			conn.Close()
			result.Open = true
			c.JSON(http.StatusOK, gin.H{"result": result})
			return
		}

		// If direct connection fails, try using Docker container
		open, service, err := docker.CheckPort(ctx, req.Host, req.Port)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"result": result,
				"note":   "Direct check failed, Docker-based check also failed",
			})
			return
		}

		result.Open = open
		result.Service = service

		c.JSON(http.StatusOK, gin.H{"result": result})
	}
}
