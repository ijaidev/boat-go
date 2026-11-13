package credentials

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Usage returns proxied AI usage per UTC day, credential and model, and in total.
//
// Without UsageRequest.Profile the usage charged to the wallet in scope is returned. With Profile,
// the requests made with that profile's credentials are counted instead, whoever pays; only the
// profile's managers can read it. Usage is read-only.
func (s *Service) Usage(ctx context.Context, req UsageRequest, opts ...core.RequestOption) (*model.AccountAiUsageResponse, error) {
	query := make(url.Values)
	if req.Days != 0 {
		query.Set("days", strconv.Itoa(req.Days))
	}
	if req.Profile != "" {
		query.Set("profile", req.Profile)
	}
	if req.Org != "" {
		query.Set("org", req.Org)
	}

	var out model.AccountAiUsageResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/ai-usage",
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// SandboxUsage returns one sandbox's proxied AI usage per UTC day, credential and model, and whether
// the proxy blocks it. Usage is read-only.
func (s *Service) SandboxUsage(ctx context.Context, sandboxID string, req SandboxUsageRequest, opts ...core.RequestOption) (*model.SandboxAiUsageResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	query := make(url.Values)
	if req.Days != 0 {
		query.Set("days", strconv.Itoa(req.Days))
	}

	var out model.SandboxAiUsageResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "ai-usage"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
