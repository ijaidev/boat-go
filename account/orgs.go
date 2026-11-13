package account

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Orgs lists the wallets this account can bill: the personal account first, then every
// organization with its id, name, role and member count. The entry with Active set is the wallet
// new sandboxes bill when a request names none.
//
// An organization can be passed by id or name wherever an org is accepted. Use SetActiveOrg to
// change the account-wide default, or core.WithOrg to scope one request.
func (s *Service) Orgs(ctx context.Context, opts ...core.RequestOption) (*model.OrgListResponse, error) {
	var out model.OrgListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/orgs",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// activeOrgRequest is the body of PATCH /orgs/active. Org is required by the API, and null selects
// the personal account.
type activeOrgRequest struct {
	Org *string `json:"org"`
}

// SetActiveOrg changes the account-wide default wallet for new sandboxes.
//
// org is an organization id or name the account belongs to, or "personal". An empty org sends null,
// which also selects the personal account. The change applies to every client: the API, the CLI and
// the dashboard. Sandboxes already running keep the wallet they were created with.
//
// This differs from core.WithOrg, which scopes a single request through X-Boat-Org and does not
// change the default.
//
// A name shared by several organizations returns 409 ambiguous_org; pass the id instead. An
// organization the account does not belong to returns 403 not_org_member.
func (s *Service) SetActiveOrg(ctx context.Context, org string, opts ...core.RequestOption) (*model.ActiveOrgResponse, error) {
	body := activeOrgRequest{}
	if org != "" {
		body.Org = &org
	}

	var out model.ActiveOrgResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPatch,
		Path:   "/orgs/active",
		JSON:   body,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
