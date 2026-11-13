package files

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// URLRequest is the body of the signed upload and download URL endpoints.
//
// Path is required and must resolve under /home/user or /tmp. TTLSeconds is optional: nil uses the
// server default of 15 minutes, and the maximum is one hour. Use core.Ptr to set it.
type URLRequest struct {
	Path       string `json:"path"`
	TTLSeconds *int   `json:"ttlSeconds,omitempty"`
}

// UploadURL returns a short-lived signed URL for writing one file with no size limit.
//
// PUT the raw file body to the returned URL; it needs no Authorization header and may be sent by any
// HTTP client. Missing directories are created, and an existing file is replaced only once the upload
// completes. Use this for files over the 5 MiB inline write cap.
//
// The URL and its token are secrets: do not log them, store them, or include them in error text.
// The URL expires at FileURLResponse.ExpiresAt.
func (s *Service) UploadURL(ctx context.Context, sandboxID string, req URLRequest, opts ...core.RequestOption) (*model.FileURLResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}

	var out model.FileURLResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "upload-url"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadURL returns a short-lived signed URL for reading one file with no size limit.
//
// GET the returned URL; it needs no Authorization header and may be sent by any HTTP client. Use
// this for files over the 5 MiB inline read cap, or for streaming.
//
// The URL and its token are secrets: do not log them, store them, or include them in error text.
// The URL expires at FileURLResponse.ExpiresAt.
func (s *Service) DownloadURL(ctx context.Context, sandboxID string, req URLRequest, opts ...core.RequestOption) (*model.FileURLResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}

	var out model.FileURLResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "files", "download-url"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
