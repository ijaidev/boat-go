package environments

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SetSecretFile adds or replaces one secret file by path.
//
// The call mints a new immutable version and does not rewrite the rest of the environment.
// Contents is a secret: the SDK never logs it or puts it in error text. The request is never
// retried automatically because a retry could mint another version.
func (s *Service) SetSecretFile(ctx context.Context, environmentID string, req SecretFileRequest, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    environmentPath(environmentID, "secret-files"),
		Query:   orgQuery(req.Org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteSecretFile removes the secret file at path and mints a new immutable version.
//
// Set the organization with WithOrg. The request is never retried automatically.
func (s *Service) DeleteSecretFile(ctx context.Context, environmentID, path string, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requirePath(path); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    environmentPath(environmentID, "secret-files"),
		Query:   url.Values{"path": {path}},
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
