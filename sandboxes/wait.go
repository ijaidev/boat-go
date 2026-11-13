package sandboxes

import (
	"context"
	"fmt"
	"time"

	"github.com/ijaidev/boat-go/model"
)

// WaitOptions controls WaitUntilReady. PollInterval defaults to one second. A zero Timeout means
// no deadline beyond the caller's context.
type WaitOptions struct {
	PollInterval time.Duration
	Timeout      time.Duration
}

// WaitUntilReady polls Get until the sandbox is ready, idle or running.
//
// It returns an error when the sandbox reaches error, cancelled or archived, or when the context or
// Timeout ends first.
func (s *Service) WaitUntilReady(ctx context.Context, id string, opts WaitOptions) (*model.Sandbox, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	for {
		sandbox, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		switch sandbox.State {
		case model.SandboxStateReady, model.SandboxStateIdle, model.SandboxStateRunning:
			return sandbox, nil
		case model.SandboxStateError, model.SandboxStateCancelled, model.SandboxStateArchived:
			return nil, fmt.Errorf("sandboxes: sandbox %s is %s", id, sandbox.State)
		}

		timer := time.NewTimer(opts.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
