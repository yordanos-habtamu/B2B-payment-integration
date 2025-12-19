package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yordanos-habtamu/b2b-payments/internal/policy"
)

type OPAMiddleware struct {
	opaClient *policy.OPAClient
}

func NewOPAMiddleware() (*OPAMiddleware, error) {
	opaClient, err := policy.NewOPAClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OPA client: %w", err)
	}

	return &OPAMiddleware{
		opaClient: opaClient,
	}, nil
}

// Authorize checks the request against OPA policies.
func (om *OPAMiddleware) Authorize() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get tenant ID from context (set by TenantExtraction middleware)
			tenantID, err := GetTenantID(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing tenant context")
			}

			// Prepare policy input
			input := policy.PolicyInput{
				TenantID:  tenantID,
				Method:    c.Request().Method,
				Path:      c.Request().URL.Path,
				Headers:   map[string]interface{}{},
				UserAgent: c.Request().UserAgent(),
				ClientIP:  getClientIP(c),
				Attributes: map[string]interface{}{
					"permissions": []string{"create_payments", "update_payments", "read_payments"},
					"tier":        "enterprise",
					"compliant":   true,
				},
			}

			// Add relevant headers for policy evaluation
			relevantHeaders := []string{
				"Content-Type", "Accept", "Idempotency-Key",
				"X-Request-ID", "X-Forwarded-For", "X-Real-IP",
			}
			for _, header := range relevantHeaders {
				if value := c.Request().Header.Get(header); value != "" {
					input.Headers[header] = value
				}
			}

			// Evaluate policy
			result, err := om.opaClient.Evaluate(context.Background(), input)
			if err != nil {
				c.Logger().Error("OPA evaluation failed", "error", err, "tenant", tenantID)
				return echo.NewHTTPError(http.StatusInternalServerError, "authorization check failed")
			}

			// Check if request is allowed
			if !result.Allowed {
				c.Logger().Warn("Access denied by policy", 
					"tenant", tenantID,
					"method", input.Method,
					"path", input.Path,
					"reason", result.Reason,
				)
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("access denied: %s", result.Reason))
			}

			// Store policy result in context for potential downstream use
			c.Set("policy_result", result)

			// Log successful authorization for audit
			c.Logger().Info("Request authorized",
				"tenant", tenantID,
				"method", input.Method,
				"path", input.Path,
			)

			return next(c)
		}
	}
}

// Helper function to extract client IP from request
func getClientIP(c echo.Context) string {
	// This is a simplified version - in production, you might want more sophisticated IP extraction
	return c.Request().RemoteAddr
}
