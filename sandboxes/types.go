package sandboxes

import (
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// CreateRequest is the body of POST /sandboxes.
//
// TTLSeconds has three states: leave it nil for the server default, use core.Some(n) for an explicit
// auto-stop, or use core.Null[int]() to disable auto-stop.
type CreateRequest struct {
	Type        model.SandboxType                    `json:"type,omitempty"`
	TTLSeconds  *core.Optional[int]                  `json:"ttlSeconds,omitempty"`
	Labels      map[string]string                    `json:"labels,omitempty"`
	Env         map[string]string                    `json:"env,omitempty"`
	Environment string                               `json:"environment,omitempty"`
	NoEnv       *bool                                `json:"noEnv,omitempty"`
	Snapshots   *bool                                `json:"snapshots,omitempty"`
	Monitoring  *bool                                `json:"monitoring,omitempty"`
	FailFast    *bool                                `json:"failFast,omitempty"`
	Count       int                                  `json:"count,omitempty"`
	SetupScript string                               `json:"setupScript,omitempty"`
	Name        string                               `json:"name,omitempty"`
	Visibility  model.CreateSandboxRequestVisibility `json:"visibility,omitempty"`
	HostedPorts []model.HostPortRequest              `json:"hostedPorts,omitempty"`
	Files       []CreateFile                         `json:"files,omitempty"`
	Org         string                               `json:"org,omitempty"`
	TeamID      string                               `json:"teamId,omitempty"`
	From        string                               `json:"from,omitempty"`
}

// CreateFile is a file written before the sandbox reports ready.
type CreateFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"`
}

// ResumeRequest is the body of POST /sandboxes/{id}/resume.
type ResumeRequest struct {
	Type        model.SandboxType   `json:"type,omitempty"`
	TTLSeconds  *core.Optional[int] `json:"ttlSeconds,omitempty"`
	Env         map[string]string   `json:"env,omitempty"`
	Environment string              `json:"environment,omitempty"`
	NoEnv       *bool               `json:"noEnv,omitempty"`
	FailFast    *bool               `json:"failFast,omitempty"`
}

// ForkRequest is the body of POST /sandboxes/{id}/fork.
type ForkRequest struct {
	Labels      map[string]string   `json:"labels,omitempty"`
	FailFast    *bool               `json:"failFast,omitempty"`
	Monitoring  *bool               `json:"monitoring,omitempty"`
	Env         map[string]string   `json:"env,omitempty"`
	Environment string              `json:"environment,omitempty"`
	NoEnv       *bool               `json:"noEnv,omitempty"`
	Type        model.SandboxType   `json:"type,omitempty"`
	TTLSeconds  *core.Optional[int] `json:"ttlSeconds,omitempty"`
}

// UpdateRequest is the body of PATCH /sandboxes/{id}.
type UpdateRequest struct {
	Name       string              `json:"name,omitempty"`
	TTLSeconds *core.Optional[int] `json:"ttlSeconds,omitempty"`
	Subdomain  string              `json:"subdomain,omitempty"`
}

// ListRequest filters GET /sandboxes.
type ListRequest struct {
	Limit         int
	Cursor        string
	Sort          string
	State         []string
	Name          string
	NamePrefix    string
	StoppedBefore *time.Time
	DataExpired   *bool
	Label         []string
	Include       []string
}

// StopOptions is the body of POST /sandboxes/{id}/stop.
type StopOptions struct {
	Force *bool `json:"force,omitempty"`
}

// UsageRequest filters GET /sandboxes/{id}/usage.
type UsageRequest struct {
	Since *time.Time
	Until *time.Time
}

// ShareResult is returned by Share.
type ShareResult struct {
	Sandbox         *model.Sandbox
	RestartRequired bool
	Message         string
}

// UnshareResult is returned by Unshare.
type UnshareResult struct {
	Sandbox *model.Sandbox
	Message string
}

// RecoverResult is returned by Recover.
type RecoverResult struct {
	Message string
}
