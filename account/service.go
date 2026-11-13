// Package account wraps the Boat account surface: the authenticated user, organizations and the
// active wallet, creation limits, API keys, data retention, deletion operations and the legacy
// sandbox secrets setup.
package account

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat account API.
type Service struct {
	d core.Doer
}

// New returns an account service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("account: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}

func apiKeyPath(id string, suffix ...string) string {
	path := "/api-keys/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func requireAPIKeyID(id string) error {
	if id == "" {
		return errors.New("account: api key id is required")
	}
	return nil
}

func requireOperationID(id string) error {
	if id == "" {
		return errors.New("account: deletion operation id is required")
	}
	return nil
}
