// Package files wraps the Boat sandbox filesystem and sandbox access API.
//
// It covers inline file reads and writes, directory listing, search, find, folders, moves,
// permissions, text replacement, signed upload and download URLs, binary artifact downloads, and
// sandbox access through hosted ports, the desktop and SSH.
//
// Paths are relative to the sandbox work directory /home/user unless they start with a slash. An
// absolute path must resolve under /home/user or /tmp. Inline reads and writes are capped at 5 MiB:
// a larger file returns a 413 file_too_large and must use UploadURL or DownloadURL.
//
// Signed upload and download URLs, desktop URLs and token-gated host URLs are secrets. Do not log
// them and do not put them in error text.
package files

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

var errNoTransport = errors.New("files: service has no transport")

// Service is the Boat sandbox files and access API.
type Service struct {
	d core.Doer
}

// New returns a files service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errNoTransport
	}
	return s.d.Do(ctx, req, out, opts...)
}

func (s *Service) raw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	if s == nil || s.d == nil {
		return nil, errNoTransport
	}
	return s.d.DoRaw(ctx, req, opts...)
}

func sandboxPath(id string, suffix ...string) string {
	path := "/sandboxes/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func requireID(id string) error {
	if id == "" {
		return errors.New("files: sandbox id is required")
	}
	return nil
}

func requirePath(path string) error {
	if path == "" {
		return errors.New("files: path is required")
	}
	return nil
}
