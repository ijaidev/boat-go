package snapshots

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Delete permanently deletes one snapshot. The call is asynchronous; poll the account
// deletion-operation endpoint with the returned operation id.
//
// It sends X-Ascii-Confirm-Delete set to the snapshot id. A missing or mismatched confirmation is
// refused with 409.
func (s *Service) Delete(ctx context.Context, snapshotID string, opts ...core.RequestOption) (*model.DeletionOperation, error) {
	if err := requireSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	req := &core.Request{
		Method: http.MethodDelete,
		Path:   snapshotPath(snapshotID),
	}
	req.SetHeader("X-Ascii-Confirm-Delete", snapshotID)

	var out model.DeletionOperationResponse
	if err := s.do(ctx, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Operation, nil
}

// DeleteForSandbox permanently deletes every unpinned snapshot of a sandbox and returns the
// accepted deletion operations, one per snapshot.
//
// The sandbox itself and its saved snapshots stay. It sends X-Ascii-Confirm-Delete set to the
// sandbox id. The call is asynchronous; poll the account deletion-operation endpoint for each
// returned operation.
func (s *Service) DeleteForSandbox(ctx context.Context, sandboxID string, opts ...core.RequestOption) ([]model.DeletionOperation, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	req := &core.Request{
		Method: http.MethodDelete,
		Path:   sandboxPath(sandboxID, "snapshots"),
	}
	req.SetHeader("X-Ascii-Confirm-Delete", sandboxID)

	var out model.DeletionOperationListResponse
	if err := s.do(ctx, req, &out, opts...); err != nil {
		return nil, err
	}
	return out.Operations, nil
}
