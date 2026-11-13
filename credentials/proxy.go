package credentials

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// GetProxy returns the AI credentials proxy switch and its price for the account in scope.
//
// The sandbox receives placeholder keys and base URLs that point at Boat while the proxy is on; real
// provider keys never enter the sandbox.
func (s *Service) GetProxy(ctx context.Context, req GetProxyRequest, opts ...core.RequestOption) (*model.AiProxySettings, error) {
	var out model.AiProxySettings
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/ai-proxy",
		Query:  proxyQuery(req.Org),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetProxy turns the AI credentials proxy on or off for the account in scope and returns the new
// state.
//
// It is a security-sensitive and billable change. While the proxy is on, the wallet pays for each UTC
// day the proxy is on and per proxied request, and sandboxes receive placeholders instead of real
// keys. Running sandboxes switch at their next start. Only an organization's owner can change the
// switch for that organization. The request is never retried.
func (s *Service) SetProxy(ctx context.Context, req SetProxyRequest, opts ...core.RequestOption) (*model.AiProxySettings, error) {
	var out model.AiProxySettings
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    "/ai-proxy",
		Query:   proxyQuery(req.Org),
		JSON:    model.UpdateAiProxyRequest{Enabled: req.Enabled},
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func proxyQuery(org string) url.Values {
	if org == "" {
		return nil
	}
	return url.Values{"org": {org}}
}
