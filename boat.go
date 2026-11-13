package boat

import (
	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/agents"
	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/transport"
	"github.com/ijaidev/boat-go/sandboxes"
	"github.com/ijaidev/boat-go/snapshots"
	"github.com/ijaidev/boat-go/telemetry"
)

// Version is the SDK version reported in the User-Agent. It is not the Boat API version.
const Version = "0.1.0-dev"

// Config controls the client. It is an alias of core.Config.
//
// BaseURL defaults to https://boat.dev/api/v1. APIKey or TokenSource is required. MaxRetries 0
// uses the default of three retries; a negative value disables retries. RetryMin and RetryMax
// default to 200ms and 5s.
type Config = core.Config

// Client is the root Boat SDK client. Its fields are concrete domain services.
type Client struct {
	Account      *account.Service
	Agents       *agents.Service
	CI           *ci.Service
	Commands     *commands.Service
	Environments *environments.Service
	Files        *files.Service
	Sandboxes    *sandboxes.Service
	Snapshots    *snapshots.Service
	Telemetry    *telemetry.Service
}

// New builds a client with one shared transport.
func New(cfg Config) (*Client, error) {
	if cfg.UserAgent == "" {
		cfg.UserAgent = "boat-go/" + Version
	}
	transportClient, err := transport.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		Account:      account.New(transportClient),
		Agents:       agents.New(transportClient),
		CI:           ci.New(transportClient),
		Commands:     commands.New(transportClient),
		Environments: environments.New(transportClient),
		Files:        files.New(transportClient),
		Sandboxes:    sandboxes.New(transportClient),
		Snapshots:    snapshots.New(transportClient),
		Telemetry:    telemetry.New(transportClient),
	}, nil
}
