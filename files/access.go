package files

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// HostRequest is the body of POST /sandboxes/{sandboxId}/host.
//
// Port is required and must be 1-65535. Public requests an ungated URL instead of one that needs the
// _token query parameter. Localhost marks a service that listens on 127.0.0.1 or ::1 only; leave it
// unset for a service bound to 0.0.0.0. Title is an optional display title.
type HostRequest struct {
	Port      int    `json:"port"`
	Public    *bool  `json:"public,omitempty"`
	Localhost *bool  `json:"localhost,omitempty"`
	Title     string `json:"title,omitempty"`
}

// Host registers a stable public HTTPS route for a port inside the sandbox and opens the firewall.
//
// It is idempotent per sandbox and port. The returned URL carries a _token query parameter unless
// req.Public is true; treat a token-gated URL as a secret and never log it.
func (s *Service) Host(ctx context.Context, sandboxID string, req HostRequest, opts ...core.RequestOption) (*model.HostPortResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if req.Port < 1 || req.Port > 65535 {
		return nil, fmt.Errorf("files: port %d is outside 1-65535", req.Port)
	}

	var out model.HostPortResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "host"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// DesktopTheme is the color theme of the non-VNC desktop stream.
type DesktopTheme string

const (
	// DesktopThemeLight uses the light desktop theme.
	DesktopThemeLight DesktopTheme = "light"
	// DesktopThemeDark uses the dark desktop theme.
	DesktopThemeDark DesktopTheme = "dark"
)

// DesktopRequest selects the desktop stream mode and options.
//
// VNC requests noVNC mode through ?vnc=1. Theme applies only to non-VNC mode. PublicAccess asks for
// a noVNC URL without an access token; it has no effect without VNC.
type DesktopRequest struct {
	VNC          bool
	Theme        DesktopTheme
	PublicAccess *bool
}

type desktopBody struct {
	PublicAccess *bool `json:"publicAccess,omitempty"`
}

// Desktop returns a secret-bearing desktop or noVNC URL for live computer-use visibility.
//
// A response with Provisioning true means VNC is still being prepared; poll again. The desktop URL
// carries an access token unless PublicAccess is true. Treat it as a secret and never log it.
func (s *Service) Desktop(ctx context.Context, sandboxID string, req DesktopRequest, opts ...core.RequestOption) (*model.DesktopResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	switch req.Theme {
	case "", DesktopThemeLight, DesktopThemeDark:
	default:
		return nil, fmt.Errorf("files: unsupported desktop theme %q", req.Theme)
	}

	query := make(url.Values)
	if req.VNC {
		query.Set("vnc", "1")
	}
	if req.Theme != "" {
		query.Set("theme", string(req.Theme))
	}

	var body any
	if req.PublicAccess != nil {
		body = desktopBody{PublicAccess: req.PublicAccess}
	}

	var out model.DesktopResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "desktop"),
		Query:  query,
		JSON:   body,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// SSHKeyRequest is the body of POST /sandboxes/{sandboxId}/sshkey.
//
// Key is a public SSH key in OpenSSH format. Private keys are rejected with 400.
type SSHKeyRequest struct {
	Key string `json:"key"`
}

// SSHKey adds a public key so the caller can SSH to the sandbox as user "user".
//
// The response carries MachineIP, the host to connect to on port 22, and SSHEndpoint when the machine
// has no public IPv4 of its own.
func (s *Service) SSHKey(ctx context.Context, sandboxID string, req SSHKeyRequest, opts ...core.RequestOption) (*model.SSHKeyResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if req.Key == "" {
		return nil, errors.New("files: ssh key is required")
	}

	var out model.SSHKeyResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "sshkey"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
