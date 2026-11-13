package account

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Me returns the GitHub identity, account status and zero-data-retention flag of the authenticated
// Boat account.
func (s *Service) Me(ctx context.Context, opts ...core.RequestOption) (*model.MeResponse, error) {
	var out model.MeResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/me",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
