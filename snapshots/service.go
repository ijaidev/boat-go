// Package snapshots wraps Boat snapshots and saved snapshots: listing captured state, browsing
// and downloading its files, deleting it, and managing saved snapshots and their share codes.
package snapshots

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat snapshots API.
type Service struct {
	d core.Doer
}

// New returns a snapshot service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("snapshots: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}

// raw executes a binary request and maps non-2xx responses to *core.APIError.
func (s *Service) raw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	if s == nil || s.d == nil {
		return nil, errors.New("snapshots: service has no transport")
	}
	resp, err := s.d.DoRaw(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return nil, core.ErrorFromResponse(resp)
	}
	return resp, nil
}

func snapshotPath(id string, suffix ...string) string {
	path := "/snapshots/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func sandboxPath(id string, suffix ...string) string {
	path := "/sandboxes/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func savedPath(name string, suffix ...string) string {
	path := "/saved-snapshots/" + url.PathEscape(name)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func requireSnapshotID(id string) error {
	if id == "" {
		return errors.New("snapshots: snapshot id is required")
	}
	return nil
}

func requireSandboxID(id string) error {
	if id == "" {
		return errors.New("snapshots: sandbox id is required")
	}
	return nil
}

func requireName(name string) error {
	if name == "" {
		return errors.New("snapshots: saved snapshot name is required")
	}
	return nil
}
