package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/internal/transport"
	"github.com/ijaidev/boat-go/snapshots"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingClient struct {
	inner  core.HTTPClient
	closes *int32
}

func (c *trackingClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.inner.Do(req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.Body != nil {
		resp.Body = &trackingBody{ReadCloser: resp.Body, closes: c.closes}
	}
	return resp, nil
}

type trackingBody struct {
	io.ReadCloser
	closes *int32
}

func (b *trackingBody) Close() error {
	atomic.AddInt32(b.closes, 1)
	return b.ReadCloser.Close()
}

func TestService_Pull_StreamsSignedDownload(t *testing.T) {
	t.Parallel()

	var downloadPath, downloadToken string
	downloadSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/artifact",
		Status: http.StatusOK,
		Body:   "artifact bytes",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("Authorization = %q, want none on a signed download", got)
			}
			if r.URL.Query().Get("token") != "secret-token" {
				t.Fatalf("token = %q, want secret-token", r.URL.Query().Get("token"))
			}
		},
	})
	signedURL := downloadSrv.URL + "/artifact?token=secret-token"

	apiSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_1/files/download-url",
		Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"GET","path":"dist/app.tar","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL),
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var req files.URLRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode download-url request: %v", err)
			}
			downloadPath = req.Path
			if req.TTLSeconds != nil {
				downloadToken = "ttl-set"
			}
		},
	})

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	var closes int32
	svc := New(Config{
		Files:      files.New(tr),
		HTTPClient: &trackingClient{inner: http.DefaultClient, closes: &closes},
	})

	var dst bytes.Buffer
	if err := svc.Pull(context.Background(), "bx_1", "dist/app.tar", &dst); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if dst.String() != "artifact bytes" {
		t.Fatalf("downloaded %q, want artifact bytes", dst.String())
	}
	if downloadPath != "dist/app.tar" {
		t.Fatalf("download path = %q, want dist/app.tar", downloadPath)
	}
	if downloadToken != "" {
		t.Fatalf("ttlSeconds = %q, want nil", downloadToken)
	}
	if got := atomic.LoadInt32(&closes); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_Pull_DownloadFailure(t *testing.T) {
	t.Parallel()

	downloadSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/artifact",
		Status: http.StatusInternalServerError,
	})
	apiSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_1/files/download-url",
		Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"GET","path":"dist/app.tar","expiresAt":"2026-11-13T00:00:00Z"}`, downloadSrv.URL+"/artifact"),
	})

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{Files: files.New(tr), HTTPClient: http.DefaultClient})

	var dst bytes.Buffer
	err := svc.Pull(context.Background(), "bx_1", "dist/app.tar", &dst)
	if err == nil {
		t.Fatal("Pull returned nil error, want download failure")
	}
	if !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("error = %q, want http 500", err)
	}
	if dst.Len() != 0 {
		t.Fatalf("destination has %d bytes, want none", dst.Len())
	}
}

func TestService_Pull_RedactsSignedURL(t *testing.T) {
	t.Parallel()

	signedURL := "https://storage.example.com/object?token=secret-token"
	apiSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_1/files/download-url",
		Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"GET","path":"dist/app.tar","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL),
	})

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	client := httpClientFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("get %q: network down", r.URL.String())
	})
	svc := New(Config{Files: files.New(tr), HTTPClient: client})

	err := svc.Pull(context.Background(), "bx_1", "dist/app.tar", io.Discard)
	if err == nil {
		t.Fatal("Pull returned nil error, want request failure")
	}
	if strings.Contains(err.Error(), signedURL) || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaks the signed URL: %q", err)
	}
}

func TestService_Pull_RequiresWriter(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if err := svc.Pull(context.Background(), "bx_1", "dist/app.tar", nil); err == nil {
		t.Fatal("Pull returned nil error, want a writer error")
	}
}

func TestService_Pull_RequiresFilesService(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if err := svc.Pull(context.Background(), "bx_1", "dist/app.tar", io.Discard); err == nil {
		t.Fatal("Pull returned nil error, want missing files service error")
	}
}

func TestService_PullSnapshot_StreamsAndCloses(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/snap_1/files",
		Body:   "snapshot file bytes",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("path"); got != "workspace/result.json" {
				t.Fatalf("path = %q, want workspace/result.json", got)
			}
		},
	})

	var closes int32
	tr, err := transport.New(core.Config{
		BaseURL:    srv.URL,
		APIKey:     "boat_test",
		HTTPClient: &trackingClient{inner: http.DefaultClient, closes: &closes},
		MaxRetries: 1,
		RetryMin:   time.Millisecond,
		RetryMax:   time.Millisecond,
	})
	if err != nil {
		t.Fatalf("transport.New: %v", err)
	}
	svc := New(Config{Snapshots: snapshots.New(tr)})

	var dst bytes.Buffer
	if err := svc.PullSnapshot(context.Background(), "snap_1", "workspace/result.json", &dst); err != nil {
		t.Fatalf("PullSnapshot: %v", err)
	}
	if dst.String() != "snapshot file bytes" {
		t.Fatalf("downloaded %q, want snapshot file bytes", dst.String())
	}
	if got := atomic.LoadInt32(&closes); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_PullSnapshot_ErrorClosesBody(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/snap_1/files",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"snapshot.error","status":404,"code":"not_found","message":"missing"}`,
	})

	var closes int32
	tr, err := transport.New(core.Config{
		BaseURL:    srv.URL,
		APIKey:     "boat_test",
		HTTPClient: &trackingClient{inner: http.DefaultClient, closes: &closes},
		MaxRetries: 1,
		RetryMin:   time.Millisecond,
		RetryMax:   time.Millisecond,
	})
	if err != nil {
		t.Fatalf("transport.New: %v", err)
	}
	svc := New(Config{Snapshots: snapshots.New(tr)})

	err = svc.PullSnapshot(context.Background(), "snap_1", "workspace/result.json", io.Discard)
	if err == nil {
		t.Fatal("PullSnapshot returned nil error, want not found")
	}
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
	if got := atomic.LoadInt32(&closes); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_PullSnapshot_RequiresWriter(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if err := svc.PullSnapshot(context.Background(), "snap_1", "file", nil); err == nil {
		t.Fatal("PullSnapshot returned nil error, want a writer error")
	}
}

func TestService_PullSnapshot_RequiresSnapshotsService(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if err := svc.PullSnapshot(context.Background(), "snap_1", "file", io.Discard); err == nil {
		t.Fatal("PullSnapshot returned nil error, want missing snapshots service error")
	}
}
