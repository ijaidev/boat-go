package agents

import (
	"context"
	"net/http"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Prompt queues a natural-language work item for an agent harness inside a running sandbox.
//
// The call returns as soon as the prompt is queued (202). Observe progress with Events or poll
// PromptStatus; there is no completion webhook. A prompt ends in finished, failed or interrupted.
//
// Prompting requires provider credentials that are already configured in the Boat dashboard. A 409
// provider_not_configured is terminal: the SDK returns it unchanged and never logs or rewrites its
// details, which may include a setup URL.
func (s *Service) Prompt(ctx context.Context, sandboxID string, req PromptRequest, opts ...core.RequestOption) (*model.PromptResponse, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	var out model.PromptResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "prompt"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// PromptStatus returns the current status of one prompt run.
//
// This is a single read. A caller that wants to wait for completion can poll until PromptRun.Done
// or use WaitForPrompt.
func (s *Service) PromptStatus(ctx context.Context, sandboxID, promptID string, opts ...core.RequestOption) (*model.PromptRun, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	if err := require(promptID, "prompt id"); err != nil {
		return nil, err
	}
	var out model.PromptRunResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "prompts", promptID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.PromptRun, nil
}

// WaitForPrompt polls PromptStatus until the run is done or the context or opts.Timeout ends.
//
// It returns the terminal run for finished, failed and interrupted prompts without turning the
// prompt outcome into an error. Inspect PromptRun.Status. API and transport errors are returned
// immediately.
func (s *Service) WaitForPrompt(ctx context.Context, sandboxID, promptID string, opts WaitOptions) (*model.PromptRun, error) {
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	for {
		run, err := s.PromptStatus(ctx, sandboxID, promptID)
		if err != nil {
			return nil, err
		}
		if run.Done || promptTerminal(run.Status) {
			return run, nil
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

func promptTerminal(status model.PromptRunStatus) bool {
	switch status {
	case model.PromptRunStatusFinished, model.PromptRunStatusFailed, model.PromptRunStatusInterrupted:
		return true
	default:
		return false
	}
}
