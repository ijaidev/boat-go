package ci

import (
	"context"
	"errors"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// PublicReposRequest is the body of POST /ci/public.
//
// Account is the linked GitHub account login or installation id. Allow turns jobs from that account's
// public repositories on or off; nil keeps the current value. Users restricts runs to the listed
// GitHub users: nil keeps the current list, a pointer to an empty slice allows anyone, and a non-empty
// slice allows only those users. DryRun checks the users on GitHub without changing anything. Org is
// the organization in scope and may be empty.
type PublicReposRequest struct {
	Account string    `json:"account"`
	Allow   *bool     `json:"allow,omitempty"`
	Users   *[]string `json:"users,omitempty"`
	DryRun  *bool     `json:"dryRun,omitempty"`
	Org     string    `json:"org,omitempty"`
}

// PublicRepos sets whether an account's public repositories may run jobs and which GitHub users may
// start them.
//
// A run is allowed only when both its author and the user who started it, if different, are on the
// list. Each user is checked on GitHub. Only the person who linked the account or an owner or admin of
// its organization can change the setting. The API key needs account.admin.
func (s *Service) PublicRepos(ctx context.Context, req PublicReposRequest, opts ...core.RequestOption) (*model.CiPublicReposResponse, error) {
	if req.Account == "" {
		return nil, errors.New("ci: account is required")
	}
	var out model.CiPublicReposResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   "/ci/public",
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
