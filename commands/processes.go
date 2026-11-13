package commands

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Processes lists the user-space processes of a sandbox, busiest first.
//
// A running sandbox is read live. A stopped sandbox returns the last stored sample with Live false.
func (s *Service) Processes(ctx context.Context, sandboxID string, opts ...core.RequestOption) (*model.ProcessListResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	var out model.ProcessListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "processes"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
