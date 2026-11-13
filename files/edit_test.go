package files_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Mkdir_SendsMode(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/folder",
		Body:   `{"ok":true,"type":"folder.created","success":true,"path":"/home/user/new","name":"new","isDir":true,"isSymlink":false,"size":0,"mode":"0700","modifiedAt":"2026-01-02T03:04:05Z"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Mkdir(context.Background(), "bx_example", files.MkdirRequest{
		Path: "/home/user/new",
		Mode: core.Ptr("0700"),
	})
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if body["path"] != "/home/user/new" || body["mode"] != "0700" {
		t.Fatalf("body = %v", body)
	}
	if !got.IsDir || got.Path != "/home/user/new" {
		t.Fatalf("response = %#v", got)
	}
}

func TestService_Move_SendsOverwrite(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/move",
		Body:   `{"ok":true,"type":"file.moved","success":true,"path":"/home/user/a","destination":"/tmp/b"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Move(context.Background(), "bx_example", files.MoveRequest{
		Source:      "/home/user/a",
		Destination: "/tmp/b",
		Overwrite:   core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if body["source"] != "/home/user/a" || body["destination"] != "/tmp/b" || body["overwrite"] != true {
		t.Fatalf("body = %v", body)
	}
	if got.Destination != "/tmp/b" {
		t.Fatalf("destination = %q", got.Destination)
	}
}

func TestService_Chmod_SendsMode(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/permissions",
		Body:   `{"ok":true,"type":"file.permissions_set","success":true,"path":"/home/user/run.sh","name":"run.sh","isDir":false,"isSymlink":false,"size":10,"mode":"0755","modifiedAt":"2026-01-02T03:04:05Z"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Chmod(context.Background(), "bx_example", files.ChmodRequest{
		Path: "/home/user/run.sh",
		Mode: "0755",
	})
	if err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if body["path"] != "/home/user/run.sh" || body["mode"] != "0755" {
		t.Fatalf("body = %v", body)
	}
	if got.Mode != "0755" {
		t.Fatalf("mode = %q", got.Mode)
	}
}

func TestService_Replace_ReturnsPerFileResults(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/replace",
		Body: `{"ok":true,"type":"file.replaced","success":true,"results":[
			{"file":"/home/user/a.txt","success":true,"replacements":2},
			{"file":"/home/user/b.txt","success":false,"error":"too large"}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Replace(context.Background(), "bx_example", files.ReplaceRequest{
		Paths:    []string{"/home/user/a.txt", "/home/user/b.txt"},
		Pattern:  "old",
		NewValue: "new",
	})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	paths, ok := body["paths"].([]any)
	if !ok || len(paths) != 2 {
		t.Fatalf("body paths = %v", body["paths"])
	}
	if body["pattern"] != "old" || body["newValue"] != "new" {
		t.Fatalf("body = %v", body)
	}
	if len(got.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(got.Results))
	}
	if !got.Results[0].Success || got.Results[0].Replacements == nil || *got.Results[0].Replacements != 2 {
		t.Fatalf("first result = %#v", got.Results[0])
	}
	if got.Results[1].Success || got.Results[1].Error == nil {
		t.Fatalf("second result = %#v", got.Results[1])
	}
}

func TestService_Edit_RequiresInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *files.Service) error
	}{
		{
			name: "mkdir missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Mkdir(ctx, "bx_example", files.MkdirRequest{})
				return err
			},
		},
		{
			name: "move missing source",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Move(ctx, "bx_example", files.MoveRequest{Destination: "/tmp/b"})
				return err
			},
		},
		{
			name: "move missing destination",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Move(ctx, "bx_example", files.MoveRequest{Source: "/home/user/a"})
				return err
			},
		},
		{
			name: "chmod missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Chmod(ctx, "bx_example", files.ChmodRequest{Mode: "0755"})
				return err
			},
		},
		{
			name: "chmod missing mode",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Chmod(ctx, "bx_example", files.ChmodRequest{Path: "/home/user/a"})
				return err
			},
		},
		{
			name: "replace empty paths",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Replace(ctx, "bx_example", files.ReplaceRequest{Pattern: "old"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if err := tt.call(context.Background(), files.New(d)); err == nil {
				t.Fatal("call = nil, want validation error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("Do calls = %d, want 0", len(d.Calls))
			}
		})
	}
}
