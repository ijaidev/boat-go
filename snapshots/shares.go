package snapshots

import (
	"context"
	"errors"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// CreateShare makes a secret code that lets any Boat account create a sandbox from a saved
// snapshot.
//
// The full code is in the response only: Boat stores a hash and never shows it again. Treat
// ShareCode as a secret; never log or persist it in plain text. Re-saving the name changes what
// the code deploys.
func (s *Service) CreateShare(ctx context.Context, name string, opts ...core.RequestOption) (*model.NamedSnapshotShareCreatedResponse, error) {
	if err := requireName(name); err != nil {
		return nil, err
	}
	var out model.NamedSnapshotShareCreatedResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   savedPath(name, "shares"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListShares returns the share codes of a saved snapshot, active and revoked. Only code prefixes
// are returned; the full codes are never shown again.
func (s *Service) ListShares(ctx context.Context, name string, opts ...core.RequestOption) ([]model.NamedSnapshotShare, error) {
	if err := requireName(name); err != nil {
		return nil, err
	}
	var out model.NamedSnapshotShareListResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   savedPath(name, "shares"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Shares, nil
}

// RevokeShare revokes one share code of a saved snapshot.
//
// The code stops creating new sandboxes; sandboxes already created from it keep their own copy.
func (s *Service) RevokeShare(ctx context.Context, name, shareID string, opts ...core.RequestOption) (*model.NamedSnapshotShareRevokedResponse, error) {
	if err := requireName(name); err != nil {
		return nil, err
	}
	if shareID == "" {
		return nil, errors.New("snapshots: share id is required")
	}
	var out model.NamedSnapshotShareRevokedResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodDelete,
		Path:   savedPath(name, "shares", shareID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
