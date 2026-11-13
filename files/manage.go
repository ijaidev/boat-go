package files

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

var errPatternRequired = errors.New("files: pattern is required")

// DeleteOptions controls Delete.
type DeleteOptions struct {
	// Recursive removes a non-empty directory. A symlink is always removed itself, not its target.
	Recursive bool
}

// Delete removes one file, symlink or directory and returns its canonical path.
//
// A directory needs DeleteOptions.Recursive. /home/user and /tmp themselves cannot be deleted.
func (s *Service) Delete(ctx context.Context, sandboxID, path string, options DeleteOptions, opts ...core.RequestOption) (*model.FileDeleteResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(path); err != nil {
		return nil, err
	}
	query := url.Values{"path": {path}}
	if options.Recursive {
		query.Set("recursive", "true")
	}

	var out model.FileDeleteResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodDelete,
		Path:   sandboxPath(sandboxID, "files"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRequest filters List.
type ListRequest struct {
	// Path is the directory to list. Empty uses /home/user.
	Path string
}

// List returns one directory level, sorted by name.
//
// A symlink is listed as a symlink and is not followed. Truncated is true when the directory holds
// more than 10,000 entries, which is the API's cap; there is no pagination.
func (s *Service) List(ctx context.Context, sandboxID string, req ListRequest, opts ...core.RequestOption) (*model.FileListResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	query := make(url.Values)
	if req.Path != "" {
		query.Set("path", req.Path)
	}

	var out model.FileListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "files", "list"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Info returns the size, permission bits, type and modification time of one path.
func (s *Service) Info(ctx context.Context, sandboxID, path string, opts ...core.RequestOption) (*model.FileInfoResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(path); err != nil {
		return nil, err
	}

	var out model.FileInfoResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "files", "info"),
		Query:  url.Values{"path": {path}},
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchRequest filters Search.
type SearchRequest struct {
	// Path is the directory to walk. Empty uses /home/user.
	Path string
	// Pattern is a glob matched against each file and directory name, for example "*.py".
	Pattern string
}

// Search returns every path under req.Path whose name matches the glob req.Pattern.
//
// Symlinks are not followed. Files holds absolute paths, at most 10,000. Errors is present only when
// part of the tree could not be searched; the results are then incomplete.
func (s *Service) Search(ctx context.Context, sandboxID string, req SearchRequest, opts ...core.RequestOption) (*model.FileSearchResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if req.Pattern == "" {
		return nil, errPatternRequired
	}
	query := make(url.Values)
	if req.Path != "" {
		query.Set("path", req.Path)
	}
	query.Set("pattern", req.Pattern)

	var out model.FileSearchResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "files", "search"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// FindRequest filters Find.
type FindRequest struct {
	// Path is the directory to walk. Empty uses /home/user.
	Path string
	// Pattern is plain text, or an extended regular expression when Regex is true.
	Pattern string
	// Regex treats Pattern as an extended regular expression.
	Regex bool
}

// Find returns every line under req.Path that contains req.Pattern.
//
// Binary files are skipped and symlinks are not followed. Matches holds at most 1,000 lines, each cut
// at 500 characters. Errors is present only when part of the tree could not be searched; the matches
// are then incomplete.
func (s *Service) Find(ctx context.Context, sandboxID string, req FindRequest, opts ...core.RequestOption) (*model.FileFindResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if req.Pattern == "" {
		return nil, errPatternRequired
	}
	query := make(url.Values)
	if req.Path != "" {
		query.Set("path", req.Path)
	}
	query.Set("pattern", req.Pattern)
	if req.Regex {
		query.Set("regex", "true")
	}

	var out model.FileFindResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "files", "find"),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
