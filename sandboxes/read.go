package sandboxes

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Get returns one sandbox.
func (s *Service) Get(ctx context.Context, id string, opts ...core.RequestOption) (*model.Sandbox, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out model.SandboxInfoResponse
	if err := s.do(ctx, &core.Request{Method: http.MethodGet, Path: sandboxPath(id)}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Sandbox, nil
}

// List returns one page of sandboxes.
func (s *Service) List(ctx context.Context, req ListRequest, opts ...core.RequestOption) (core.Page[model.Sandbox], error) {
	var out model.SandboxListResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/sandboxes",
		Query:  listQuery(req),
	}, &out, opts...)
	if err != nil {
		return core.Page[model.Sandbox]{}, err
	}

	page := core.Page[model.Sandbox]{Items: out.Sandboxes}
	if out.PageInfo != nil {
		page.HasMore = out.PageInfo.HasMore
		if out.PageInfo.NextCursor != nil {
			page.NextCursor = *out.PageInfo.NextCursor
		}
	}
	return page, nil
}

// Update changes a sandbox name, subdomain or auto-stop TTL.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest, opts ...core.RequestOption) (*model.Sandbox, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out model.SandboxInfoResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodPatch,
		Path:   sandboxPath(id),
		JSON:   req,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out.Sandbox, nil
}

// Usage returns the machine time and cost of one sandbox. It also works after a stop.
func (s *Service) Usage(ctx context.Context, id string, req UsageRequest, opts ...core.RequestOption) (*model.SandboxUsageResponse, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	query := make(url.Values)
	if req.Since != nil {
		query.Set("since", req.Since.UTC().Format(time.RFC3339))
	}
	if req.Until != nil {
		query.Set("until", req.Until.UTC().Format(time.RFC3339))
	}
	var out model.SandboxUsageResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(id, "usage"),
		Query:  query,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
	if len(req.State) > 0 {
		query.Set("state", strings.Join(req.State, ","))
	}
	if req.Name != "" {
		query.Set("name", req.Name)
	}
	if req.NamePrefix != "" {
		query.Set("namePrefix", req.NamePrefix)
	}
	if req.StoppedBefore != nil {
		query.Set("stoppedBefore", req.StoppedBefore.UTC().Format(time.RFC3339))
	}
	if req.DataExpired != nil {
		query.Set("dataExpired", strconv.FormatBool(*req.DataExpired))
	}
	for _, label := range req.Label {
		query.Add("label", label)
	}
	if len(req.Include) > 0 {
		query.Set("include", strings.Join(req.Include, ","))
	}
	return query
}
