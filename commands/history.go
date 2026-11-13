package commands

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// HistoryRequest filters GET /sandboxes/{sandboxId}/command-history.
//
// Mode is a comma-separated list of sync, stream, detached, process or commands (the first three).
// Query matches the command line, case-insensitively. Zero values use the server defaults.
type HistoryRequest struct {
	Mode   string
	Query  string
	Limit  int
	Cursor string
}

// LogsRequest selects one read of a command's stored output.
//
// Stream defaults to stdout, Offset to 0, Limit to 1 MiB and Encoding to utf8. Use Offset and
// NextOffset to page through output.
type LogsRequest struct {
	Stream   model.CommandLogsParamsStream
	Offset   int
	Limit    int
	Encoding model.CommandLogsParamsEncoding
}

// History returns one page of command and process history, newest first.
func (s *Service) History(ctx context.Context, sandboxID string, req HistoryRequest, opts ...core.RequestOption) (core.Page[model.CommandRecord], error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return core.Page[model.CommandRecord]{}, err
	}
	query := make(url.Values)
	if req.Mode != "" {
		query.Set("mode", req.Mode)
	}
	if req.Query != "" {
		query.Set("q", req.Query)
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var out model.CommandListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "command-history"),
		Query:  query,
	}, &out, opts...); err != nil {
		return core.Page[model.CommandRecord]{}, err
	}

	page := core.Page[model.CommandRecord]{Items: out.Commands}
	if out.NextCursor != nil && *out.NextCursor != "" {
		page.NextCursor = *out.NextCursor
		page.HasMore = true
	}
	return page, nil
}

// GetHistory returns one command record from the history.
func (s *Service) GetHistory(ctx context.Context, sandboxID, commandID string, opts ...core.RequestOption) (*model.CommandRecord, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireCommandID(commandID); err != nil {
		return nil, err
	}
	var out model.CommandInfoResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   historyPath(sandboxID, commandID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Command, nil
}

// Logs reads one chunk of a command's stored output.
//
// It works while the sandbox is stopped. A running command gains output as the sandbox copies it, so
// read with NextOffset until Done.
func (s *Service) Logs(ctx context.Context, sandboxID, commandID string, req LogsRequest, opts ...core.RequestOption) (*model.CommandLogsResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireCommandID(commandID); err != nil {
		return nil, err
	}
	if req.Offset < 0 {
		return nil, errors.New("commands: log offset must not be negative")
	}
	if req.Limit < 0 {
		return nil, errors.New("commands: log limit must not be negative")
	}

	query := make(url.Values)
	if req.Stream != "" {
		query.Set("stream", string(req.Stream))
	}
	if req.Offset > 0 {
		query.Set("offset", strconv.Itoa(req.Offset))
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Encoding != "" {
		query.Set("encoding", string(req.Encoding))
	}

	var out model.CommandLogsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   historyPath(sandboxID, commandID, "logs"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
