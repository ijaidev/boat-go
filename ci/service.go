// Package ci wraps the Boat CI endpoints for GitHub Actions jobs that run on Boat.
//
// It reads the CI overview and single jobs, fetches a job's GitHub log, returns the GitHub App
// install or authorization link, and sets which public repositories may run jobs. Jobs are read-only
// views: boat starts the runner, GitHub owns the workflow.
package ci

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

var errNoTransport = errors.New("ci: service has no transport")

// Service is the Boat CI API.
type Service struct {
	d core.Doer
}

// New returns a CI service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errNoTransport
	}
	return s.d.Do(ctx, req, out, opts...)
}

func jobPath(jobID string, suffix ...string) string {
	path := "/ci/jobs/" + url.PathEscape(jobID)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func requireJobID(jobID string) error {
	if jobID == "" {
		return errors.New("ci: job id is required")
	}
	return nil
}
