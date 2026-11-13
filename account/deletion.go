package account

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// DeletionOperation returns the current state of one accepted sandbox or snapshot deletion.
//
// Deletions are asynchronous. The target is already gone from every list, restore and fork once the
// operation exists, but stored data can still be purged in the background. Poll this method until
// Status is model.DeletionOperationStatusCompleted or model.DeletionOperationStatusBlocked. Only
// operations owned by the authenticated account are returned; another account's id answers 404.
func (s *Service) DeletionOperation(ctx context.Context, operationID string, opts ...core.RequestOption) (*model.DeletionOperation, error) {
	if err := requireOperationID(operationID); err != nil {
		return nil, err
	}

	var out model.DeletionOperationResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/deletion-operations/" + url.PathEscape(operationID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Operation, nil
}
