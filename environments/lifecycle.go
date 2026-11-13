package environments

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// List returns every non-deleted environment with its latest-version configuration.
//
// The endpoint has no cursor pagination: all environments come back in one slice. A default
// environment named base is created on first access if none exists.
func (s *Service) List(ctx context.Context, req ListRequest, opts ...core.RequestOption) ([]model.SandboxEnvironment, error) {
	var out model.SandboxEnvironmentListResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/environments",
		Query:  orgQuery(req.Org),
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return out.Environments, nil
}

// Create adds a named environment and returns the full environment list.
//
// The new environment starts at version 1 with every fine-grained credential flag on and
// safeForThirdParties off. Find it by Name in the returned slice. The request is never retried
// automatically because a lost response could otherwise create a duplicate environment.
func (s *Service) Create(ctx context.Context, req CreateRequest, opts ...core.RequestOption) ([]model.SandboxEnvironment, error) {
	if err := requireName(req.Name); err != nil {
		return nil, err
	}
	var out model.SandboxEnvironmentListResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    "/environments",
		Query:   orgQuery(req.Org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return out.Environments, nil
}

// Update renames, re-flags or edits an environment and returns its new latest version.
//
// Any flag or content change mints a new immutable version. Sandboxes already running stay pinned
// to their old version until Upgrade is called.
func (s *Service) Update(ctx context.Context, environmentID string, req UpdateRequest, opts ...core.RequestOption) (*model.SandboxEnvironment, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	var out model.SandboxEnvironmentResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    environmentPath(environmentID),
		Query:   orgQuery(req.Org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return out.Environment, nil
}

// Delete soft-deletes an environment and returns it when the server includes it.
//
// The default environment cannot be deleted, and at least one environment is always kept. The call
// is never retried automatically.
func (s *Service) Delete(ctx context.Context, environmentID string, opts ...core.RequestOption) (*model.SandboxEnvironment, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	var out model.SandboxEnvironmentResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    environmentPath(environmentID),
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return out.Environment, nil
}

// Upgrade moves active sandboxes to the environment's latest version.
//
// An empty AgentIDs upgrades every active sandbox of the caller that is on an older version. The
// server scrubs secrets the new version dropped and hot-pushes the new configuration. The call is
// never retried automatically because it has side effects on running sandboxes.
func (s *Service) Upgrade(ctx context.Context, environmentID string, req UpgradeRequest, opts ...core.RequestOption) (*model.UpgradeSandboxEnvironmentResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	var out model.UpgradeSandboxEnvironmentResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    environmentPath(environmentID, "upgrade"),
		Query:   orgQuery(req.Org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
