package credentials

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// SetSandboxAccess blocks or unblocks a sandbox's AI placeholders and returns the new state.
//
// Blocking works while the sandbox is running or stopped. The proxy then refuses that sandbox's
// requests with 403. It has no effect on a sandbox that holds real provider keys. The sandbox creator,
// or a manager of its credential profile, can call it. The request is never retried.
func (s *Service) SetSandboxAccess(ctx context.Context, sandboxID string, req SetSandboxAccessRequest, opts ...core.RequestOption) (*model.SandboxAiAccessResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	var out model.SandboxAiAccessResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    sandboxPath(sandboxID, "ai-access"),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
