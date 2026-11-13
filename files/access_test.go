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
	"github.com/ijaidev/boat-go/model"
)

func TestService_Host_SendsPortOptions(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/host",
		Body:   `{"ok":true,"type":"port.hosted","success":true,"port":3000,"url":"https://sandbox-3000.example.com/?_token=test-redacted","isProtected":true,"access":"private"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Host(context.Background(), "bx_example", files.HostRequest{
		Port:      3000,
		Public:    core.Ptr(true),
		Localhost: core.Ptr(true),
		Title:     "preview",
	})
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if body["port"] != float64(3000) || body["public"] != true || body["localhost"] != true || body["title"] != "preview" {
		t.Fatalf("body = %v", body)
	}
	if got.URL == nil || *got.URL != "https://sandbox-3000.example.com/?_token=test-redacted" {
		t.Fatalf("url = %v", got.URL)
	}
	if got.Access == nil || *got.Access != model.HostPortResponseAccessPrivate {
		t.Fatalf("access = %v", got.Access)
	}
	if got.IsProtected == nil || !*got.IsProtected {
		t.Fatalf("isProtected = %v", got.IsProtected)
	}
	if got.Port == nil || *got.Port != 3000 {
		t.Fatalf("port = %v", got.Port)
	}
}

func TestService_Host_RequiresValidPort(t *testing.T) {
	t.Parallel()

	for _, port := range []int{-1, 0, 65536} {
		d := &testutil.Doer{}
		if _, err := files.New(d).Host(context.Background(), "bx_example", files.HostRequest{Port: port}); err == nil {
			t.Fatalf("Host(port=%d) = nil, want validation error", port)
		}
		if len(d.Calls) != 0 {
			t.Fatalf("Do calls = %d, want 0", len(d.Calls))
		}
	}
}

func TestService_Desktop_SendsVNCAndTheme(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/desktop",
		Body:   `{"ok":true,"type":"desktop.url","success":true,"desktopUrl":"https://desktop.example.com/vnc.html?token=test-redacted","ip":"203.0.113.10","mode":"vnc"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("vnc"); got != "1" {
				t.Fatalf("vnc = %q, want 1", got)
			}
			if got := query.Get("theme"); got != "dark" {
				t.Fatalf("theme = %q, want dark", got)
			}
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.Desktop(context.Background(), "bx_example", files.DesktopRequest{
		VNC:          true,
		Theme:        files.DesktopThemeDark,
		PublicAccess: core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("Desktop: %v", err)
	}
	if body["publicAccess"] != true {
		t.Fatalf("body = %v", body)
	}
	if got.DesktopURL == nil || *got.DesktopURL != "https://desktop.example.com/vnc.html?token=test-redacted" {
		t.Fatalf("desktop url = %v", got.DesktopURL)
	}
	if got.Mode == nil || *got.Mode != "vnc" {
		t.Fatalf("mode = %v", got.Mode)
	}
}

func TestService_Desktop_ProvisionsWithoutURL(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/desktop",
		Body:   `{"ok":true,"type":"desktop.provisioning","provisioning":true,"message":"Preparing VNC desktop"}`,
	})

	got, err := svc.Desktop(context.Background(), "bx_example", files.DesktopRequest{VNC: true})
	if err != nil {
		t.Fatalf("Desktop: %v", err)
	}
	if got.Provisioning == nil || !*got.Provisioning || got.DesktopURL != nil {
		t.Fatalf("response = %#v", got)
	}
}

func TestService_Desktop_RejectsUnknownTheme(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	if _, err := files.New(d).Desktop(context.Background(), "bx_example", files.DesktopRequest{Theme: files.DesktopTheme("blue")}); err == nil {
		t.Fatal("Desktop = nil, want theme error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("Do calls = %d, want 0", len(d.Calls))
	}
}

func TestService_SSHKey_ConfiguresKey(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/sshkey",
		Body:   `{"ok":true,"type":"ssh_key.configured","success":true,"machineIp":"203.0.113.10","sshUser":"user","hostKey":"ssh-ed25519 AAAAexampleHostKey","sshEndpoint":"203.0.113.10:22001"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})

	got, err := svc.SSHKey(context.Background(), "bx_example", files.SSHKeyRequest{
		Key: "ssh-ed25519 AAAAexamplePublicKey user@example.com",
	})
	if err != nil {
		t.Fatalf("SSHKey: %v", err)
	}
	if body["key"] != "ssh-ed25519 AAAAexamplePublicKey user@example.com" {
		t.Fatalf("body = %v", body)
	}
	if got.MachineIP == nil || *got.MachineIP != "203.0.113.10" {
		t.Fatalf("machine ip = %v", got.MachineIP)
	}
	if got.SSHUser == nil || *got.SSHUser != "user" {
		t.Fatalf("ssh user = %v", got.SSHUser)
	}
	if got.SSHEndpoint == nil || *got.SSHEndpoint != "203.0.113.10:22001" {
		t.Fatalf("ssh endpoint = %v", got.SSHEndpoint)
	}
	if got.HostKey == nil || *got.HostKey != "ssh-ed25519 AAAAexampleHostKey" {
		t.Fatalf("host key = %v", got.HostKey)
	}
}

func TestService_SSHKey_RequiresKey(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	if _, err := files.New(d).SSHKey(context.Background(), "bx_example", files.SSHKeyRequest{}); err == nil {
		t.Fatal("SSHKey = nil, want key error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("Do calls = %d, want 0", len(d.Calls))
	}
}
