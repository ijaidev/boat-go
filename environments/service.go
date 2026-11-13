// Package environments wraps the Boat sandbox environment and repository APIs.
//
// An environment is a reusable start configuration: repositories, environment variables, secret
// files, setup behavior and credential toggles. Editing an environment mints a new immutable
// version; sandboxes stay pinned to the version they started with until Upgrade moves them.
//
// Environment routes select an organization with the org query parameter, not the X-Boat-Org
// header. Set the Org field on a request type or pass WithOrg. The /repos routes have no org
// parameter; they operate on the default environment.
//
// Environment variable values and secret file contents are secrets. The SDK never logs them and
// does not add them to error text.
package environments

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
)

var errNoTransport = errors.New("environments: service has no transport")

// Service is the Boat sandbox environments and repositories API.
type Service struct {
	d core.Doer
}

// New returns an environments service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errNoTransport
	}
	return s.d.Do(ctx, req, out, opts...)
}

// WithOrg scopes an environment request to an organization through the org query parameter.
//
// Use it for methods that do not take an Org field, such as Delete and SetVar. The environment
// routes only honor ?org=; core.WithOrg sets X-Boat-Org, which those routes ignore.
func WithOrg(org string) core.RequestOption {
	return core.WithQuery("org", org)
}

func environmentPath(id string, suffix ...string) string {
	path := "/environments/" + url.PathEscape(id)
	for _, part := range suffix {
		path += "/" + url.PathEscape(part)
	}
	return path
}

func orgQuery(org string) url.Values {
	query := make(url.Values)
	if org != "" {
		query.Set("org", org)
	}
	return query
}

func requireEnvironmentID(id string) error {
	if id == "" {
		return errors.New("environments: environment id is required")
	}
	return nil
}

func requireName(name string) error {
	if name == "" {
		return errors.New("environments: environment name is required")
	}
	return nil
}

func requireVarKey(key string) error {
	if key == "" {
		return errors.New("environments: variable key is required")
	}
	return nil
}

func requirePath(path string) error {
	if path == "" {
		return errors.New("environments: secret file path is required")
	}
	return nil
}

func requireRepoID(id string) error {
	if id == "" {
		return errors.New("environments: repository id is required")
	}
	return nil
}
