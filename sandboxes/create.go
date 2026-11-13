package sandboxes

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Create provisions one sandbox and returns the accepted sandbox.
//
// A generated Idempotency-Key is sent by default, so a retry of the same logical call cannot
// create a second billable sandbox. Override it with core.WithIdempotencyKey.
//
// If req.Count is greater than 1, use CreateMany to receive every sandbox in the batch.
func (s *Service) Create(ctx context.Context, req CreateRequest, opts ...core.RequestOption) (*model.Sandbox, error) {
	out, err := s.create(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	return &out.Sandbox, nil
}

// CreateMany provisions a batch from a saved snapshot and returns every sandbox.
//
// Count must be greater than 1 and From must name a saved snapshot. For a single sandbox, Create
// is the simpler call.
func (s *Service) CreateMany(ctx context.Context, req CreateRequest, opts ...core.RequestOption) ([]model.Sandbox, error) {
	out, err := s.create(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if out.Sandboxes != nil {
		return *out.Sandboxes, nil
	}
	return []model.Sandbox{out.Sandbox}, nil
}

func (s *Service) create(ctx context.Context, req CreateRequest, opts ...core.RequestOption) (*model.CreateSandboxResponse, error) {
	r := &core.Request{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		JSON:   req,
	}
	if err := withDefaultIdempotencyKey(r); err != nil {
		return nil, err
	}
	var out model.CreateSandboxResponse
	if err := s.do(ctx, r, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func withDefaultIdempotencyKey(req *core.Request) error {
	key, err := core.NewIdempotencyKey()
	if err != nil {
		return err
	}
	req.SetHeader("Idempotency-Key", key)
	return nil
}
