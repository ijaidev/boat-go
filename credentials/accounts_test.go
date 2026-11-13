package credentials_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_ListAccounts_ReturnsAccounts(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/credential-profiles/work/accounts/openai-api-key",
		Body:   accountsJSON,
	})

	accounts, err := svc.ListAccounts(context.Background(), "work", model.OpenaiAPIKey)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accounts))
	}
	if !accounts[0].Active || accounts[0].Label != "ending 1234" {
		t.Fatalf("first account = %#v", accounts[0])
	}
	if accounts[1].Active || accounts[1].ID != "acct_2" {
		t.Fatalf("second account = %#v", accounts[1])
	}
	if accounts[1].AddedAt == nil {
		t.Fatal("addedAt = nil, want a time")
	}
	wantAdded := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	if !accounts[1].AddedAt.Equal(wantAdded) {
		t.Fatalf("addedAt = %v, want %v", accounts[1].AddedAt, wantAdded)
	}
}

func TestService_AddAccount_SendsKey(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/credential-profiles/work/accounts/openai-api-key",
		Status: http.StatusCreated,
		Body:   accountsJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body struct {
				Value       string `json:"value"`
				KeepCurrent *bool  `json:"keepCurrent"`
			}
			decodeBody(t, r, &body)
			if body.Value != "test_key_value" {
				t.Fatal("key value was not forwarded")
			}
			if body.KeepCurrent == nil || !*body.KeepCurrent {
				t.Fatalf("keepCurrent = %v, want true", body.KeepCurrent)
			}
		},
	})

	accounts, err := svc.AddAccount(context.Background(), "work", model.OpenaiAPIKey, credentials.SetKeyRequest{
		Value:       "test_key_value",
		KeepCurrent: core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accounts))
	}
}

func TestService_SwitchAccount_ActivatesAccount(t *testing.T) {
	t.Parallel()

	switched := `{
		"ok": true,
		"type": "credential_account.active",
		"success": true,
		"profileId": "prof_example",
		"credential": "openai-api-key",
		"accounts": [
			{"id": "acct_1", "label": "ending 1234", "active": false, "addedAt": "2026-10-03T00:00:00Z"},
			{"id": "acct_2", "label": "ending 5678", "active": true}
		]
	}`

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work/accounts/openai-api-key/acct_2/active",
		Body:   switched,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if r.ContentLength != 0 {
				t.Fatalf("ContentLength = %d, want 0", r.ContentLength)
			}
		},
	})

	accounts, err := svc.SwitchAccount(context.Background(), "work", model.OpenaiAPIKey, "acct_2")
	if err != nil {
		t.Fatalf("SwitchAccount: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accounts))
	}
	if accounts[0].Active || !accounts[1].Active {
		t.Fatalf("active flags = %v, %v", accounts[0].Active, accounts[1].Active)
	}
}

func TestService_DeleteAccount_ReturnsRemainingAccounts(t *testing.T) {
	t.Parallel()

	remaining := `{
		"ok": true,
		"type": "credential_account.deleted",
		"success": true,
		"profileId": "prof_example",
		"credential": "openai-api-key",
		"accounts": [{"id": "acct_2", "label": "ending 5678", "active": true}]
	}`

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/credential-profiles/work/accounts/openai-api-key/acct_1",
		Body:   remaining,
	})

	accounts, err := svc.DeleteAccount(context.Background(), "work", model.OpenaiAPIKey, "acct_1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != "acct_2" || !accounts[0].Active {
		t.Fatalf("accounts = %#v", accounts)
	}
}
