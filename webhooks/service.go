// Package webhooks wraps the Boat webhook management API and verifies inbound webhook deliveries.
//
// Management covers listing, creating, reading, updating, deleting and rotating the signing secret
// of account-wide sandbox lifecycle endpoints. Verify checks the HMAC-SHA256 signature of a
// delivery before parsing its JSON body.
//
// Verification is stateless. The caller owns deduplication, secret storage and the decision to
// acknowledge a delivery; use Event.DeliveryID as the idempotency key.
package webhooks

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat webhooks API.
type Service struct {
	d core.Doer
}

// New returns a webhook service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("webhooks: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}

func webhookPath(id string, suffix ...string) string {
	path := "/webhooks/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func requireID(id string) error {
	if id == "" {
		return errors.New("webhooks: webhook id is required")
	}
	return nil
}
