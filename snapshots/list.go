package snapshots

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ListRequest filters the snapshot list endpoints.
//
// Limit is left out of the query when zero, so the server default applies. Sort is "asc" or
// "desc"; the server default is "desc".
type ListRequest struct {
	Limit  int
	Cursor string
	Sort   string
}

// List returns one page of completed snapshots across every sandbox in the account.
func (s *Service) List(ctx context.Context, req ListRequest, opts ...core.RequestOption) (core.Page[model.SnapshotSummary], error) {
	return s.list(ctx, "/snapshots", req, opts...)
}

// ListForSandbox returns one page of completed snapshots for one sandbox.
func (s *Service) ListForSandbox(ctx context.Context, sandboxID string, req ListRequest, opts ...core.RequestOption) (core.Page[model.SnapshotSummary], error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return core.Page[model.SnapshotSummary]{}, err
	}
	return s.list(ctx, sandboxPath(sandboxID, "snapshots"), req, opts...)
}

func (s *Service) list(ctx context.Context, path string, req ListRequest, opts ...core.RequestOption) (core.Page[model.SnapshotSummary], error) {
	var out model.SnapshotListResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   path,
		Query:  listQuery(req),
	}, &out, opts...)
	if err != nil {
		return core.Page[model.SnapshotSummary]{}, err
	}

	page := core.Page[model.SnapshotSummary]{Items: out.Snapshots}
	if out.PageInfo != nil {
		page.HasMore = out.PageInfo.HasMore
		if out.PageInfo.NextCursor != nil {
			page.NextCursor = *out.PageInfo.NextCursor
		}
	}
	return page, nil
}

// Latest returns the most recent completed snapshot for a sandbox, or nil when it has none.
func (s *Service) Latest(ctx context.Context, sandboxID string, opts ...core.RequestOption) (*model.SnapshotSummary, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	var out model.SnapshotLatestResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "snapshots", "latest"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Snapshot, nil
}

func listQuery(req ListRequest) url.Values {
	query := make(url.Values)
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}
	if req.Sort != "" {
		query.Set("sort", req.Sort)
	}
	return query
}
