package account

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// LimitsRequest selects the wallet Limits reads. An empty field omits that query parameter.
type LimitsRequest struct {
	// Org is an organization id or name, or "personal". It is the spec's org query parameter.
	Org string

	// TeamID is the legacy alias for Org. The API gives it precedence when both are set.
	TeamID string
}

// Limits returns the creation and concurrency limits for the wallet a sandbox create would bill:
// the requested organization, else the account's active wallet, else the personal account.
//
// The call is read-only. Use it before a batch of starts to check CanStart, the remaining start
// windows and the concurrent-sandbox budget. TeamID on the response is set when an organization
// wallet was read.
//
// LimitsRequest.Org is a query parameter. core.WithOrg sets the X-Boat-Org header instead; the query
// wins when both are set.
func (s *Service) Limits(ctx context.Context, req LimitsRequest, opts ...core.RequestOption) (*model.LimitsResponse, error) {
	query := make(url.Values)
	if req.Org != "" {
		query.Set("org", req.Org)
	}
	if req.TeamID != "" {
		query.Set("teamId", req.TeamID)
	}

	var out model.LimitsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/limits",
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
