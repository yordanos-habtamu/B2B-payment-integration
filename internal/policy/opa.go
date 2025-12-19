package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
)

type OPAClient struct {
	policy *rego.PreparedEvalQuery
	store  rego.Store
}

type PolicyInput struct {
	TenantID   string                 `json:"tenant_id"`
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	Headers    map[string]interface{} `json:"headers"`
	UserAgent  string                 `json:"user_agent"`
	ClientIP   string                 `json:"client_ip"`
	Attributes map[string]interface{} `json:"attributes"`
}

type PolicyResult struct {
	Allowed bool                   `json:"allowed"`
	Reason  string                 `json:"reason,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

const defaultPolicy = `
package b2b_payments.auth

default allow = false

allow {
    input.method == "GET"
    input.path == "/health"
}

allow {
    input.method == "GET"
    input.path == "/api/v1/whoami"
    has_tenant_id
}

allow {
    input.method == "GET"
    input.path == "/api/v1/payments"
    has_tenant_id
    tenant_active
}

allow {
    input.method == "POST"
    input.path == "/api/v1/payments"
    has_tenant_id
    tenant_active
    tenant_can_create_payments
}

allow {
    input.method == "GET"
    starts_with(input.path, "/api/v1/payments/")
    has_tenant_id
    tenant_active
    tenant_can_access_payment
}

allow {
    input.method == "PUT"
    starts_with(input.path, "/api/v1/payments/")
    has_tenant_id
    tenant_active
    tenant_can_update_payment
}

has_tenant_id {
    input.tenant_id != ""
}

tenant_active {
    # In a real implementation, this would check against a database
    # For now, we'll assume all tenants are active
    true
}

tenant_can_create_payments {
    # Check if tenant has permission to create payments
    # This could be based on subscription tier, compliance status, etc.
    input.attributes["permissions"][_] == "create_payments"
}

tenant_can_access_payment {
    # Check if tenant can access specific payment
    # This would involve checking payment ownership
    true
}

tenant_can_update_payment {
    # Check if tenant can update specific payment
    # This would involve checking payment status and ownership
    input.attributes["permissions"][_] == "update_payments"
}
`

func NewOPAClient() (*OPAClient, error) {
	// Create in-memory store for policies
	store := inmem.New()

	// Compile the policy
	ctx := context.Background()
	policy, err := rego.New(
		rego.Query("data.b2b_payments.auth.allow"),
		rego.Store(store),
		rego.Module("b2b_payments.rego", defaultPolicy),
	).PrepareForEval(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to compile OPA policy: %w", err)
	}

	return &OPAClient{
		policy: &policy,
		store:  store,
	}, nil
}

func (opa *OPAClient) Evaluate(ctx context.Context, input PolicyInput) (*PolicyResult, error) {
	// Prepare input for OPA
	inputMap := map[string]interface{}{
		"tenant_id":  input.TenantID,
		"method":      input.Method,
		"path":        input.Path,
		"headers":     input.Headers,
		"user_agent":  input.UserAgent,
		"client_ip":   input.ClientIP,
		"attributes":  input.Attributes,
	}

	// Evaluate the policy
	results, err := (*opa.policy).Eval(ctx, rego.EvalInput(inputMap))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Parse results
	if len(results) == 0 {
		return &PolicyResult{Allowed: false, Reason: "no policy decision"}, nil
	}

	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return &PolicyResult{Allowed: false, Reason: "invalid policy result"}, nil
	}

	result := &PolicyResult{
		Allowed: allowed,
	}

	if !allowed {
		result.Reason = "access denied by policy"
	}

	return result, nil
}

func (opa *OPAClient) UpdatePolicy(policy string) error {
	ctx := context.Background()
	newPolicy, err := rego.New(
		rego.Query("data.b2b_payments.auth.allow"),
		rego.Store(opa.store),
		rego.Module("b2b_payments.rego", policy),
	).PrepareForEval(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to compile new policy: %w", err)
	}

	opa.policy = &newPolicy
	return nil
}

// Helper function to extract client IP from request
func getClientIP(c echo.Context) string {
	// Check X-Forwarded-For header first
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.Request().RemoteAddr
}
