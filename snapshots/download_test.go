package snapshots_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/snapshots"
)

type trackedBody struct {
	io.Reader
	closed int
}

func (b *trackedBody) Close() error {
	b.closed++
	return nil
}

func TestService_File_ReturnsBodyForCallerToClose(t *testing.T) {
	t.Parallel()

	body := &trackedBody{Reader: strings.NewReader("hello world")}
	d := &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
		core.Apply(req, opts...)
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q", req.Method)
		}
		if req.Path != "/snapshots/"+snapshotID+"/files" {
			t.Fatalf("path = %q", req.Path)
		}
		if got := req.Query.Get("path"); got != "src/main.go" {
			t.Fatalf("path query = %q", got)
		}
		if got := req.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("Accept = %q", got)
		}
		if got := req.Header.Get("Range"); got != "bytes=0-10" {
			t.Fatalf("Range = %q", got)
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": {"application/octet-stream"}},
			Body:          body,
			ContentLength: 11,
		}, nil
	}}
	svc := snapshots.New(d)

	dl, err := svc.File(context.Background(), snapshotID, "src/main.go", core.WithHeader("Range", "bytes=0-10"))
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if dl.ContentType != "application/octet-stream" {
		t.Fatalf("content type = %q", dl.ContentType)
	}
	if dl.ContentLength != 11 {
		t.Fatalf("content length = %d", dl.ContentLength)
	}
	if body.closed != 0 {
		t.Fatal("service closed the body before the caller")
	}
	got, err := io.ReadAll(dl.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("body = %q", got)
	}
	if err := dl.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	if body.closed != 1 {
		t.Fatalf("close count = %d, want 1", body.closed)
	}
}

func TestService_File_MapsErrorAndClosesBody(t *testing.T) {
	t.Parallel()

	body := &trackedBody{Reader: strings.NewReader(`{"ok":false,"type":"error","status":409,"code":"legacy_snapshot","message":"no inventory"}`)}
	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusConflict,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       body,
		}, nil
	}}
	svc := snapshots.New(d)

	_, err := svc.File(context.Background(), snapshotID, "src/main.go")
	if !core.IsCode(err, "legacy_snapshot") {
		t.Fatalf("error = %v, want legacy_snapshot", err)
	}
	if body.closed != 1 {
		t.Fatalf("close count = %d, want 1", body.closed)
	}
}

func TestService_File_MapsTransportError(t *testing.T) {
	t.Parallel()

	want := errors.New("dial failed")
	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return nil, want
	}}
	svc := snapshots.New(d)

	_, err := svc.File(context.Background(), snapshotID, "src/main.go")
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped dial failure", err)
	}
}

func TestService_File_RequiresSnapshotID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.File(context.Background(), "", "src/main.go"); err == nil {
		t.Fatal("File = nil, want id error")
	}
}

func TestService_Download_RequestsWholeSnapshot(t *testing.T) {
	t.Parallel()

	body := &trackedBody{Reader: strings.NewReader("tar bytes")}
	d := &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		if req.Path != "/snapshots/"+snapshotID+"/files" {
			t.Fatalf("path = %q", req.Path)
		}
		if _, ok := req.Query["path"]; ok {
			t.Fatalf("path query = %v, want none", req.Query["path"])
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": {"application/x-tar"}},
			Body:          body,
			ContentLength: 9,
		}, nil
	}}
	svc := snapshots.New(d)

	dl, err := svc.Download(context.Background(), snapshotID)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if dl.ContentType != "application/x-tar" || dl.ContentLength != 9 {
		t.Fatalf("download = %#v", dl)
	}
	if err := dl.Body.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestService_DownloadManifest_ReturnsChunkURLs(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/" + snapshotID + "/download",
		Body: `{"ok":true,"type":"snapshot.download","snapshotId":"` + snapshotID + `","sandboxId":"bx_example",
			"kind":"incremental","generation":3,"expiresInSeconds":3600,"reconstruct":"concat",
			"inventory":{"r2Key":"chains/x/inventory.json.zst","signedUrl":"https://r2.example/inventory?sig=test"},
			"chunks":[{"snapshotId":"22222222-2222-4222-8222-222222222222","generation":0,"chunkIndex":0,
				"r2Key":"chains/x/chunks/00.tar.zst","sizeBytes":209715200,
				"sha256":"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
				"signedUrl":"https://r2.example/chunks/00?sig=test"}]}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	manifest, err := svc.DownloadManifest(context.Background(), snapshotID)
	if err != nil {
		t.Fatalf("DownloadManifest: %v", err)
	}
	if len(manifest.Chunks) != 1 || manifest.Chunks[0].Generation != 0 {
		t.Fatalf("chunks = %#v", manifest.Chunks)
	}
	if manifest.ExpiresInSeconds != 3600 {
		t.Fatalf("expires = %d", manifest.ExpiresInSeconds)
	}
}

func TestService_DownloadManifest_MapsCompacted(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/" + snapshotID + "/download",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"error","status":409,"code":"snapshot_compacted","message":"older layers were cleaned up"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.DownloadManifest(context.Background(), snapshotID)
	if !core.IsCode(err, "snapshot_compacted") {
		t.Fatalf("error = %v, want snapshot_compacted", err)
	}
}

func TestService_DownloadManifest_RequiresSnapshotID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.DownloadManifest(context.Background(), ""); err == nil {
		t.Fatal("DownloadManifest = nil, want id error")
	}
}
