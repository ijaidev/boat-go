package credentials_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func newService(t *testing.T, routes ...testutil.Route) *credentials.Service {
	t.Helper()
	srv := testutil.NewServer(t, routes...)
	return credentials.New(testutil.Transport(t, srv.URL, "boat_test_key"))
}

func decodeBody(t *testing.T, r *http.Request, out any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
}

func TestService_NilTransport(t *testing.T) {
	t.Parallel()

	svc := credentials.New(nil)
	if _, err := svc.ListProfiles(context.Background()); err == nil {
		t.Fatal("ListProfiles = nil, want transport error")
	}
	if _, err := svc.GetProxy(context.Background(), credentials.GetProxyRequest{}); err == nil {
		t.Fatal("GetProxy = nil, want transport error")
	}
}

func TestService_RequiresIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *credentials.Service) error
	}{
		{
			name: "create profile name",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.CreateProfile(ctx, credentials.CreateProfileRequest{})
				return err
			},
		},
		{
			name: "update profile name",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.UpdateProfile(ctx, "", credentials.UpdateProfileRequest{})
				return err
			},
		},
		{
			name: "delete profile name",
			call: func(ctx context.Context, svc *credentials.Service) error {
				return svc.DeleteProfile(ctx, "")
			},
		},
		{
			name: "set key profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "set key credential",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "work", "", credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "set key value",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{})
				return err
			},
		},
		{
			name: "delete key profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteKey(ctx, "", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name: "delete key credential",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteKey(ctx, "work", "")
				return err
			},
		},
		{
			name: "list accounts profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListAccounts(ctx, "", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name: "list accounts credential",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListAccounts(ctx, "work", "")
				return err
			},
		},
		{
			name: "add account profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "add account credential",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "work", "", credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "add account value",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{})
				return err
			},
		},
		{
			name: "switch account profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SwitchAccount(ctx, "", model.OpenaiAPIKey, "acct_1")
				return err
			},
		},
		{
			name: "switch account credential",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SwitchAccount(ctx, "work", "", "acct_1")
				return err
			},
		},
		{
			name: "switch account account",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SwitchAccount(ctx, "work", model.OpenaiAPIKey, "")
				return err
			},
		},
		{
			name: "delete account account",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteAccount(ctx, "work", model.OpenaiAPIKey, "")
				return err
			},
		},
		{
			name: "set custom api profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "", "STRIPE_KEY", credentials.SetCustomAPIRequest{Value: "test_value", BaseURL: "https://api.example.com"})
				return err
			},
		},
		{
			name: "set custom api name",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "", credentials.SetCustomAPIRequest{Value: "test_value", BaseURL: "https://api.example.com"})
				return err
			},
		},
		{
			name: "set custom api value",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{BaseURL: "https://api.example.com"})
				return err
			},
		},
		{
			name: "set custom api base url",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{Value: "test_value"})
				return err
			},
		},
		{
			name: "delete custom api profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteCustomAPI(ctx, "", "STRIPE_KEY")
				return err
			},
		},
		{
			name: "delete custom api name",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteCustomAPI(ctx, "work", "")
				return err
			},
		},
		{
			name: "sandbox usage sandbox id",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SandboxUsage(ctx, "", credentials.SandboxUsageRequest{})
				return err
			},
		},
		{
			name: "sandbox access sandbox id",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetSandboxAccess(ctx, "", credentials.SetSandboxAccessRequest{Blocked: true})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if err := tt.call(context.Background(), credentials.New(d)); err == nil {
				t.Fatal("call = nil, want validation error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("calls = %d, want 0", len(d.Calls))
			}
		})
	}
}

func TestService_ReadsAreRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *credentials.Service) error
	}{
		{
			name: "list profiles",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListProfiles(ctx)
				return err
			},
		},
		{
			name: "list accounts",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListAccounts(ctx, "work", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name: "get proxy",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.GetProxy(ctx, credentials.GetProxyRequest{})
				return err
			},
		},
		{
			name: "usage",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.Usage(ctx, credentials.UsageRequest{})
				return err
			},
		},
		{
			name: "sandbox usage",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SandboxUsage(ctx, "bx_example", credentials.SandboxUsageRequest{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{DoFunc: func(context.Context, *core.Request, any, ...core.RequestOption) error {
				return nil
			}}
			if err := tt.call(context.Background(), credentials.New(d)); err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(d.Calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(d.Calls))
			}
			if d.Calls[0].NoRetry {
				t.Fatal("NoRetry = true, want a retryable read")
			}
		})
	}
}

func TestService_MutationsAreNotRetried(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *credentials.Service) error
	}{
		{
			name: "create profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.CreateProfile(ctx, credentials.CreateProfileRequest{Name: "work"})
				return err
			},
		},
		{
			name: "update profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.UpdateProfile(ctx, "work", credentials.UpdateProfileRequest{Name: "renamed"})
				return err
			},
		},
		{
			name: "delete profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				return svc.DeleteProfile(ctx, "work")
			},
		},
		{
			name: "set key",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "delete key",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteKey(ctx, "work", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name: "add account",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name: "switch account",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SwitchAccount(ctx, "work", model.OpenaiAPIKey, "acct_1")
				return err
			},
		},
		{
			name: "delete account",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteAccount(ctx, "work", model.OpenaiAPIKey, "acct_1")
				return err
			},
		},
		{
			name: "set custom api",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{Value: "test_value", BaseURL: "https://api.example.com"})
				return err
			},
		},
		{
			name: "delete custom api",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteCustomAPI(ctx, "work", "STRIPE_KEY")
				return err
			},
		},
		{
			name: "set proxy",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetProxy(ctx, credentials.SetProxyRequest{Enabled: true})
				return err
			},
		},
		{
			name: "set sandbox access",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetSandboxAccess(ctx, "bx_example", credentials.SetSandboxAccessRequest{Blocked: true})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{DoFunc: func(context.Context, *core.Request, any, ...core.RequestOption) error {
				return nil
			}}
			if err := tt.call(context.Background(), credentials.New(d)); err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(d.Calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(d.Calls))
			}
			if !d.Calls[0].NoRetry {
				t.Fatal("NoRetry = false, want true")
			}
		})
	}
}
