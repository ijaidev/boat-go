// Package credentials wraps Boat AI credential profiles, credential accounts, custom APIs, the AI
// credentials proxy and AI usage.
//
// A credential profile is a named set of AI credentials. Sandboxes use a profile to reach model
// providers without copying keys into the sandbox. API keys, subscription tokens and custom API
// values are secrets: they are write-only, are never returned by the API and are never included in
// SDK error text.
//
// The AI credentials proxy keeps real provider keys out of sandboxes. With the proxy on, a sandbox
// receives placeholder keys and base URLs that point at Boat, and Boat substitutes the real key for
// each request. Proxy settings and sandbox AI access are security-sensitive: a change affects every
// sandbox that uses the profile or sandbox.
package credentials

import (
	"context"
	"errors"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

var errNoTransport = errors.New("credentials: service has no transport")

// Service is the Boat AI credentials API.
type Service struct {
	d core.Doer
}

// New returns a credentials service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errNoTransport
	}
	return s.d.Do(ctx, req, out, opts...)
}

func profilePath(profile string, suffix ...string) string {
	path := "/credential-profiles/" + url.PathEscape(profile)
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

func requireProfile(profile string) error {
	if profile == "" {
		return errors.New("credentials: profile is required")
	}
	return nil
}

func requireProfileName(name string) error {
	if name == "" {
		return errors.New("credentials: profile name is required")
	}
	return nil
}

func requireCredential(credential model.CredentialID) error {
	if credential == "" {
		return errors.New("credentials: credential is required")
	}
	return nil
}

func requireAccount(account string) error {
	if account == "" {
		return errors.New("credentials: account is required")
	}
	return nil
}

func requireKeyValue(value string) error {
	if value == "" {
		return errors.New("credentials: key value is required")
	}
	return nil
}

func requireCustomAPI(name string) error {
	if name == "" {
		return errors.New("credentials: custom API name is required")
	}
	return nil
}

func requireCustomAPIValue(value string) error {
	if value == "" {
		return errors.New("credentials: custom API value is required")
	}
	return nil
}

func requireCustomAPIBaseURL(value string) error {
	if value == "" {
		return errors.New("credentials: custom API base URL is required")
	}
	return nil
}

func requireSandboxID(id string) error {
	if id == "" {
		return errors.New("credentials: sandbox id is required")
	}
	return nil
}
