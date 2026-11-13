// Package commands runs commands in Boat sandboxes and reads their status, output, history and
// processes.
package commands

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat commands API.
type Service struct {
	d core.Doer
}

// New returns a commands service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("commands: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}

func (s *Service) raw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	if s == nil || s.d == nil {
		return nil, errors.New("commands: service has no transport")
	}
	return s.d.DoRaw(ctx, req, opts...)
}

func sandboxPath(id string, parts ...string) string {
	path := "/sandboxes/" + url.PathEscape(id)
	for _, part := range parts {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func commandPath(sandboxID string, processID int, parts ...string) string {
	all := append([]string{"commands", strconv.Itoa(processID)}, parts...)
	return sandboxPath(sandboxID, all...)
}

func historyPath(sandboxID, commandID string, parts ...string) string {
	all := append([]string{"command-history", commandID}, parts...)
	return sandboxPath(sandboxID, all...)
}

func requireSandboxID(id string) error {
	if id == "" {
		return errors.New("commands: sandbox id is required")
	}
	return nil
}

func requireProcessID(id int) error {
	if id <= 0 {
		return errors.New("commands: process id must be positive")
	}
	return nil
}

func requireCommandID(id string) error {
	if id == "" {
		return errors.New("commands: command id is required")
	}
	return nil
}
