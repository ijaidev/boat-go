package boat_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestNew_WiresServices(t *testing.T) {
	t.Parallel()

	var gotUA string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example",
		Body:   `{"ok":true,"type":"sandbox.info","sandbox":{"id":"bx_example","name":"n","state":"ready","desktopAvailable":false,"snapshotAvailable":false}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotUA = r.Header.Get("User-Agent")
		},
	})

	client, err := boat.New(boat.Config{BaseURL: srv.URL, APIKey: "boat_test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.Agents == nil {
		t.Fatal("Agents = nil")
	}
	if client.Commands == nil {
		t.Fatal("Commands = nil")
	}
	if client.Files == nil {
		t.Fatal("Files = nil")
	}
	if client.Telemetry == nil {
		t.Fatal("Telemetry = nil")
	}
	if client.Sandboxes == nil {
		t.Fatal("Sandboxes = nil")
	}
	if client.Snapshots == nil {
		t.Fatal("Snapshots = nil")
	}
	sb, err := client.Sandboxes.Get(context.Background(), "bx_example")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb.ID != "bx_example" {
		t.Fatalf("sandbox id = %q", sb.ID)
	}
	if gotUA != "boat-go/"+boat.Version {
		t.Fatalf("User-Agent = %q, want boat-go/%s", gotUA, boat.Version)
	}
}

func TestNew_ValidatesConfig(t *testing.T) {
	t.Parallel()

	if _, err := boat.New(boat.Config{}); err == nil {
		t.Fatal("New = nil, want missing credentials error")
	}
}

func TestIsCode_ReExport(t *testing.T) {
	t.Parallel()

	err := &boat.APIError{Status: 404, Code: "not_found"}
	if !boat.IsCode(err, "not_found") {
		t.Fatal("IsCode = false")
	}
	if !errors.Is(err, err) {
		t.Fatal("errors.Is = false")
	}
}
