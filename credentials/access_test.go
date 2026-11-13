package credentials_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_SetSandboxAccess_SendsBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		blocked bool
	}{
		{name: "block", blocked: true},
		{name: "unblock", blocked: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t, testutil.Route{
				Method: http.MethodPut,
				Path:   "/sandboxes/bx_example/ai-access",
				Body: fmt.Sprintf(
					`{"ok": true, "type": "sandbox.ai_access_updated", "success": true, "sandboxId": "bx_example", "blocked": %t}`,
					tt.blocked,
				),
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					var body struct {
						Blocked bool `json:"blocked"`
					}
					decodeBody(t, r, &body)
					if body.Blocked != tt.blocked {
						t.Fatalf("blocked = %t, want %t", body.Blocked, tt.blocked)
					}
				},
			})

			res, err := svc.SetSandboxAccess(context.Background(), "bx_example", credentials.SetSandboxAccessRequest{
				Blocked: tt.blocked,
			})
			if err != nil {
				t.Fatalf("SetSandboxAccess: %v", err)
			}
			if res.SandboxID != "bx_example" || res.Blocked != tt.blocked {
				t.Fatalf("response = %#v", res)
			}
		})
	}
}
