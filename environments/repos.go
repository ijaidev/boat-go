package environments

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// AddRepo adds one repository to an environment, or updates its base branch.
//
// The path takes the repository databaseId from ListRepos. The call mints a new immutable version;
// existing sandboxes stay pinned until Upgrade. The request is never retried automatically.
func (s *Service) AddRepo(ctx context.Context, environmentID string, req AddRepoRequest, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requireRepoID(req.RepositoryID); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    environmentPath(environmentID, "repos"),
		Query:   orgQuery(req.Org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveRepo removes one repository from an environment and mints a new immutable version.
//
// The path takes the repository databaseId from ListRepos. Set the organization with WithOrg. The
// request is never retried automatically.
func (s *Service) RemoveRepo(ctx context.Context, environmentID, repositoryID string, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requireRepoID(repositoryID); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    environmentPath(environmentID, "repos", repositoryID),
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRepos returns GitHub repositories grouped by installation plus the current selection for
// future sandboxes.
//
// The /repos routes have no org parameter: they operate on the default environment. PageInfo carries
// the cursor for the next page when HasMore is true.
func (s *Service) ListRepos(ctx context.Context, req ListReposRequest, opts ...core.RequestOption) (*model.ReposResponse, error) {
	query := make(url.Values)
	if req.Sync != nil {
		query.Set("sync", strconv.FormatBool(*req.Sync))
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}
	if req.Sort != "" {
		query.Set("sort", req.Sort)
	}
	if req.Query != "" {
		query.Set("q", req.Query)
	}
	if req.Selected != nil {
		query.Set("selected", strconv.FormatBool(*req.Selected))
	}

	var out model.ReposResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/repos",
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// SelectRepo selects one repository for future sandboxes and returns the updated selection.
//
// The call is idempotent: selecting an already-selected repository updates its base branch. It has
// no effect on sandboxes that already exist.
func (s *Service) SelectRepo(ctx context.Context, req SelectRepoRequest, opts ...core.RequestOption) (*model.RepoSelectionResponse, error) {
	if err := requireRepoID(req.RepositoryID); err != nil {
		return nil, err
	}
	var out model.RepoSelectionResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   "/repos",
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
