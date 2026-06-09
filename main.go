package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"docker-manager/internal/api"
	dockerclient "docker-manager/internal/docker"
)

//go:embed web/static
//go:embed web/static/**/*
var staticFiles embed.FS

func main() {
	host := flag.String("host", "0.0.0.0", "Host address to bind to")
	port := flag.Int("port", 8080, "Port to listen on")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println("DockerManager v1.0.0")
		fmt.Println("A lightweight Docker management tool")
		os.Exit(0)
	}

	// Connect to Docker
	fmt.Println("Connecting to Docker daemon...")
	docker := dockerclient.MustNewClient()
	fmt.Println("Connected to Docker daemon successfully")

	// Setup API router
	router := api.SetupRouter(docker)

	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "web/static")
	if err != nil {
		log.Fatalf("Failed to load static files: %v", err)
	}

	// Handle static file serving for all non-API routes
	router.NoRoute(func(c *gin.Context) {
		// Try to serve static file
		path := c.Request.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Remove leading slash
		cleanPath := strings.TrimPrefix(path, "/")

		data, err := fs.ReadFile(staticFS, cleanPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// Determine content type
		contentType := getContentType(cleanPath)
		c.Data(http.StatusOK, contentType, data)
	})

	addr := net.JoinHostPort(*host, fmt.Sprintf("%d", *port))
	fmt.Printf("\n  🐳 DockerManager is running!\n")
	fmt.Printf("  Web UI: http://%s\n", addr)
	fmt.Printf("  API:    http://%s/api\n\n", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "text/plain; charset=utf-8"
	}
}
