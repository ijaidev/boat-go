// Package jobs orchestrates workflows that span several Boat services.
//
// It is the only package that imports multiple domain services. Run creates a sandbox, waits for
// it, optionally pushes a local git work tree, runs a command, reads usage and deletes the sandbox.
// Push uploads a git-aware archive and extracts it inside a sandbox. Pull and PullSnapshot stream
// files out of a sandbox or a snapshot.
//
// A command exit code is a normal result, not an error. Failures after sandbox creation still clean
// up the sandbox unless the caller asked to keep it.
package jobs

import (
	"errors"
	"net/http"
	"time"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/sandboxes"
	"github.com/ijaidev/boat-go/snapshots"
)

const defaultCleanupTimeout = 30 * time.Second

// Config wires the services and HTTP client used by jobs.
//
// Sandboxes is required by Run. Commands is required by Run and Push. Files is required by Push,
// Pull and Run with a SourceDir. Snapshots is required by PullSnapshot. HTTPClient is used for
// signed upload and download URLs and defaults to http.DefaultClient.
type Config struct {
	Sandboxes  *sandboxes.Service
	Commands   *commands.Service
	Files      *files.Service
	Snapshots  *snapshots.Service
	HTTPClient core.HTTPClient
}

// Service exposes cross-domain Boat workflows.
type Service struct {
	sandboxes  *sandboxes.Service
	commands   *commands.Service
	files      *files.Service
	snapshots  *snapshots.Service
	httpClient core.HTTPClient
}

// New returns a jobs service. A missing HTTPClient uses http.DefaultClient.
func New(cfg Config) *Service {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{
		sandboxes:  cfg.Sandboxes,
		commands:   cfg.Commands,
		files:      cfg.Files,
		snapshots:  cfg.Snapshots,
		httpClient: httpClient,
	}
}

func (s *Service) requireRun(sourceDir string) error {
	if s == nil {
		return errors.New("jobs: service is nil")
	}
	if s.sandboxes == nil {
		return errors.New("jobs: sandboxes service is required")
	}
	if s.commands == nil {
		return errors.New("jobs: commands service is required")
	}
	if sourceDir != "" && s.files == nil {
		return errors.New("jobs: files service is required to push a source directory")
	}
	return nil
}

func (s *Service) requirePush() error {
	if s == nil {
		return errors.New("jobs: service is nil")
	}
	if s.commands == nil {
		return errors.New("jobs: commands service is required")
	}
	if s.files == nil {
		return errors.New("jobs: files service is required")
	}
	return nil
}

func (s *Service) requireFiles() error {
	if s == nil {
		return errors.New("jobs: service is nil")
	}
	if s.files == nil {
		return errors.New("jobs: files service is required")
	}
	return nil
}

func (s *Service) requireSnapshots() error {
	if s == nil {
		return errors.New("jobs: service is nil")
	}
	if s.snapshots == nil {
		return errors.New("jobs: snapshots service is required")
	}
	return nil
}
