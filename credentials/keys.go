package credentials

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SetKey sets one API key on a profile and returns the updated profile.
//
// The key is write-only: it is never returned by the API and is not included in SDK errors. A sign-in
// credential such as claude-subscription answers 400 not_an_api_key; connect sign-ins on the Boat
// dashboard instead. Running sandboxes that use the profile receive the new key immediately.
//
// The request is never retried.
func (s *Service) SetKey(ctx context.Context, profile string, credential model.CredentialID, req SetKeyRequest, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	if err := requireKeyValue(req.Value); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    profilePath(profile, "keys", string(credential)),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

// DeleteKey removes one credential from a profile and returns the updated profile.
//
// It removes an API key or a sign-in. The request is never retried.
func (s *Service) DeleteKey(ctx context.Context, profile string, credential model.CredentialID, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    profilePath(profile, "keys", string(credential)),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}
