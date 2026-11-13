package account

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// UpdateSecretsRequest is the body of POST /secrets. It is a full replacement, not a patch: send
// the complete environment file and every secret file that should remain configured.
//
// Environment secret values in EnvContents and SecretFiles are sensitive. The SDK sends them as-is
// and never logs them.
type UpdateSecretsRequest struct {
	EnvContents string             `json:"envContents"`
	SecretFiles []model.SecretFile `json:"secretFiles"`
}

// Secrets returns the account's legacy sandbox secret setup: the .env-style environment contents
// and the configured secret files.
//
// The values are account secrets. Never log the response, store it in a world-readable place, or
// include it in an error. The returned EnvContents is plaintext by design because the caller may
// need to edit and resend it.
func (s *Service) Secrets(ctx context.Context, opts ...core.RequestOption) (*model.SecretsResponse, error) {
	var out model.SecretsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/secrets",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateSecrets replaces the account's legacy sandbox secret setup.
//
// Omitted fields are treated by the server as empty values, but the SDK always sends both fields so
// a zero-value request clears the setup. The response reports how many active sandboxes received
// the new values in Pushed.
//
// The values are secrets: never log the request or the response.
func (s *Service) UpdateSecrets(ctx context.Context, req UpdateSecretsRequest, opts ...core.RequestOption) (*model.SecretsResponse, error) {
	if req.SecretFiles == nil {
		req.SecretFiles = []model.SecretFile{}
	}

	var out model.SecretsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   "/secrets",
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
