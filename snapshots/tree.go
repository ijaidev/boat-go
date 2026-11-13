package snapshots

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Tree returns the flat file and directory listing captured in a snapshot.
//
// Entries are paths relative to the snapshot root, exactly the files a resume or download returns;
// base-image system entries are excluded. Legacy snapshots and inventories too large to expand
// answer with TreeAvailable false and a Reason instead of an error.
func (s *Service) Tree(ctx context.Context, snapshotID string, opts ...core.RequestOption) (*model.SnapshotTreeResponse, error) {
	if err := requireSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	var out model.SnapshotTreeResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   snapshotPath(snapshotID, "tree"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
