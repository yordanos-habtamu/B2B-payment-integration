package middleware

import (
	
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	TenantContextKey = "tenant_id"
)

// TenantExtraction extracts tenant ID from the verified client certificate's Common Name.
// Expected format: CN=tenant-<tenant_id>.yourorg.com
func TenantExtraction() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tlsConnState := c.Request().TLS
			if tlsConnState == nil || len(tlsConnState.PeerCertificates) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing client certificate")
			}

			clientCert := tlsConnState.PeerCertificates[0]

			// We only trust verified chains (mTLS already enforces this)
			if tlsConnState.VerifiedChains == nil || len(tlsConnState.VerifiedChains) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "client certificate not verified by trusted CA")
			}

			cn := clientCert.Subject.CommonName
			if cn == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "client certificate missing Common Name")
			}

			// Extract tenant ID from expected pattern: tenant-<id>.yourorg.com
			if !strings.HasPrefix(cn, "tenant-") || !strings.HasSuffix(cn, ".yourorg.com") {
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("invalid client certificate CN: %s", cn))
			}

			tenantID := strings.TrimSuffix(strings.TrimPrefix(cn, "tenant-"), ".yourorg.com")
			if tenantID == "" {
				return echo.NewHTTPError(http.StatusForbidden, "empty tenant ID in certificate")
			}

			// Inject into context
			c.Set(TenantContextKey, tenantID)

			// Optional: log for audit
			c.Logger().Infof("Authenticated tenant: %s (from cert SN: %x)", tenantID, clientCert.SerialNumber)

			return next(c)
		}
	}
}

// GetTenantID is a helper for handlers/services
func GetTenantID(c echo.Context) (string, error) {
	tenantID, ok := c.Get(TenantContextKey).(string)
	if !ok || tenantID == "" {
		return "", fmt.Errorf("tenant ID not found in context")
	}
	return tenantID, nil
}