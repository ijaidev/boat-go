package credentials

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ListAccounts returns every account of one credential, in the order they were added.
//
// Labels are the account's email for a sign-in or a key's last characters. Secrets are never
// returned. The profile must be one the caller manages.
func (s *Service) ListAccounts(ctx context.Context, profile string, credential model.CredentialID, opts ...core.RequestOption) ([]model.CredentialAccount, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	var out model.CredentialAccountsResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   profilePath(profile, "accounts", string(credential)),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Accounts, nil
}

// AddAccount adds an API key as another account of a credential and returns the accounts.
//
// The first key of a credential becomes the one in use. A sign-in credential answers 400
// not_an_api_key. The key is write-only and is never logged. The request is never retried.
func (s *Service) AddAccount(ctx context.Context, profile string, credential model.CredentialID, req SetKeyRequest, opts ...core.RequestOption) ([]model.CredentialAccount, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	if err := requireKeyValue(req.Value); err != nil {
		return nil, err
	}
	var out model.CredentialAccountsResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPost,
		Path:    profilePath(profile, "accounts", string(credential)),
		JSON:    req,
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Accounts, nil
}

// SwitchAccount makes one account of a credential the one in use and returns the accounts.
//
// The previous account stays as another account. Every sandbox on the profile switches at once,
// running ones included, with no restart. Switching to the account already in use changes nothing.
// The request is never retried.
func (s *Service) SwitchAccount(ctx context.Context, profile string, credential model.CredentialID, account string, opts ...core.RequestOption) ([]model.CredentialAccount, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	if err := requireAccount(account); err != nil {
		return nil, err
	}
	var out model.CredentialAccountsResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodPut,
		Path:    profilePath(profile, "accounts", string(credential), account, "active"),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Accounts, nil
}

// DeleteAccount forgets one account of a credential and returns the accounts that remain.
//
// Deleting the account in use switches the credential to another account first, or removes the
// credential when it was the last one. The request is never retried.
func (s *Service) DeleteAccount(ctx context.Context, profile string, credential model.CredentialID, account string, opts ...core.RequestOption) ([]model.CredentialAccount, error) {
	if err := requireProfile(profile); err != nil {
		return nil, err
	}
	if err := requireCredential(credential); err != nil {
		return nil, err
	}
	if err := requireAccount(account); err != nil {
		return nil, err
	}
	var out model.CredentialAccountsResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    profilePath(profile, "accounts", string(credential), account),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Accounts, nil
}
