package files_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Artifact_StreamsBinary(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/artifacts",
		Header: http.Header{
			"Content-Type":   {"application/octet-stream"},
			"Content-Length": {"10"},
		},
		Body: "artifact!\n",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("path"); got != "/home/user/out.bin" {
				t.Fatalf("path = %q", got)
			}
			if got := r.Header.Get("Accept"); got != "application/octet-stream" {
				t.Fatalf("Accept = %q, want application/octet-stream", got)
			}
		},
	})

	got, err := svc.Artifact(context.Background(), "bx_example", "/home/user/out.bin")
	if err != nil {
		t.Fatalf("Artifact: %v", err)
	}
	if got.ContentType != "application/octet-stream" {
		t.Fatalf("content type = %q", got.ContentType)
	}
	if got.ContentLength != 10 {
		t.Fatalf("content length = %d, want 10", got.ContentLength)
	}
	content, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := got.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	if string(content) != "artifact!\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestService_Artifact_DoesNotBufferBody(t *testing.T) {
	t.Parallel()

	body := &trackingBody{Reader: strings.NewReader("streamed")}
	d := &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		if req.Path != "/sandboxes/bx_example/artifacts" {
			t.Fatalf("path = %q", req.Path)
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": {"application/octet-stream"}},
			Body:          body,
			ContentLength: 8,
		}, nil
	}}

	got, err := files.New(d).Artifact(context.Background(), "bx_example", "/home/user/out.bin")
	if err != nil {
		t.Fatalf("Artifact: %v", err)
	}
	if body.reads != 0 {
		t.Fatalf("reads = %d before the caller read, want 0", body.reads)
	}
	content, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(content) != "streamed" {
		t.Fatalf("content = %q", content)
	}
	if err := got.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	if body.closed != 1 {
		t.Fatalf("closes = %d, want 1", body.closed)
	}
}

func TestService_Artifact_MapsErrorAndClosesBody(t *testing.T) {
	t.Parallel()

	body := &trackingBody{Reader: strings.NewReader(
		`{"ok":false,"type":"sandbox.error","status":404,"code":"not_found","message":"artifact not found","requestId":"req_1"}`,
	)}
	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       body,
		}, nil
	}}

	got, err := files.New(d).Artifact(context.Background(), "bx_example", "/home/user/missing.bin")
	if got != nil {
		t.Fatalf("download = %#v, want nil", got)
	}
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
	if body.closed != 1 {
		t.Fatalf("closes = %d, want 1", body.closed)
	}
}

func TestService_Artifact_ClosesBodyOnTransportError(t *testing.T) {
	t.Parallel()

	body := &trackingBody{Reader: strings.NewReader("partial")}
	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       body,
		}, errors.New("connection reset")
	}}

	got, err := files.New(d).Artifact(context.Background(), "bx_example", "/home/user/out.bin")
	if err == nil {
		t.Fatal("Artifact = nil, want transport error")
	}
	if got != nil {
		t.Fatalf("download = %#v, want nil", got)
	}
	if body.closed != 1 {
		t.Fatalf("closes = %d, want 1", body.closed)
	}
}

func TestService_Artifact_RequiresPath(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	if _, err := files.New(d).Artifact(context.Background(), "bx_example", ""); err == nil {
		t.Fatal("Artifact = nil, want path error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("DoRaw calls = %d, want 0", len(d.Calls))
	}
}

type trackingBody struct {
	io.Reader
	reads  int
	closed int
}

func (b *trackingBody) Read(p []byte) (int, error) {
	b.reads++
	return b.Reader.Read(p)
}

func (b *trackingBody) Close() error {
	b.closed++
	return nil
}

var _ io.ReadCloser = (*trackingBody)(nil)
