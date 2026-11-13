package account

import (
	"context"
	"errors"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// CreateScopedAPIKeyRequest is the body of POST /api-keys/scoped.
//
// At least one of Preset, Actions, SandboxIDs or EnvironmentIDs is required. Fields with empty
// values are omitted. TTL accepts a duration such as "90d" or "24h", a number of seconds, or
// "never"; the API default is 90 days and the maximum is 365 days.
type CreateScopedAPIKeyRequest struct {
	Name           string   `json:"name,omitempty"`
	TTL            string   `json:"ttl,omitempty"`
	Preset         string   `json:"preset,omitempty"`
	Actions        []string `json:"actions,omitempty"`
	SandboxIDs     []string `json:"sandboxIds,omitempty"`
	EnvironmentIDs []string `json:"environmentIds,omitempty"`
}

// APIKeys lists API key metadata and the scope catalog. Raw secrets are never returned after
// creation or rotation.
//
// A restricted API key receives an empty list; only an unrestricted account key or a browser/CLI
// session receives the account inventory. The catalog reports the available actions, presets, TTL
// bounds and whether scoped creation is enabled.
func (s *Service) APIKeys(ctx context.Context, opts ...core.RequestOption) (*model.APIKeysResponse, error) {
	var out model.APIKeysResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/api-keys",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateScopedAPIKey creates a scoped, expiring API key.
//
// The response carries the raw Secret exactly once. Store it before discarding the response and
// never log it. Later calls only return metadata.
//
// The request must name a preset, actions, sandbox ids or environment ids; the method rejects an
// empty scope locally so a misspelling cannot create a full-access key. The API answers 400 for an
// unknown field and 503 while scoped credential creation is not activated. Creation is not retried.
func (s *Service) CreateScopedAPIKey(ctx context.Context, req CreateScopedAPIKeyRequest, opts ...core.RequestOption) (*model.APIKeySecretResponse, error) {
	if req.Preset == "" && len(req.Actions) == 0 && len(req.SandboxIDs) == 0 && len(req.EnvironmentIDs) == 0 {
		return nil, errors.New("account: scoped api key requires a preset, actions, sandbox ids or environment ids")
	}

	var out model.APIKeySecretResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    "/api-keys/scoped",
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// RotateAPIKey replaces an API key's raw secret without changing its expiry or scope.
//
// The response carries the replacement Secret exactly once. Store it before discarding the response
// and never log it. An expired key cannot be rotated and returns 409 api_key_expired. Rotation is
// not retried because the previous secret stops working immediately.
func (s *Service) RotateAPIKey(ctx context.Context, apiKeyID string, opts ...core.RequestOption) (*model.APIKeySecretResponse, error) {
	if err := requireAPIKeyID(apiKeyID); err != nil {
		return nil, err
	}

	var out model.APIKeySecretResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    apiKeyPath(apiKeyID, "rotate"),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeAPIKey revokes a key immediately and returns the remaining key metadata.
//
// A key may revoke itself or a key it could create. Revocation is not retried.
func (s *Service) RevokeAPIKey(ctx context.Context, apiKeyID string, opts ...core.RequestOption) (*model.APIKeyRevokeResponse, error) {
	if err := requireAPIKeyID(apiKeyID); err != nil {
		return nil, err
	}

	var out model.APIKeyRevokeResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    apiKeyPath(apiKeyID),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// APIKeyUsage returns a key's request total for the 30-day UTC window and the sandboxes and Agents
// it created that still exist. It works for revoked keys owned by the authenticated account.
func (s *Service) APIKeyUsage(ctx context.Context, apiKeyID string, opts ...core.RequestOption) (*model.APIKeyUsageResponse, error) {
	if err := requireAPIKeyID(apiKeyID); err != nil {
		return nil, err
	}

	var out model.APIKeyUsageResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   apiKeyPath(apiKeyID, "usage"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
