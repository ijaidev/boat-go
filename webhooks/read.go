package webhooks

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// List returns every registered webhook endpoint. The endpoint is not paginated and the signing
// secret is never included.
func (s *Service) List(ctx context.Context, opts ...core.RequestOption) ([]model.Webhook, error) {
	var out model.WebhookListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/webhooks",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Webhooks, nil
}

// Get returns one webhook endpoint without its signing secret.
func (s *Service) Get(ctx context.Context, webhookID string, opts ...core.RequestOption) (*model.Webhook, error) {
	if err := requireID(webhookID); err != nil {
		return nil, err
	}
	var out model.WebhookResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   webhookPath(webhookID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Webhook, nil
}
