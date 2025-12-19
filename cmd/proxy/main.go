package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yordanos-habtamu/b2b-payments/internal/config"
	"github.com/yordanos-habtamu/b2b-payments/internal/proxy"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get backend servers from environment or use defaults
	backendServers := getBackendServers()

	// Initialize load balancer
	lb, err := proxy.NewLoadBalancer(backendServers, proxy.RoundRobin)
	if err != nil {
		log.Fatalf("Failed to create load balancer: %v", err)
	}

	// Start health checks
	lb.StartHealthChecks()
	defer lb.StopHealthChecks()

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// Health check endpoint for the proxy itself
	e.GET("/health", func(c echo.Context) error {
		stats := lb.GetServerStats()
		return c.JSON(200, map[string]interface{}{
			"status": "healthy",
			"proxy":  true,
			"stats":  stats,
		})
	})

	// Load balancer stats endpoint
	e.GET("/stats", func(c echo.Context) error {
		stats := lb.GetServerStats()
		return c.JSON(200, stats)
	})

	// Proxy all other requests to backend servers
	e.Any("/*", lb.ProxyHandler())

	// Configuration
	port := cfg.Port
	if port == 0 {
		port = 8080 // Default for proxy
	}

	// Start server
	go func() {
		log.Printf("Starting load balancer on port %d", port)
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down proxy server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Proxy server shutdown complete")
}

func getBackendServers() []string {
	// Try to get servers from environment variable
	if servers := os.Getenv("BACKEND_SERVERS"); servers != "" {
		// Parse comma-separated list of servers
		// This is a simple implementation - in production you might want more sophisticated parsing
		return []string{servers}
	}

	// Default backend servers
	return []string{
		"http://localhost:8443", // Default API server
		"http://localhost:8444", // Additional instance for load balancing
		"http://localhost:8445", // Additional instance for load balancing
	}
}