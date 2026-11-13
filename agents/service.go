// Package agents wraps the Boat integrated agents API: prompts, prompt status, sandbox events,
// conversations, steering and interruption.
package agents

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat integrated agents API.
type Service struct {
	d core.Doer
}

// New returns an agents service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("agents: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}

func (s *Service) raw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	if s == nil || s.d == nil {
		return nil, errors.New("agents: service has no transport")
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

func require(value, name string) error {
	if value == "" {
		return errors.New("agents: " + name + " is required")
	}
	return nil
}
