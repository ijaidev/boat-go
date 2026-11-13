package files

import (
	"context"
	"errors"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// MkdirRequest is the body of POST /sandboxes/{sandboxId}/files/folder.
//
// Path is required; Mode is optional octal permission bits, for example "755".
type MkdirRequest struct {
	Path string  `json:"path"`
	Mode *string `json:"mode,omitempty"`
}

// Mkdir creates one directory. Missing parent directories are created, and an existing directory is
// not an error.
func (s *Service) Mkdir(ctx context.Context, sandboxID string, req MkdirRequest, opts ...core.RequestOption) (*model.FileInfoResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}

	var out model.FileInfoResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "folder"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// MoveRequest is the body of POST /sandboxes/{sandboxId}/files/move.
//
// Source and Destination are required. Overwrite is optional and defaults to false, which makes an
// existing destination an error.
type MoveRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Overwrite   *bool  `json:"overwrite,omitempty"`
}

// Move renames one file, symlink or directory. It works across /home/user and /tmp.
//
// A symlink is moved itself, never its target. Missing parent directories of the destination are
// created.
func (s *Service) Move(ctx context.Context, sandboxID string, req MoveRequest, opts ...core.RequestOption) (*model.FileMoveResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if req.Source == "" {
		return nil, errors.New("files: source path is required")
	}
	if req.Destination == "" {
		return nil, errors.New("files: destination path is required")
	}

	var out model.FileMoveResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "move"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChmodRequest is the body of POST /sandboxes/{sandboxId}/files/permissions.
//
// Path and Mode are required; Mode holds octal permission bits, for example "755".
type ChmodRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// Chmod sets the permission bits of one path. Ownership and group are unchanged.
func (s *Service) Chmod(ctx context.Context, sandboxID string, req ChmodRequest, opts ...core.RequestOption) (*model.FileInfoResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}
	if req.Mode == "" {
		return nil, errors.New("files: mode is required")
	}

	var out model.FileInfoResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "permissions"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplaceRequest is the body of POST /sandboxes/{sandboxId}/files/replace.
//
// Paths holds 1 to 100 files of up to 5 MiB each, Pattern is plain text and NewValue is required;
// use "" to delete the matches.
type ReplaceRequest struct {
	Paths    []string `json:"paths"`
	Pattern  string   `json:"pattern"`
	NewValue string   `json:"newValue"`
}

// Replace replaces every occurrence of req.Pattern with req.NewValue in each file.
//
// One result is returned per file. A file that fails does not stop the others; inspect
// FileReplaceResponse.Results and the per-file Error.
func (s *Service) Replace(ctx context.Context, sandboxID string, req ReplaceRequest, opts ...core.RequestOption) (*model.FileReplaceResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if len(req.Paths) == 0 {
		return nil, errors.New("files: at least one path is required")
	}

	var out model.FileReplaceResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "replace"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
