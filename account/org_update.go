package account

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// UpdateOrgRequest is the body of PATCH /orgs/{org}.
//
// At least one field is required. A name change is owner-only; owners and admins can change the
// default visibility of new sandboxes.
type UpdateOrgRequest struct {
	Name                *string `json:"name,omitempty"`
	NewSandboxesPrivate *bool   `json:"newSandboxesPrivate,omitempty"`
}

// OrgUpdateResult is the fields changed by UpdateOrg.
//
// A field is nil when the request did not change it.
type OrgUpdateResult struct {
	Success             bool
	Name                *string
	NewSandboxesPrivate *bool
}

type orgUpdateResponse struct {
	Success             bool    `json:"success"`
	Name                *string `json:"name"`
	NewSandboxesPrivate *bool   `json:"newSandboxesPrivate"`
}

// UpdateOrg renames an organization or changes whether its new sandboxes start private.
//
// org is an organization id or name. A rename makes requests that use the old name fail; the id
// does not change. Existing sandboxes keep their visibility, and a fork of a private sandbox stays
// private. The call is never retried: a rename can make the old path invalid.
func (s *Service) UpdateOrg(ctx context.Context, org string, req UpdateOrgRequest, opts ...core.RequestOption) (*OrgUpdateResult, error) {
	if org == "" {
		return nil, errors.New("account: organization is required")
	}
	if req.Name == nil && req.NewSandboxesPrivate == nil {
		return nil, errors.New("account: at least one organization field is required")
	}

	var out orgUpdateResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPatch,
		Path:    "/orgs/" + url.PathEscape(org),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &OrgUpdateResult{
		Success:             out.Success,
		Name:                out.Name,
		NewSandboxesPrivate: out.NewSandboxesPrivate,
	}, nil
}
