package agents

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Conversations lists every conversation on a sandbox, most recently prompted first.
//
// Each entry reports whether a turn is running, how many prompts it holds, the last harness and
// model, a prompt preview, and whether it is the conversation a Prompt without New or
// ConversationID would continue.
func (s *Service) Conversations(ctx context.Context, sandboxID string) ([]model.Conversation, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	var out model.ConversationsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "conversations"),
	}, &out); err != nil {
		return nil, err
	}
	return out.Conversations, nil
}

// Steer sends an extra instruction into a turn that is already running.
//
// A native harness keeps working and reports Mode native. A harness without a mid-turn primitive
// is interrupted and resumed on the same conversation, which reports Native false. A 409
// no_running_turn means nothing was in flight; use Prompt instead. Inspect APIError.Details for the
// terminal reason (turn_ended, turn_not_started, no_turn, not_running, delivery_failed).
func (s *Service) Steer(ctx context.Context, sandboxID string, req SteerRequest) (*model.SteerResponse, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	var out model.SteerResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "steer"),
		JSON:   req,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Interrupt stops running agent work on a sandbox.
//
// An empty conversation interrupts every conversation. Pass a conversation id to stop only that
// one, leaving the others running.
func (s *Service) Interrupt(ctx context.Context, sandboxID, conversation string) (*model.SandboxActionResponse, error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	r := &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "interrupt"),
	}
	if conversation != "" {
		r.AddQuery("conversation", conversation)
	}
	var out model.SandboxActionResponse
	if err := s.do(ctx, r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
