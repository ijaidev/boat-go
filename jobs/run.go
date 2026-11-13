package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/sandboxes"
)

// RunRequest describes one create, push, exec, usage and delete workflow.
//
// SourceDir is optional. When set, Push uploads the local git work tree before the command runs.
// RemoteDir is the sandbox directory to extract into and defaults to ".". Stdout and Stderr may be
// nil. Keep keeps the sandbox on both success and failure. KeepOnError keeps it only when a step
// after creation fails.
type RunRequest struct {
	Create      sandboxes.CreateRequest
	Command     string
	Cwd         string
	SourceDir   string
	RemoteDir   string
	Stdout      io.Writer
	Stderr      io.Writer
	Keep        bool
	KeepOnError bool
}

// RunResult is the outcome of Run.
//
// SandboxID is set as soon as the sandbox was created, including when a later step fails. ExitCode
// and Signal come from the command's exit event. Usage is nil when the best-effort usage request
// failed; UsageError then carries that failure without failing the run. Pushed reports whether the
// source archive was uploaded and extracted.
type RunResult struct {
	SandboxID  string
	ExitCode   int
	Signal     string
	Usage      *model.SandboxUsageResponse
	Pushed     bool
	UsageError error
}

// Run creates a sandbox, waits until it is ready, optionally pushes SourceDir, runs Command, reads
// usage and deletes the sandbox.
//
// A non-zero command exit code is returned in RunResult.ExitCode, not as an error. A command stream
// failure is an error. Usage is best effort and never fails the run. On failure the sandbox is
// deleted unless Keep or KeepOnError is set; on success it is deleted unless Keep is set. Cleanup
// uses a fresh bounded context, so a canceled caller still cleans up, and a cleanup failure is
// joined with the primary error.
func (s *Service) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	if strings.TrimSpace(req.Command) == "" {
		return nil, errors.New("jobs: command is required")
	}
	if err := s.requireRun(req.SourceDir); err != nil {
		return nil, err
	}

	sandbox, err := s.sandboxes.Create(ctx, req.Create)
	if err != nil {
		return nil, fmt.Errorf("jobs: create sandbox: %w", err)
	}

	result := &RunResult{SandboxID: sandbox.ID}
	runErr := s.runOnSandbox(ctx, sandbox.ID, req, result)
	if runErr != nil {
		if req.Keep || req.KeepOnError {
			return result, runErr
		}
		if cleanupErr := s.cleanupSandbox(ctx, sandbox.ID); cleanupErr != nil {
			return result, errors.Join(runErr, cleanupErr)
		}
		return result, runErr
	}
	if req.Keep {
		return result, nil
	}
	if cleanupErr := s.cleanupSandbox(ctx, sandbox.ID); cleanupErr != nil {
		return result, cleanupErr
	}
	return result, nil
}

func (s *Service) runOnSandbox(ctx context.Context, sandboxID string, req RunRequest, result *RunResult) error {
	if _, err := s.sandboxes.WaitUntilReady(ctx, sandboxID, sandboxes.WaitOptions{}); err != nil {
		return fmt.Errorf("jobs: wait for sandbox %s: %w", sandboxID, err)
	}
	if req.SourceDir != "" {
		if err := s.Push(ctx, sandboxID, req.SourceDir, req.RemoteDir); err != nil {
			return err
		}
		result.Pushed = true
	}
	if err := s.runCommand(ctx, sandboxID, req, result); err != nil {
		return err
	}

	usage, err := s.sandboxes.Usage(ctx, sandboxID, sandboxes.UsageRequest{})
	result.Usage = usage
	result.UsageError = err
	return nil
}

func (s *Service) runCommand(ctx context.Context, sandboxID string, req RunRequest, result *RunResult) error {
	commandReq := commands.RunRequest{Command: req.Command}
	if req.Cwd != "" {
		commandReq.Cwd = core.Ptr(req.Cwd)
	}

	stream, err := s.commands.RunCommand(ctx, sandboxID, commandReq)
	if err != nil {
		return fmt.Errorf("jobs: run command: %w", err)
	}
	defer stream.Close()

	exited := false
	for {
		event, err := stream.Next(ctx)
		if errors.Is(err, io.EOF) {
			if !exited {
				return errors.New("jobs: command stream ended without an exit event")
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("jobs: read command stream: %w", err)
		}

		switch event.Type {
		case commands.EventStdout:
			if err := writeOutput(req.Stdout, event.Data); err != nil {
				return fmt.Errorf("jobs: write command stdout: %w", err)
			}
		case commands.EventStderr:
			if err := writeOutput(req.Stderr, event.Data); err != nil {
				return fmt.Errorf("jobs: write command stderr: %w", err)
			}
		case commands.EventExit:
			exited = true
			if event.ExitCode != nil {
				result.ExitCode = *event.ExitCode
			}
			result.Signal = event.Signal
		case commands.EventError:
			detail := event.Message
			if detail == "" {
				detail = event.Error
			}
			if detail == "" {
				return errors.New("jobs: command failed")
			}
			return fmt.Errorf("jobs: command failed: %s", detail)
		}
	}
}

func (s *Service) cleanupSandbox(ctx context.Context, sandboxID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultCleanupTimeout)
	defer cancel()
	if _, err := s.sandboxes.Delete(cleanupCtx, sandboxID); err != nil {
		return fmt.Errorf("jobs: delete sandbox %s: %w", sandboxID, err)
	}
	return nil
}

func writeOutput(w io.Writer, data string) error {
	if w == nil {
		return nil
	}
	_, err := io.WriteString(w, data)
	return err
}
