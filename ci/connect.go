package ci

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ConnectRequest is the body of POST /ci/connect.
//
// Set Account to authorize and link an existing GitHub installation; leave it empty to install the
// Boat CI app. Org selects the organization whose wallet pays for the jobs; omit it for the wallet in
// scope. OrgApp installs the organization variant, which needs only the Self-hosted runners
// permission and works for GitHub organizations only.
type ConnectRequest struct {
	Org     string `json:"org,omitempty"`
	Account string `json:"account,omitempty"`
	OrgApp  *bool  `json:"orgApp,omitempty"`
}

// Connect returns a GitHub URL that installs the Boat CI app or authorizes an existing installation.
//
// Open InstallURL in a browser; it is valid for 30 minutes and carries a one-time authorization
// state, so treat it as a secret and never log it. The request is never retried: a retry could create
// a second installation or authorization. The API key needs account.admin.
func (s *Service) Connect(ctx context.Context, req ConnectRequest, opts ...core.RequestOption) (*model.CiConnectResponse, error) {
	var out model.CiConnectResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    "/ci/connect",
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
