package credentials

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ListProfiles returns every AI credential profile the caller can use, personal and organization
// profiles alike.
//
// The endpoint has no cursor pagination: every profile comes back in one response. Secret values are
// never returned; Connected and CustomApis carry names only.
func (s *Service) ListProfiles(ctx context.Context, opts ...core.RequestOption) ([]model.CredentialProfile, error) {
	var out model.CredentialProfileListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/credential-profiles",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Profiles, nil
}

// CreateProfile creates an empty profile or a copy of an existing one and returns it.
//
// The request is never retried: a lost response could otherwise create a second profile. Only an
// organization's owners can create a profile for it.
func (s *Service) CreateProfile(ctx context.Context, req CreateProfileRequest, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfileName(req.Name); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    "/credential-profiles",
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

// UpdateProfile renames a profile or makes it its owner's default.
//
// Sandboxes on the owner's default switch at their next start. The request is never retried.
func (s *Service) UpdateProfile(ctx context.Context, profile string, req UpdateProfileRequest, opts ...core.RequestOption) (*model.CredentialProfile, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	var out model.CredentialProfileResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    profilePath(profile),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

// DeleteProfile deletes one profile.
//
// A default profile answers 409 is_default and a profile named by an environment answers 409 in_use.
// Make another profile the default, or remove the environment reference, before deleting. The request
// is never retried.
func (s *Service) DeleteProfile(ctx context.Context, profile string, opts ...core.RequestOption) error {
	if err := requireProfile(profile); err != nil {
		return err
	}
	var out model.CredentialProfileDeletedResponse
	return s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    profilePath(profile),
		NoRetry: true,
	}, &out, opts...)
}
