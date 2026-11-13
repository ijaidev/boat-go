package files_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_UploadURL_ReturnsSignedFields(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/upload-url",
		Body:   `{"ok":true,"type":"file.upload_url","success":true,"url":"https://upload.example.com/signed?token=test-redacted","method":"PUT","path":"/home/user/big.bin","expiresAt":"2026-01-02T03:19:05Z"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.UploadURL(context.Background(), "bx_example", files.URLRequest{
		Path:       "/home/user/big.bin",
		TTLSeconds: core.Ptr(3600),
	})
	if err != nil {
		t.Fatalf("UploadURL: %v", err)
	}
	if body["path"] != "/home/user/big.bin" || body["ttlSeconds"] != float64(3600) {
		t.Fatalf("body = %v", body)
	}
	if got.URL != "https://upload.example.com/signed?token=test-redacted" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.Method != model.PUT {
		t.Fatalf("method = %q, want PUT", got.Method)
	}
	if got.Type != model.FileUploadURL {
		t.Fatalf("type = %q, want file.upload_url", got.Type)
	}
	if got.Path != "/home/user/big.bin" {
		t.Fatalf("path = %q", got.Path)
	}
	if want := time.Date(2026, 1, 2, 3, 19, 5, 0, time.UTC); !got.ExpiresAt.Equal(want) {
		t.Fatalf("expiresAt = %v, want %v", got.ExpiresAt, want)
	}
}

func TestService_UploadURL_OmitsTTL(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/upload-url",
		Body:   `{"ok":true,"type":"file.upload_url","success":true,"url":"https://upload.example.com/signed?token=test-redacted","method":"PUT","path":"/home/user/a.bin","expiresAt":"2026-01-02T03:19:05Z"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	if _, err := svc.UploadURL(context.Background(), "bx_example", files.URLRequest{Path: "/home/user/a.bin"}); err != nil {
		t.Fatalf("UploadURL: %v", err)
	}
	if _, ok := body["ttlSeconds"]; ok {
		t.Fatalf("ttlSeconds = %v, want omitted", body["ttlSeconds"])
	}
}

func TestService_DownloadURL_ReturnsSignedFields(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/files/download-url",
		Body:   `{"ok":true,"type":"file.download_url","success":true,"url":"https://download.example.com/signed?token=test-redacted","method":"GET","path":"/home/user/report.pdf","expiresAt":"2026-01-02T03:19:05Z"}`,
	})

	got, err := svc.DownloadURL(context.Background(), "bx_example", files.URLRequest{Path: "/home/user/report.pdf"})
	if err != nil {
		t.Fatalf("DownloadURL: %v", err)
	}
	if got.Method != model.GET {
		t.Fatalf("method = %q, want GET", got.Method)
	}
	if got.Type != model.FileDownloadURL {
		t.Fatalf("type = %q, want file.download_url", got.Type)
	}
	if got.URL != "https://download.example.com/signed?token=test-redacted" || got.Path != "/home/user/report.pdf" {
		t.Fatalf("response = %#v", got)
	}
}

func TestService_URLs_RequirePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *files.Service) error
	}{
		{
			name: "upload missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.UploadURL(ctx, "bx_example", files.URLRequest{})
				return err
			},
		},
		{
			name: "download missing path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.DownloadURL(ctx, "bx_example", files.URLRequest{})
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
