package ci

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// State filters the job list in a CI overview.
type State string

const (
	// StateAll includes every job.
	StateAll State = "all"
	// StateActive includes queued and running jobs.
	StateActive State = "active"
	// StateFailed includes failed, never-started and refused jobs.
	StateFailed State = "failed"
)

// OverviewRequest filters GET /ci.
//
// Account and Repo narrow the setup problems, counts, stats and jobs together. State narrows only
// the job list. An empty State means the server default, all. Limit must be between 1 and 500; zero
// means the server default of 200.
type OverviewRequest struct {
	Account string
	Repo    string
	State   State
	Limit   int
}

// Overview returns the CI setup state, linked GitHub accounts, 7-day counts and stats, and jobs.
//
// It covers the accounts linked to the caller and to the organization in scope. The API key needs
// account.read. Use StateActive or StateFailed to narrow the job list, and poll Overview or Job while
// jobs run.
func (s *Service) Overview(ctx context.Context, req OverviewRequest, opts ...core.RequestOption) (*model.CiOverviewResponse, error) {
	if req.Limit < 0 || req.Limit > 500 {
		return nil, fmt.Errorf("ci: limit %d is outside 0-500", req.Limit)
	}
	switch req.State {
	case "", StateAll, StateActive, StateFailed:
	default:
		return nil, fmt.Errorf("ci: unsupported state %q", req.State)
	}

	query := make(url.Values)
	if req.Account != "" {
		query.Set("account", req.Account)
	}
	if req.Repo != "" {
		query.Set("repo", req.Repo)
	}
	if req.State != "" {
		query.Set("state", string(req.State))
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}

	var out model.CiOverviewResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/ci",
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
