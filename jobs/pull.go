package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
)

// Pull downloads one sandbox file through a signed download URL and streams it to w.
//
// It uses files.DownloadURL, so the file has no inline 5 MiB limit. The response is copied
// incrementally and is never buffered. w must be non-nil. The signed URL is never included in an
// error. Request options apply to the download-URL call.
func (s *Service) Pull(ctx context.Context, sandboxID, remotePath string, w io.Writer, opts ...core.RequestOption) error {
	if w == nil {
		return errors.New("jobs: writer is required")
	}
	if err := s.requireFiles(); err != nil {
		return err
	}

	signed, err := s.files.DownloadURL(ctx, sandboxID, files.URLRequest{Path: remotePath}, opts...)
	if err != nil {
		return fmt.Errorf("jobs: get download URL: %w", err)
	}

	operation := fmt.Sprintf("pull %q", remotePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed.URL, nil)
	if err != nil {
		return fmt.Errorf("jobs: %s: invalid signed download URL", operation)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return requestFailed(operation, err)
	}
	if resp.Body == nil {
		return fmt.Errorf("jobs: %s: response has no body", operation)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("jobs: %s: http %d", operation, resp.StatusCode)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("jobs: %s: %w", operation, err)
	}
	return nil
}

// PullSnapshot streams one file, folder or whole snapshot to w without contacting the sandbox.
//
// The snapshot body is closed exactly once. An empty path downloads the whole snapshot as a tar.
// w must be non-nil. Request options apply to the snapshot request.
func (s *Service) PullSnapshot(ctx context.Context, snapshotID, path string, w io.Writer, opts ...core.RequestOption) error {
	if w == nil {
		return errors.New("jobs: writer is required")
	}
	if err := s.requireSnapshots(); err != nil {
		return err
	}

	download, err := s.snapshots.File(ctx, snapshotID, path, opts...)
	if err != nil {
		return fmt.Errorf("jobs: pull snapshot: %w", err)
	}
	if download.Body == nil {
		return errors.New("jobs: snapshot response has no body")
	}
	defer download.Body.Close()

	if _, err := io.Copy(w, download.Body); err != nil {
		return fmt.Errorf("jobs: pull snapshot: %w", err)
	}
	return nil
}
