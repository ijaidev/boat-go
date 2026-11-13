package credentials

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SetCustomAPI adds or replaces one custom API on a profile and returns the updated profile.
//
// The sandbox receives the secret as the environment variable named in the path, and the base URL as
// <NAME>_BASE_URL. Provider names (ANTHROPIC_*, OPENAI_* and similar) and names ending in _BASE_URL
// are reserved. The value is write-only and is never logged. The request is never retried.
func (s *Service) SetCustomAPI(ctx context.Context, profile, name string, req SetCustomAPIRequest, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCustomAPI(name); err != nil {
		return nil, err
	}
	if err := requireCustomAPIValue(req.Value); err != nil {
		return nil, err
	}
	if err := requireCustomAPIBaseURL(req.BaseURL); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    profilePath(profile, "custom-apis", name),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

// DeleteCustomAPI removes one custom API from a profile and returns the updated profile.
//
// The request is never retried.
func (s *Service) DeleteCustomAPI(ctx context.Context, profile, name string, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCustomAPI(name); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    profilePath(profile, "custom-apis", name),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}
