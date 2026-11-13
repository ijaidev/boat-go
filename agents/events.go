package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Events returns one page of sandbox work, lifecycle and progress events.
//
// The endpoint is extensible: unknown event types and additional properties are preserved in
// model.SandboxEvent rather than rejected. Cursors are opaque; pass EventsPage.NextCursor back
// unchanged.
//
// When EventsRequest.ETag is set it is sent as If-None-Match. An unchanged page answers 304; Events
// then returns an empty EventsPage with the same ETag and no error, which makes long polling cheap.
func (s *Service) Events(ctx context.Context, sandboxID string, req EventsRequest, opts ...core.RequestOption) (*EventsPage, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	return s.events(ctx, sandboxID, req, opts...)
}

func (s *Service) events(ctx context.Context, sandboxID string, req EventsRequest, opts ...core.RequestOption) (*EventsPage, error) {
	r := &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "events"),
		Query:  eventsQuery(req),
	}
	if req.ETag != "" {
		r.SetHeader("If-None-Match", req.ETag)
	}

	resp, err := s.raw(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified {
		_ = resp.Body.Close()
		etag := resp.Header.Get("ETag")
		if etag == "" {
			etag = req.ETag
		}
		return &EventsPage{Items: []model.SandboxEvent{}, ETag: etag}, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, core.ErrorFromResponse(resp)
	}
	defer resp.Body.Close()

	var out model.EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("agents: decode events response: %w", err)
	}

	page := &EventsPage{
		Items: out.Events,
		ETag:  resp.Header.Get("ETag"),
	}
	if out.PageInfo != nil {
		page.HasMore = out.PageInfo.HasMore
		if out.PageInfo.NextCursor != nil {
			page.NextCursor = *out.PageInfo.NextCursor
		}
	}
	return page, nil
}

func eventsQuery(req EventsRequest) url.Values {
	query := make(url.Values)
	if len(req.Type) > 0 {
		types := make([]string, 0, len(req.Type))
		for _, eventType := range req.Type {
			if eventType != "" {
				types = append(types, eventType)
			}
		}
		if len(types) > 0 {
			query.Set("type", strings.Join(types, ","))
		}
	}
	for _, conversation := range req.Conversation {
		if conversation != "" {
			query.Add("conversation", conversation)
		}
	}
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
