package agents

import (
	"time"

	"github.com/ijaidev/boat-go/model"
)

// PromptRequest is the body of POST /sandboxes/{sandboxID}/prompt.
//
// Provider credentials must already be configured in the Boat dashboard. An empty Provider is
// omitted from the body, which selects the harness saved on the Agents dashboard.
//
// A sandbox runs many conversations in parallel. Set New to start a fresh one, set ConversationID
// to continue a specific one, or leave both zero to continue the most recently active
// conversation. The response carries the conversation id the prompt ran in.
type PromptRequest struct {
	Provider        model.PromptRequestProvider `json:"provider,omitempty"`
	Model           string                      `json:"model,omitempty"`
	ReasoningEffort string                      `json:"reasoningEffort,omitempty"`
	Fast            *bool                       `json:"fast,omitempty"`
	Prompt          string                      `json:"prompt"`
	New             bool                        `json:"new,omitempty"`
	ConversationID  string                      `json:"conversationId,omitempty"`
}

// EventsRequest filters GET /sandboxes/{sandboxID}/events.
//
// Type matches event types such as prompt, response and steer. Conversation scopes the response to
// one or more conversation ids. Cursor is opaque: pass back NextCursor from EventsPage and never
// build or decode one. ETag is sent as If-None-Match, so an unchanged page returns a 304 and an
// empty EventsPage instead of an error.
//
// PollInterval tunes StreamEvents only; Events ignores it. Zero uses one second.
type EventsRequest struct {
	Type         []string
	Conversation []string
	Limit        int
	Cursor       string
	Sort         string
	ETag         string
	PollInterval time.Duration
}

// EventsPage is one page of sandbox events.
//
// Items keeps the API's extensible event shape, including unknown types and additional properties.
// An unchanged long-poll returns an empty page with the same ETag.
type EventsPage struct {
	Items      []model.SandboxEvent
	NextCursor string
	HasMore    bool
	ETag       string
}

// SteerRequest is the body of POST /sandboxes/{sandboxID}/steer.
//
// Conversation is optional. Leave it empty to steer the most recently active conversation.
type SteerRequest struct {
	Conversation string `json:"conversation,omitempty"`
	Message      string `json:"message"`
}

// WaitOptions controls WaitForPrompt. PollInterval defaults to one second. A zero Timeout means no
// deadline beyond the caller's context.
type WaitOptions struct {
	PollInterval time.Duration
	Timeout      time.Duration
}
