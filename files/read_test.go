package files_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Read_ReadsUTF8(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.read","success":true,"path":"/home/user/notes.txt","encoding":"utf8","size":6,"content":"hello\n"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("path"); got != "/home/user/notes.txt" {
				t.Fatalf("path = %q", got)
			}
			if got := query.Get("encoding"); got != "utf8" {
				t.Fatalf("encoding = %q", got)
			}
		},
	})

	got, err := svc.Read(context.Background(), "bx_example", "/home/user/notes.txt", files.EncodingUTF8)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Path != "/home/user/notes.txt" || got.Size != 6 {
		t.Fatalf("result = %#v", got.FileReadResponse)
	}
	if got.Encoding != model.FileReadResponseEncodingUTF8 {
		t.Fatalf("encoding = %q", got.Encoding)
	}
	content, err := got.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestService_Read_ReadsBase64(t *testing.T) {
	t.Parallel()

	// "AAH/" is the standard base64 form of 0x00 0x01 0xff.
	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.read","success":true,"path":"/home/user/blob.bin","encoding":"base64","size":3,"content":"AAH/"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("encoding"); got != "base64" {
				t.Fatalf("encoding = %q", got)
			}
		},
	})

	got, err := svc.Read(context.Background(), "bx_example", "/home/user/blob.bin", files.EncodingBase64)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	content, err := got.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	want := []byte{0x00, 0x01, 0xff}
	if len(content) != len(want) || content[0] != want[0] || content[1] != want[1] || content[2] != want[2] {
		t.Fatalf("content = %v, want %v", content, want)
	}
}

func TestService_Read_DefaultsToUTF8(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.read","success":true,"path":"/home/user/a.txt","encoding":"utf8","size":1,"content":"a"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("encoding"); got != "utf8" {
				t.Fatalf("encoding = %q, want utf8", got)
			}
		},
	})

	got, err := svc.Read(context.Background(), "bx_example", "/home/user/a.txt", "")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	content, err := got.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if string(content) != "a" {
		t.Fatalf("content = %q", content)
	}
}

func TestService_Read_TooLargeIsTerminal(t *testing.T) {
	t.Parallel()

	var calls int
	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files",
		Status: http.StatusRequestEntityTooLarge,
		Body:   `{"ok":false,"type":"sandbox.error","status":413,"code":"file_too_large","message":"file is too large","requestId":"req_1","error":{"code":"file_too_large","details":{"maxBytes":5242880,"sizeBytes":6291456}}}`,
		Assert: func(t *testing.T, _ *http.Request) {
			t.Helper()
			calls++
		},
	})

	_, err := svc.Read(context.Background(), "bx_example", "/home/user/big.bin", files.EncodingUTF8)
	if !core.IsCode(err, "file_too_large") {
		t.Fatalf("error = %v, want file_too_large", err)
	}
	if calls != 1 {
		t.Fatalf("requests = %d, want 1: a 413 must not be retried", calls)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *core.APIError", err)
	}
	if got := apiErr.Details["maxBytes"]; got != float64(5242880) {
		t.Fatalf("maxBytes = %v", got)
	}
}

func TestService_Read_InvalidBase64Content(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/files",
		Body:   `{"ok":true,"type":"file.read","success":true,"path":"/home/user/bad.bin","encoding":"base64","size":1,"content":"!!!"}`,
	})

	got, err := svc.Read(context.Background(), "bx_example", "/home/user/bad.bin", files.EncodingBase64)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := got.Bytes(); err == nil {
		t.Fatal("Bytes = nil, want base64 error")
	}
}

func TestService_Read_NilResult(t *testing.T) {
	t.Parallel()

	var res *files.ReadResult
	if _, err := res.Bytes(); err == nil {
		t.Fatal("Bytes = nil, want nil result error")
	}
}

func TestService_Read_RequiresInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		path     string
		encoding files.Encoding
	}{
		{name: "missing sandbox id", path: "/home/user/a"},
		{name: "missing path", id: "bx_example", encoding: files.EncodingUTF8},
		{name: "unsupported encoding", id: "bx_example", path: "/home/user/a", encoding: files.Encoding("latin1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if _, err := files.New(d).Read(context.Background(), tt.id, tt.path, tt.encoding); err == nil {
				t.Fatal("Read = nil, want validation error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("Do calls = %d, want 0", len(d.Calls))
			}
		})
	}
}
