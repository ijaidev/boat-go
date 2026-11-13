package environments

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SetVar sets or replaces one environment variable.
//
// The call mints a new immutable version and does not rewrite the rest of the environment. Value is
// a secret: the SDK never logs it or puts it in error text. Set the organization with WithOrg. The
// request is never retried automatically because a retry could mint another version.
func (s *Service) SetVar(ctx context.Context, environmentID, key, value string, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requireVarKey(key); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodPut,
		Path:   environmentPath(environmentID, "vars", key),
		JSON: struct {
			Value string `json:"value"`
		}{Value: value},
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteVar removes one environment variable and mints a new immutable version.
//
// It does not rewrite the rest of the environment. Set the organization with WithOrg. The request is
// never retried automatically.
func (s *Service) DeleteVar(ctx context.Context, environmentID, key string, opts ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error) {
	if err := requireEnvironmentID(environmentID); err != nil {
		return nil, err
	}
	if err := requireVarKey(key); err != nil {
		return nil, err
	}
	var out model.EnvironmentItemChangeResponse
	err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    environmentPath(environmentID, "vars", key),
		NoRetry: true,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
