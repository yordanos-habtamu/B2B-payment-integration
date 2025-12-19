package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type LoadBalancer struct {
	servers       []*Server
	currentIndex  int
	mu            sync.RWMutex
	healthChecker *HealthChecker
	strategy      LoadBalancingStrategy
}

type Server struct {
	URL          *url.URL
	Healthy      bool
	Weight       int
	Connections  int
	LastHealthCheck time.Time
}

type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	LeastConnections
	WeightedRoundRobin
	IPHash
)

type HealthChecker struct {
	servers  []*Server
	interval time.Duration
	client   *http.Client
	stopCh   chan struct{}
}

func NewLoadBalancer(servers []string, strategy LoadBalancingStrategy) (*LoadBalancer, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("at least one server is required")
	}

	lb := &LoadBalancer{
		servers:  make([]*Server, len(servers)),
		strategy: strategy,
	}

	// Initialize servers
	for i, serverURL := range servers {
		parsedURL, err := url.Parse(serverURL)
		if err != nil {
			return nil, fmt.Errorf("invalid server URL %s: %w", serverURL, err)
		}

		lb.servers[i] = &Server{
			URL:     parsedURL,
			Healthy: true, // Assume healthy initially
			Weight:  1,    // Default weight
		}
	}

	// Initialize health checker
	lb.healthChecker = NewHealthChecker(lb.servers, 30*time.Second)

	return lb, nil
}

func NewHealthChecker(servers []*Server, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		servers:  servers,
		interval: interval,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

func (lb *LoadBalancer) StartHealthChecks() {
	lb.healthChecker.Start()
}

func (lb *LoadBalancer) StopHealthChecks() {
	lb.healthChecker.Stop()
}

func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.checkAllServers()
			case <-hc.stopCh:
				return
			}
		}
	}()
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

func (hc *HealthChecker) checkAllServers() {
	for _, server := range hc.servers {
		go hc.checkServer(server)
	}
}

func (hc *HealthChecker) checkServer(server *Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthURL := *server.URL
	healthURL.Path = "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL.String(), nil)
	if err != nil {
		server.Healthy = false
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		server.Healthy = false
		return
	}
	defer resp.Body.Close()

	server.Healthy = resp.StatusCode == http.StatusOK
	server.LastHealthCheck = time.Now()
}

func (lb *LoadBalancer) GetNextServer() *Server {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	healthyServers := lb.getHealthyServers()
	if len(healthyServers) == 0 {
		return nil
	}

	switch lb.strategy {
	case RoundRobin:
		return lb.roundRobin(healthyServers)
	case LeastConnections:
		return lb.leastConnections(healthyServers)
	case WeightedRoundRobin:
		return lb.weightedRoundRobin(healthyServers)
	case IPHash:
		return lb.ipHash(healthyServers)
	default:
		return lb.roundRobin(healthyServers)
	}
}

func (lb *LoadBalancer) getHealthyServers() []*Server {
	var healthy []*Server
	for _, server := range lb.servers {
		if server.Healthy {
			healthy = append(healthy, server)
		}
	}
	return healthy
}

func (lb *LoadBalancer) roundRobin(servers []*Server) *Server {
	server := servers[lb.currentIndex%len(servers)]
	lb.currentIndex++
	return server
}

func (lb *LoadBalancer) leastConnections(servers []*Server) *Server {
	var selected *Server
	minConnections := int(^uint(0) >> 1) // Max int

	for _, server := range servers {
		if server.Connections < minConnections {
			minConnections = server.Connections
			selected = server
		}
	}

	return selected
}

func (lb *LoadBalancer) weightedRoundRobin(servers []*Server) *Server {
	totalWeight := 0
	for _, server := range servers {
		totalWeight += server.Weight
	}

	if totalWeight == 0 {
		return lb.roundRobin(servers)
	}

	target := lb.currentIndex % totalWeight
	currentWeight := 0

	for _, server := range servers {
		currentWeight += server.Weight
		if target < currentWeight {
			lb.currentIndex++
			return server
		}
	}

	return servers[0]
}

func (lb *LoadBalancer) ipHash(servers []*Server) *Server {
	// This is a simplified IP hash implementation
	// In a real implementation, you'd use the client's IP address
	return servers[lb.currentIndex%len(servers)]
}

func (lb *LoadBalancer) ProxyHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		server := lb.GetNextServer()
		if server == nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no healthy servers available")
		}

		// Increment connection count
		server.Connections++
		defer func() {
			server.Connections--
		}()

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(server.URL)

		// Set up error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.Logger().Error("Proxy error", "error", err, "server", server.URL.String())
			c.JSON(http.StatusBadGateway, map[string]string{
				"error": "gateway error",
			})
		}

		// Modify request to include proxy headers
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = server.URL.Scheme
			req.URL.Host = server.URL.Host
			req.Host = server.URL.Host

			// Add proxy headers
			req.Header.Set("X-Forwarded-For", c.Request().RemoteAddr)
			req.Header.Set("X-Forwarded-Proto", c.Scheme())
			req.Header.Set("X-Forwarded-Host", c.Request().Host)
			req.Header.Set("X-Real-IP", c.RealIP())
		}

		// Serve the request
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func (lb *LoadBalancer) GetServerStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := make(map[string]interface{})
	
	var healthyCount, unhealthyCount int
	serverStats := make([]map[string]interface{}, len(lb.servers))

	for i, server := range lb.servers {
		if server.Healthy {
			healthyCount++
		} else {
			unhealthyCount++
		}

		serverStats[i] = map[string]interface{}{
			"url":        server.URL.String(),
			"healthy":    server.Healthy,
			"weight":     server.Weight,
			"connections": server.Connections,
			"last_check": server.LastHealthCheck,
		}
	}

	stats["total_servers"] = len(lb.servers)
	stats["healthy_servers"] = healthyCount
	stats["unhealthy_servers"] = unhealthyCount
	stats["strategy"] = lb.strategy
	stats["servers"] = serverStats

	return stats
}

func (lb *LoadBalancer) AddServer(serverURL string, weight int) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL %s: %w", serverURL, err)
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	newServer := &Server{
		URL:     parsedURL,
		Healthy: true,
		Weight:  weight,
	}

	lb.servers = append(lb.servers, newServer)
	return nil
}

func (lb *LoadBalancer) RemoveServer(serverURL string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, server := range lb.servers {
		if server.URL.String() == serverURL {
			lb.servers = append(lb.servers[:i], lb.servers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("server %s not found", serverURL)
}
