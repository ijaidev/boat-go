package commands

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// RunRequest is the body of POST /sandboxes/{sandboxId}/commands.
//
// TimeoutSeconds is in seconds. A synchronous Run uses 30 seconds by default and accepts 1-600.
// Leave it nil for the default; Start forwards a value only when set.
type RunRequest struct {
	Command        string  `json:"command"`
	Cwd            *string `json:"cwd,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

// Run executes a command synchronously and returns its result.
//
// The call is never retried, because a failed request may still have run the command. TimeoutSeconds
// defaults to 30 seconds on the server and must be between 1 and 600.
func (s *Service) Run(ctx context.Context, sandboxID string, req RunRequest, opts ...core.RequestOption) (*model.CommandResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	var out model.CommandResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "commands"),
		JSON:   commandBody(req),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Stream executes a command and streams its output as newline-delimited frames.
//
// The first frame is started, then stdout and stderr chunks, then exactly one exit or error. Next
// returns io.EOF at the end of the stream. A malformed frame is terminal. The request is never
// retried.
func (s *Service) Stream(ctx context.Context, sandboxID string, req RunRequest, opts ...core.RequestOption) (core.Stream[model.CommandStreamFrame], error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	body := commandBody(req)
	body.Stream = core.Ptr(true)
	return openStream[model.CommandStreamFrame](s, ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "commands"),
		JSON:   body,
	}, opts...)
}

// Start starts a command in the background and returns its process id.
//
// Detached commands are not limited by the synchronous timeout. A TimeoutSeconds value is forwarded
// only when the caller sets one. The request is never retried.
func (s *Service) Start(ctx context.Context, sandboxID string, req RunRequest, opts ...core.RequestOption) (*model.CommandStartedResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	body := commandBody(req)
	body.Detached = core.Ptr(true)
	var out model.CommandStartedResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(sandboxID, "commands"),
		JSON:   body,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func commandBody(req RunRequest) model.CommandRequest {
	return model.CommandRequest{
		Command:        req.Command,
		Cwd:            req.Cwd,
		TimeoutSeconds: req.TimeoutSeconds,
	}
}
