package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yordanos-habtamu/b2b-payments/internal/server/middleware"
)

// MetricsMiddleware provides Prometheus metrics for HTTP requests
func (m *MetricsCollector) MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			
			// Process request
			err := next(c)
			
			// Calculate duration
			duration := time.Since(start).Seconds()
			
			// Extract tenant ID for metrics
			tenantID := "unknown"
			if tid, err := middleware.GetTenantID(c); err == nil {
				tenantID = tid
			}
			
			// Record metrics
			method := c.Request().Method
			endpoint := c.Path()
			statusCode := strconv.Itoa(c.Response().Status)
			
			m.RecordHTTPRequest(method, endpoint, statusCode, tenantID, duration)
			
			return err
		}
	}
}

// PaymentMetricsMiddleware provides metrics for payment operations
func (m *MetricsCollector) PaymentMetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only apply to payment endpoints
			if !isPaymentEndpoint(c.Path()) {
				return next(c)
			}
			
			start := time.Now()
			
			// Process request
			err := next(c)
			
			// Calculate duration
			duration := time.Since(start).Seconds()
			
			// Extract tenant ID
			tenantID := "unknown"
			if tid, err := middleware.GetTenantID(c); err == nil {
				tenantID = tid
			}
			
			// Record payment processing metrics
			status := "success"
			if err != nil {
				status = "error"
			}
			
			m.RecordPaymentProcessing(tenantID, status, duration)
			
			return err
		}
	}
}

func isPaymentEndpoint(path string) bool {
	return path == "/api/v1/payments" || 
		   path == "/api/v1/payments/stats" ||
		   len(path) > len("/api/v1/payments/") && 
		   path[:len("/api/v1/payments/")] == "/api/v1/payments/"
}
