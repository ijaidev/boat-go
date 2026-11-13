package files_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Delete_SendsRecursive(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.deleted","success":true,"path":"/home/user/build"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("path"); got != "/home/user/build" {
				t.Fatalf("path = %q", got)
			}
			if got := query.Get("recursive"); got != "true" {
				t.Fatalf("recursive = %q, want true", got)
			}
		},
	})

	got, err := svc.Delete(context.Background(), "bx_example", "/home/user/build", files.DeleteOptions{Recursive: true})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.Path != "/home/user/build" {
		t.Fatalf("path = %q", got.Path)
	}
}

func TestService_Delete_OmitsRecursiveByDefault(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.deleted","success":true,"path":"/home/user/a.txt"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if r.URL.Query().Has("recursive") {
				t.Fatalf("recursive = %q, want omitted", r.URL.Query().Get("recursive"))
			}
		},
	})

	if _, err := svc.Delete(context.Background(), "bx_example", "/home/user/a.txt", files.DeleteOptions{}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestService_List_ReturnsEntries(t *testing.T) {
	t.Parallel()

	wantTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files/list",
		Body: `{"ok":true,"type":"file.list","success":true,"path":"/home/user/project","truncated":true,"files":[
			{"name":"main.go","path":"/home/user/project/main.go","isDir":false,"isSymlink":false,"size":42,"mode":"0644","modifiedAt":"2026-01-02T03:04:05Z"}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("path"); got != "/home/user/project" {
				t.Fatalf("path = %q", got)
			}
		},
	})

	got, err := svc.List(context.Background(), "bx_example", files.ListRequest{Path: "/home/user/project"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Path != "/home/user/project" || !got.Truncated {
		t.Fatalf("response = %#v", got)
	}
	if len(got.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(got.Files))
	}
	entry := got.Files[0]
	if entry.Name != "main.go" || entry.IsDir || entry.IsSymlink || entry.Size != 42 || entry.Mode != "0644" {
		t.Fatalf("entry = %#v", entry)
	}
	if !entry.ModifiedAt.Equal(wantTime) {
		t.Fatalf("modified = %v, want %v", entry.ModifiedAt, wantTime)
	}
}

func TestService_List_DefaultsToWorkDirectory(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files/list",
		Body:   `{"ok":true,"type":"file.list","success":true,"path":"/home/user","truncated":false,"files":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if r.URL.Query().Has("path") {
				t.Fatalf("path = %q, want omitted", r.URL.Query().Get("path"))
			}
		},
	})

	if _, err := svc.List(context.Background(), "bx_example", files.ListRequest{}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestService_Info_ReturnsEntry(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files/info",
		Body:   `{"ok":true,"type":"file.info","success":true,"path":"/home/user/a.txt","name":"a.txt","isDir":false,"isSymlink":true,"size":3,"mode":"0755","modifiedAt":"2026-01-02T03:04:05Z"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("path"); got != "/home/user/a.txt" {
				t.Fatalf("path = %q", got)
			}
		},
	})

	got, err := svc.Info(context.Background(), "bx_example", "/home/user/a.txt")
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got.Name != "a.txt" || !got.IsSymlink || got.Mode != "0755" {
		t.Fatalf("info = %#v", got)
	}
}

func TestService_Search_SendsGlobAndReturnsPaths(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files/search",
		Body:   `{"ok":true,"type":"file.search","success":true,"files":["/home/user/a.go","/home/user/b.go"],"truncated":false,"errors":["/home/user/private: permission denied"]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("path"); got != "/home/user" {
				t.Fatalf("path = %q", got)
			}
			if got := query.Get("pattern"); got != "*.go" {
				t.Fatalf("pattern = %q", got)
			}
		},
	})

	got, err := svc.Search(context.Background(), "bx_example", files.SearchRequest{Path: "/home/user", Pattern: "*.go"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Files) != 2 || got.Files[1] != "/home/user/b.go" {
		t.Fatalf("files = %v", got.Files)
	}
	if got.Errors == nil || len(*got.Errors) != 1 {
		t.Fatalf("errors = %v", got.Errors)
	}
}

func TestService_Find_SendsRegexAndReturnsMatches(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files/find",
		Body:   `{"ok":true,"type":"file.find","success":true,"truncated":true,"matches":[{"file":"/home/user/a.go","line":3,"content":"needle here"}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("pattern"); got != "need+le" {
				t.Fatalf("pattern = %q", got)
			}
			if got := query.Get("regex"); got != "true" {
				t.Fatalf("regex = %q, want true", got)
			}
		},
	})

	got, err := svc.Find(context.Background(), "bx_example", files.FindRequest{Pattern: "need+le", Regex: true})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if !got.Truncated || len(got.Matches) != 1 {
		t.Fatalf("response = %#v", got)
	}
	if got.Matches[0].Line != 3 || got.Matches[0].Content != "needle here" {
		t.Fatalf("match = %#v", got.Matches[0])
	}
}

func TestService_Manage_RequiresInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *files.Service) error
	}{
		{
			name: "delete missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Delete(ctx, "bx_example", "", files.DeleteOptions{})
				return err
			},
		},
		{
			name: "list missing id",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.List(ctx, "", files.ListRequest{})
				return err
			},
		},
		{
			name: "info missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Info(ctx, "bx_example", "")
				return err
			},
		},
		{
			name: "search missing pattern",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Search(ctx, "bx_example", files.SearchRequest{})
				return err
			},
		},
		{
			name: "find missing pattern",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Find(ctx, "bx_example", files.FindRequest{})
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
