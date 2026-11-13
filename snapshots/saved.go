package snapshots

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SavedListRequest filters GET /saved-snapshots.
//
// Org names the billing wallet as the spec's org query parameter accepts it: an organization id,
// an organization name, or "personal". It wins over the X-Boat-Org header set by core.WithOrg.
type SavedListRequest struct {
	Name       string
	NamePrefix string
	Org        string
}

// ListSaved returns the saved snapshots of the account, newest first.
//
// The endpoint has no cursor pagination: every saved snapshot comes back in one page, so
// core.Page.HasMore is always false and NextCursor is empty.
func (s *Service) ListSaved(ctx context.Context, req SavedListRequest, opts ...core.RequestOption) (core.Page[model.SavedSnapshot], error) {
	query := make(url.Values)
	if req.Name != "" {
		query.Set("name", req.Name)
	}
	if req.NamePrefix != "" {
		query.Set("namePrefix", req.NamePrefix)
	}
	if req.Org != "" {
		query.Set("org", req.Org)
	}

	var out model.SavedSnapshotListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/saved-snapshots",
		Query:  query,
	}, &out, opts...); err != nil {
		return core.Page[model.SavedSnapshot]{}, err
	}
	return core.Page[model.SavedSnapshot]{Items: out.Snapshots}, nil
}

// Save freezes the current state of a sandbox under a name and returns the accepted saved
// snapshot in status saving.
//
// The capture runs in the background: a running sandbox takes a fresh snapshot first, which can
// run minutes. Poll GetSaved until the status is "ready" or "failed". Reusing an existing name
// replaces its artifact once the new save is ready; a save already running under that name answers
// 409 save_in_progress.
func (s *Service) Save(ctx context.Context, sandboxID, name string, opts ...core.RequestOption) (*model.SavedSnapshot, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireName(name); err != nil {
		return nil, err
	}

	var out model.SavedSnapshotSavingResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   "/saved-snapshots",
		JSON: model.SavedSnapshotSaveRequest{
			Name:      name,
			SandboxID: sandboxID,
		},
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Snapshot, nil
}

// GetSaved returns one saved snapshot by name.
func (s *Service) GetSaved(ctx context.Context, name string, opts ...core.RequestOption) (*model.SavedSnapshot, error) {
	if err := requireName(name); err != nil {
		return nil, err
	}
	var out model.SavedSnapshotInfoResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   savedPath(name),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Snapshot, nil
}

// DeleteSaved removes a saved snapshot and releases its storage.
//
// Sandboxes already deployed from it are unaffected. Deleting a name whose save is still running
// answers 409 save_in_progress.
func (s *Service) DeleteSaved(ctx context.Context, name string, opts ...core.RequestOption) error {
	if err := requireName(name); err != nil {
		return err
	}
	var out model.SavedSnapshotDeletedResponse
	return s.do(ctx, &core.Request{
		Method: http.MethodDelete,
		Path:   savedPath(name),
	}, &out, opts...)
}
