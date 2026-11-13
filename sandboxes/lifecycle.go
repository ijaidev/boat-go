package sandboxes

import (
	"context"
	"errors"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Stop archives a sandbox and snapshots its disk. It is not a pause.
func (s *Service) Stop(ctx context.Context, id string, req StopOptions, opts ...core.RequestOption) (*model.StopOperation, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out model.SandboxActionResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "stop"),
		JSON:   req,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	if out.Operation == nil {
		return nil, errors.New("sandboxes: stop response has no operation")
	}
	return out.Operation, nil
}

// GetStop returns the state of a stop operation.
func (s *Service) GetStop(ctx context.Context, id, stopID string, opts ...core.RequestOption) (*model.StopOperation, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	if stopID == "" {
		return nil, errors.New("sandboxes: stop id is required")
	}
	var out model.StopOperationResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(id, "stops", stopID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Stop, nil
}

// Resume starts a stopped sandbox from its newest snapshot.
func (s *Service) Resume(ctx context.Context, id string, req ResumeRequest, opts ...core.RequestOption) (*model.Sandbox, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out model.SandboxActionResponse
	err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "resume"),
		JSON:   req,
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	if out.Sandbox == nil {
		return nil, errors.New("sandboxes: resume response has no sandbox")
	}
	return out.Sandbox, nil
}

// Fork provisions a new sandbox from the source sandbox's newest snapshot.
//
// A generated Idempotency-Key is sent by default, so a retry cannot create a second billable fork.
func (s *Service) Fork(ctx context.Context, id string, req ForkRequest, opts ...core.RequestOption) (*model.Sandbox, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	r := &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "fork"),
		JSON:   req,
	}
	if err := withDefaultIdempotencyKey(r); err != nil {
		return nil, err
	}
	var out model.SandboxActionResponse
	if err := s.do(ctx, r, &out, opts...); err != nil {
		return nil, err
	}
	if out.Sandbox == nil {
		return nil, errors.New("sandboxes: fork response has no sandbox")
	}
	return out.Sandbox, nil
}

// Delete permanently deletes a sandbox and its snapshots. The call is asynchronous; poll the
// returned operation with account.GetDeletionOperation.
func (s *Service) Delete(ctx context.Context, id string, opts ...core.RequestOption) (*model.DeletionOperation, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	req := &core.Request{
		Method: http.MethodDelete,
		Path:   sandboxPath(id),
	}
	req.SetHeader("X-Ascii-Confirm-Delete", id)
	var out model.DeletionOperationResponse
	if err := s.do(ctx, req, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Operation, nil
}

type shareResponse struct {
	Sandbox         *model.Sandbox `json:"sandbox"`
	RestartRequired bool           `json:"restartRequired"`
	Message         string         `json:"message"`
}

// Share lets every member of the organization that pays for the sandbox use it.
func (s *Service) Share(ctx context.Context, id string, opts ...core.RequestOption) (*ShareResult, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out shareResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "share"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &ShareResult{Sandbox: out.Sandbox, RestartRequired: out.RestartRequired, Message: out.Message}, nil
}

type unshareResponse struct {
	Sandbox *model.Sandbox `json:"sandbox"`
	Message string         `json:"message"`
}

// Unshare makes a shared organization sandbox private again.
func (s *Service) Unshare(ctx context.Context, id string, opts ...core.RequestOption) (*UnshareResult, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out unshareResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "unshare"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &UnshareResult{Sandbox: out.Sandbox, Message: out.Message}, nil
}

type recoverResponse struct {
	Message string `json:"message"`
}

// Recover power-cycles a sandbox that stopped responding. The disk is not touched.
//
// Deprecated: Boat may remove this endpoint. Prefer stop and resume when the sandbox no longer
// answers.
func (s *Service) Recover(ctx context.Context, id string, opts ...core.RequestOption) (*RecoverResult, error) {
	if err := requireID(id); err != nil {
		return nil, err
	}
	var out recoverResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   sandboxPath(id, "recover"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &RecoverResult{Message: out.Message}, nil
}
