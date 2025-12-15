package middleware

import (
	"github.com/labstack/echo/v4"
)

// Authorize checks the request against OPA policies.
// This is a placeholder implementation.
func Authorize() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Implement OPA check here
			// For now, allow all requests
			return next(c)
		}
	}
}
