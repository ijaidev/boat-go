package snapshots

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Download is an open binary snapshot response.
//
// The caller owns Body and must close it. ContentLength is -1 when the server did not send a
// Content-Length header. A folder or whole-snapshot response has ContentType application/x-tar; a
// single file has application/octet-stream.
type Download struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

// File streams one file or folder out of a snapshot without contacting the sandbox.
//
// An empty path downloads the whole snapshot as a tar; Download is the named shorthand for that.
// A folder answers with a tar archive. Range requests are supported: pass
// core.WithHeader("Range", "bytes=0-1023") to resume a large file. The caller must close
// Download.Body.
func (s *Service) File(ctx context.Context, snapshotID, path string, opts ...core.RequestOption) (*Download, error) {
	if err := requireSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	query := make(url.Values)
	if path != "" {
		query.Set("path", path)
	}
	req := &core.Request{
		Method: http.MethodGet,
		Path:   snapshotPath(snapshotID, "files"),
		Query:  query,
	}
	// The transport defaults Accept to application/json, which is wrong for a binary response.
	req.SetHeader("Accept", "*/*")

	resp, err := s.raw(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	return &Download{
		Body:          resp.Body,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
	}, nil
}

// Download streams the whole snapshot as a tar archive.
//
// It is File with an empty path. Use DownloadManifest when you need the signed chunk URLs instead
// of the restored files.
func (s *Service) Download(ctx context.Context, snapshotID string, opts ...core.RequestOption) (*Download, error) {
	return s.File(ctx, snapshotID, "", opts...)
}

// DownloadManifest returns the time-limited signed URLs for every chunk needed to reconstruct the
// snapshot filesystem.
//
// The signed URLs and the inventory URL are secret-bearing: never log or persist them. A snapshot
// whose older layers were cleaned up answers 409 snapshot_compacted; read its files with File
// instead.
func (s *Service) DownloadManifest(ctx context.Context, snapshotID string, opts ...core.RequestOption) (*model.SnapshotDownloadResponse, error) {
	if err := requireSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	var out model.SnapshotDownloadResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   snapshotPath(snapshotID, "download"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
