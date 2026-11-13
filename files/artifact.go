package files

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Download is one raw binary response returned by Artifact.
//
// Body is the unread response stream. The caller owns it and must close it. ContentLength is -1 when
// the server did not send a Content-Length.
type Download struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

// Artifact downloads one file as raw bytes and returns the unread body.
//
// The file may be up to 50 MiB and must resolve under /home/user or /tmp. The response is not
// buffered: read Body and close it when done. Non-2xx responses are mapped to *core.APIError and the
// body is already closed.
func (s *Service) Artifact(ctx context.Context, sandboxID, path string, opts ...core.RequestOption) (*Download, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(path); err != nil {
		return nil, err
	}

	req := &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "artifacts"),
		Query:  url.Values{"path": {path}},
	}
	req.SetHeader("Accept", "application/octet-stream")

	resp, err := s.raw(ctx, req, opts...)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, core.ErrorFromResponse(resp)
	}
	return &Download{
		Body:          resp.Body,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
	}, nil
}
