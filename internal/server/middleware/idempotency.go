package middleware

import (
	"bytes"
	"context"
	
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Idempotency struct {
	rdb *redis.Client
	ttl time.Duration
}

type idempotencyRecord struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
}

func NewIdempotency(redisURL string, ttlHours int) (*Idempotency, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)

	// Test connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Idempotency{
		rdb: rdb,
		ttl: time.Duration(ttlHours) * time.Hour,
	}, nil
}

func (i *Idempotency) Idempotent() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only apply to state-changing methods
			method := c.Request().Method
			if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
				return next(c)
			}

			idempotencyKey := c.Request().Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing Idempotency-Key header")
			}

			tenantID, err := GetTenantID(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing tenant")
			}

			// Namespaced key: tenant + idempotency-key
			key := fmt.Sprintf("idempotency:%s:%s", tenantID, idempotencyKey)

			ctx := context.Background()

			// Check if we already have a response
			existing, err := i.rdb.Get(ctx, key).Result()
			if err == nil {
				// Hit! Return stored response
				var rec idempotencyRecord
				if json.Unmarshal([]byte(existing), &rec) != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "corrupted idempotency record")
				}

				for k, v := range rec.Headers {
					c.Response().Header().Set(k, v)
				}
				c.Response().WriteHeader(rec.StatusCode)
				_, writeErr := c.Response().Write(rec.Body)
				c.Response().Committed = true
				c.Logger().Info("Idempotent response served from cache", zap.String("key", key))
				return writeErr
			}

			if err != redis.Nil {
				c.Logger().Error("Redis error on idempotency check", zap.Error(err))
				// Continue — don't fail the request if Redis is down (degraded mode)
			}

			// Not found — proceed, but capture response
			bodyBackup := c.Request().Body
			if bodyBackup != nil {
				bodyBytes, _ := io.ReadAll(bodyBackup)
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Wrap response writer to capture output
			recorder := &responseRecorder{ResponseWriter: c.Response(), body: new(bytes.Buffer)}
			c.Response().Writer = recorder

			// Call handler
			if err := next(c); err != nil {
				c.Error(err)
			}

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			// Only store successful or client-error responses (4xx) — not transient 5xx
			if status < http.StatusInternalServerError {
				headers := map[string]string{}
				for k, vv := range recorder.Header() {
					if len(vv) > 0 {
						headers[k] = vv[0]
					}
				}

				rec := idempotencyRecord{
					StatusCode: status,
					Headers:    headers,
					Body:       json.RawMessage(recorder.body.Bytes()),
				}

				recJSON, _ := json.Marshal(rec)
				if err := i.rdb.Set(ctx, key, recJSON, i.ttl).Err(); err != nil {
					c.Logger().Warn("Failed to store idempotency record", zap.Error(err))
				} else {
					c.Logger().Info("Idempotency record stored", zap.String("key", key), zap.Int("status", status))
				}
			}

			return nil
		}
	}
}

// Helper to capture response
type responseRecorder struct {
	echo.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}