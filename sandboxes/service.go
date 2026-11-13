// Package sandboxes wraps the Boat sandbox lifecycle: create, read, update, stop, resume, fork,
// share and delete.
package sandboxes

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat sandboxes API.
type Service struct {
	d core.Doer
}

// New returns a sandbox service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("sandboxes: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
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
		return errors.New("sandboxes: sandbox id is required")
	}
	return nil
}
