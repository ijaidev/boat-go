package files_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Write_SendsBody(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.written","success":true,"path":"/home/user/run.sh","encoding":"utf8","size":10}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Write(context.Background(), "bx_example", files.WriteRequest{
		Path:    "/home/user/run.sh",
		Content: "echo hello",
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if body["path"] != "/home/user/run.sh" || body["content"] != "echo hello" {
		t.Fatalf("body = %v", body)
	}
	if body["encoding"] != "utf8" {
		t.Fatalf("encoding = %v, want utf8", body["encoding"])
	}
	if got.Path != "/home/user/run.sh" || got.Size != 10 {
		t.Fatalf("result = %#v", got)
	}
	if got.Encoding != model.FileWriteResponseEncodingUTF8 {
		t.Fatalf("encoding = %q", got.Encoding)
	}
}

func TestService_Write_SendsBase64(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.written","success":true,"path":"/home/user/blob.bin","encoding":"base64","size":3}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	if _, err := svc.Write(context.Background(), "bx_example", files.WriteRequest{
		Path:     "/home/user/blob.bin",
		Content:  "AAH/",
		Encoding: files.EncodingBase64,
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if body["encoding"] != "base64" || body["content"] != "AAH/" {
		t.Fatalf("body = %v", body)
	}
}

func TestService_Write_RequiresInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		req  files.WriteRequest
	}{
		{name: "missing sandbox id", req: files.WriteRequest{Path: "/home/user/a"}},
		{name: "missing path", id: "bx_example", req: files.WriteRequest{Content: "x"}},
		{name: "unsupported encoding", id: "bx_example", req: files.WriteRequest{Path: "/home/user/a", Encoding: files.Encoding("latin1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if _, err := files.New(d).Write(context.Background(), tt.id, tt.req); err == nil {
				t.Fatal("Write = nil, want validation error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("Do calls = %d, want 0", len(d.Calls))
			}
		})
	}
}
