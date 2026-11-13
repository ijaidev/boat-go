package account

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// DataRetentionConfirmation is the exact phrase required to enable zero data retention.
const DataRetentionConfirmation = "delete archived sandbox data"

// UpdateDataRetentionRequest is the body of PATCH /account/data-retention.
//
// Enabled changes zero data retention and needs Confirmation when true. SnapshotsOff and
// MonitoringOff change account defaults for new sandboxes only; existing sandboxes keep their
// setting. Use core.Ptr to send an explicit false.
type UpdateDataRetentionRequest struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	SnapshotsOff  *bool  `json:"snapshotsOff,omitempty"`
	MonitoringOff *bool  `json:"monitoringOff,omitempty"`
	Confirmation  string `json:"confirmation,omitempty"`
}

// DataRetention returns the account's zero-data-retention policy and snapshot/monitoring defaults.
//
// The endpoint is account-wide and is never cached. A per-Boat API key cannot read it and receives
// 403.
func (s *Service) DataRetention(ctx context.Context, opts ...core.RequestOption) (*model.DataRetentionPolicyResponse, error) {
	var out model.DataRetentionPolicyResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/account/data-retention",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDataRetention changes the account's zero-data-retention policy or its snapshot and
// monitoring defaults.
//
// The API refuses API keys and legacy permanent tokens for this endpoint: it needs an interactive
// browser sign-in session. The SDK does not and cannot work around that limitation. Expect 403 when
// authenticated with an API key.
//
// Enabling requires UpdateDataRetentionRequest.Confirmation to equal DataRetentionConfirmation
// exactly. Disabling affects future archives only and does not cancel deletion operations already
// accepted. The update is not retried because it changes account-wide deletion behavior.
func (s *Service) UpdateDataRetention(ctx context.Context, req UpdateDataRetentionRequest, opts ...core.RequestOption) (*model.DataRetentionPolicyResponse, error) {
	var out model.DataRetentionPolicyResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPatch,
		Path:    "/account/data-retention",
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
