package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/sandboxes"
	"github.com/ijaidev/boat-go/snapshots"
)

const (
	sandboxCreatedBody = `{"ok":true,"type":"sandbox.created","sandbox":{"id":"bx_1","state":"provisioning","name":"test"},"status":"provisioning","ttlSeconds":null}`
	sandboxReadyBody   = `{"ok":true,"type":"sandbox.info","sandbox":{"id":"bx_1","state":"ready","name":"test"}}`
	commandStartedBody = `{"ok":true,"type":"command.started","command":"echo hi","pid":123,"processId":123,"startedAt":"2026-11-13T00:00:00Z","success":true}`
	usageBody          = `{"ok":true,"type":"sandbox.usage","sandboxId":"bx_1","billingMultiplier":1,"dollars":0.001,"running":false,"sandboxType":"default","seconds":10,"secondsPerDollar":100000,"since":"2026-11-13T00:00:00Z","until":"2026-11-13T00:01:00Z"}`
	deletionBody       = `{"ok":true,"type":"deletion.operation","operation":{"id":"del_1","kind":"sandbox","reason":"user","status":"pending","attemptCount":0,"completedAt":null,"requestedAt":"2026-11-13T00:00:00Z","targetId":"bx_1"}}`
	serverErrorBody    = `{"ok":false,"type":"sandbox.error","status":500,"code":"internal","message":"boom"}`
)

const successFrames = `{"type":"attached","processId":123}
{"type":"heartbeat","processId":123}
{"type":"stdout","processId":123,"data":"hello\n","nextOffset":6}
{"type":"stderr","processId":123,"data":"warn\n","nextOffset":5}
{"type":"exit","processId":123,"exitCode":0,"status":"exited"}
`

const nonZeroFrames = `{"type":"attached","processId":123}
{"type":"exit","processId":123,"exitCode":7,"signal":"SIGTERM","status":"failed"}
`

const errorFrames = `{"type":"attached","processId":123}
{"type":"stderr","processId":123,"data":"oops\n","nextOffset":5}
{"type":"error","processId":123,"error":"sandbox_direct_failed","message":"command failed"}
`

type runCalls struct {
	mu      sync.Mutex
	order   []string
	deletes int
}

func (c *runCalls) add(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.order = append(c.order, name)
	if name == "delete" {
		c.deletes++
	}
}

func (c *runCalls) sequence() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.order...)
}

func (c *runCalls) deleteCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deletes
}

func (c *runCalls) record(name string) func(*testing.T, *http.Request) {
	return func(t *testing.T, _ *http.Request) {
		t.Helper()
		c.add(name)
	}
}

type runServerConfig struct {
	createStatus int
	waitStatus   int
	usageStatus  int
	deleteStatus int
	frames       string
	command      string
}

func newRunServer(t *testing.T, cfg runServerConfig) (*httptest.Server, *runCalls) {
	t.Helper()

	if cfg.createStatus == 0 {
		cfg.createStatus = http.StatusAccepted
	}
	if cfg.waitStatus == 0 {
		cfg.waitStatus = http.StatusOK
	}
	if cfg.usageStatus == 0 {
		cfg.usageStatus = http.StatusOK
	}
	if cfg.deleteStatus == 0 {
		cfg.deleteStatus = http.StatusOK
	}
	if cfg.frames == "" {
		cfg.frames = successFrames
	}
	if cfg.command == "" {
		cfg.command = "echo hi"
	}

	calls := &runCalls{}
	bodyFor := func(status int, ok string) string {
		if status >= http.StatusBadRequest {
			return serverErrorBody
		}
		return ok
	}

	srv := testutil.NewServer(t,
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes",
			Status: cfg.createStatus,
			Body:   bodyFor(cfg.createStatus, sandboxCreatedBody),
			Assert: calls.record("create"),
		},
		testutil.Route{
			Method: http.MethodGet,
			Path:   "/sandboxes/bx_1",
			Status: cfg.waitStatus,
			Body:   bodyFor(cfg.waitStatus, sandboxReadyBody),
			Assert: calls.record("wait"),
		},
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes/bx_1/commands",
			Status: http.StatusAccepted,
			Body:   commandStartedBody,
			Assert: func(t *testing.T, r *http.Request) {
				t.Helper()
				calls.add("start")
				var body struct {
					Command        string  `json:"command"`
					Detached       bool    `json:"detached"`
					Cwd            *string `json:"cwd"`
					TimeoutSeconds *int    `json:"timeoutSeconds"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode command request: %v", err)
				}
				if body.Command != cfg.command {
					t.Fatalf("command = %q, want %q", body.Command, cfg.command)
				}
				if !body.Detached {
					t.Fatal("detached = false, want true")
				}
				if body.TimeoutSeconds != nil {
					t.Fatalf("timeoutSeconds = %d, want no time limit", *body.TimeoutSeconds)
				}
			},
		},
		testutil.Route{
			Method: http.MethodGet,
			Path:   "/sandboxes/bx_1/commands/123/stream",
			Header: http.Header{"Content-Type": []string{"application/x-ndjson"}},
			Body:   cfg.frames,
			Assert: calls.record("attach"),
		},
		testutil.Route{
			Method: http.MethodGet,
			Path:   "/sandboxes/bx_1/usage",
			Status: cfg.usageStatus,
			Body:   bodyFor(cfg.usageStatus, usageBody),
			Assert: calls.record("usage"),
		},
		testutil.Route{
			Method: http.MethodDelete,
			Path:   "/sandboxes/bx_1",
			Status: cfg.deleteStatus,
			Body:   bodyFor(cfg.deleteStatus, deletionBody),
			Assert: calls.record("delete"),
		},
	)
	return srv, calls
}

func newRunService(t *testing.T, srv *httptest.Server) *Service {
	t.Helper()
	tr := testutil.Transport(t, srv.URL, "boat_test")
	return New(Config{
		Sandboxes:  sandboxes.New(tr),
		Commands:   commands.New(tr),
		Files:      files.New(tr),
		Snapshots:  snapshots.New(tr),
		HTTPClient: http.DefaultClient,
	})
}

func TestService_Run_Success(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{})
	svc := newRunService(t, srv)

	var stdout, stderr bytes.Buffer
	res, err := svc.Run(context.Background(), RunRequest{
		Create:  sandboxes.CreateRequest{},
		Command: "echo hi",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res == nil {
		t.Fatal("Run returned a nil result")
	}
	if res.SandboxID != "bx_1" {
		t.Fatalf("SandboxID = %q, want bx_1", res.SandboxID)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.Signal != "" {
		t.Fatalf("Signal = %q, want empty", res.Signal)
	}
	if res.Pushed {
		t.Fatal("Pushed = true, want false")
	}
	if res.Usage == nil || res.Usage.Seconds != 10 {
		t.Fatalf("Usage = %+v, want 10 seconds", res.Usage)
	}
	if res.UsageError != nil {
		t.Fatalf("UsageError = %v, want nil", res.UsageError)
	}
	if stdout.String() != "hello\n" {
		t.Fatalf("stdout = %q, want hello\\n", stdout.String())
	}
	if stderr.String() != "warn\n" {
		t.Fatalf("stderr = %q, want warn\\n", stderr.String())
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}

	want := []string{"create", "wait", "start", "attach", "usage", "delete"}
	if got := calls.sequence(); !equalStrings(got, want) {
		t.Fatalf("call order = %v, want %v", got, want)
	}
}

func TestService_Run_PushesSourceDir(t *testing.T) {
	t.Parallel()

	sourceDir := newPushRepo(t)

	var uploaded []byte
	uploadSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/upload",
		Status: http.StatusOK,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = body
		},
	})
	signedURL := uploadSrv.URL + "/upload"

	var sawExtract, sawStart bool
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sandboxes":
			writeJSON(w, http.StatusAccepted, sandboxCreatedBody)
		case r.Method == http.MethodGet && r.URL.Path == "/sandboxes/bx_1":
			writeJSON(w, http.StatusOK, sandboxReadyBody)
		case r.Method == http.MethodPost && r.URL.Path == "/sandboxes/bx_1/files/upload-url":
			writeJSON(w, http.StatusOK, fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"PUT","path":".boat-push-test.tgz","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL))
		case r.Method == http.MethodPost && r.URL.Path == "/sandboxes/bx_1/commands":
			var req commands.RunRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode command request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if strings.HasPrefix(req.Command, "mkdir -p ") {
				sawExtract = true
				writeJSON(w, http.StatusOK, commandResultBody)
				return
			}
			sawStart = true
			if req.Command != "echo hi" {
				t.Errorf("command = %q, want echo hi", req.Command)
			}
			writeJSON(w, http.StatusAccepted, commandStartedBody)
		case r.Method == http.MethodGet && r.URL.Path == "/sandboxes/bx_1/commands/123/stream":
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, successFrames)
		case r.Method == http.MethodGet && r.URL.Path == "/sandboxes/bx_1/usage":
			writeJSON(w, http.StatusOK, usageBody)
		case r.Method == http.MethodDelete && r.URL.Path == "/sandboxes/bx_1":
			writeJSON(w, http.StatusOK, deletionBody)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(apiSrv.Close)

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{
		Sandboxes:  sandboxes.New(tr),
		Commands:   commands.New(tr),
		Files:      files.New(tr),
		Snapshots:  snapshots.New(tr),
		HTTPClient: http.DefaultClient,
	})

	res, err := svc.Run(context.Background(), RunRequest{
		Command:   "echo hi",
		SourceDir: sourceDir,
		RemoteDir: "workspace",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Pushed {
		t.Fatal("Pushed = false, want true")
	}
	if !sawExtract {
		t.Fatal("extract command was not run")
	}
	if !sawStart {
		t.Fatal("detached command was not started")
	}
	if len(uploaded) == 0 {
		t.Fatal("upload body is empty")
	}
	if entries := readTarArchive(t, uploaded); len(entries) == 0 {
		t.Fatal("uploaded archive has no entries")
	}
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func TestService_Run_NonZeroExitIsResult(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{frames: nonZeroFrames, command: "exit 7"})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "exit 7"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExitCode != 7 {
		t.Fatalf("ExitCode = %d, want 7", res.ExitCode)
	}
	if res.Signal != "SIGTERM" {
		t.Fatalf("Signal = %q, want SIGTERM", res.Signal)
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}
}

func TestService_Run_CommandErrorCleanupPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		keep        bool
		keepOnError bool
		wantDeletes int
	}{
		{name: "delete on error", wantDeletes: 1},
		{name: "keep on error", keepOnError: true, wantDeletes: 0},
		{name: "keep always", keep: true, wantDeletes: 0},
		{name: "keep always wins", keep: true, keepOnError: true, wantDeletes: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, calls := newRunServer(t, runServerConfig{frames: errorFrames, command: "false"})
			svc := newRunService(t, srv)

			res, err := svc.Run(context.Background(), RunRequest{
				Command:     "false",
				Keep:        tt.keep,
				KeepOnError: tt.keepOnError,
			})
			if err == nil {
				t.Fatal("Run returned nil error, want command failure")
			}
			if !strings.Contains(err.Error(), "command failed") {
				t.Fatalf("error = %q, want command failure detail", err)
			}
			if res == nil || res.SandboxID != "bx_1" {
				t.Fatalf("result = %+v, want partial result with SandboxID", res)
			}
			if got := calls.deleteCount(); got != tt.wantDeletes {
				t.Fatalf("delete calls = %d, want %d", got, tt.wantDeletes)
			}
		})
	}
}

func TestService_Run_KeepOnSuccess(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "echo hi", Keep: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res == nil || res.SandboxID != "bx_1" {
		t.Fatalf("result = %+v, want SandboxID", res)
	}
	if calls.deleteCount() != 0 {
		t.Fatalf("delete calls = %d, want 0", calls.deleteCount())
	}
}

func TestService_Run_UsageFailureIsBestEffort(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{usageStatus: http.StatusInternalServerError})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Usage != nil {
		t.Fatalf("Usage = %+v, want nil", res.Usage)
	}
	if res.UsageError == nil {
		t.Fatal("UsageError = nil, want the usage failure")
	}
	var apiErr *core.APIError
	if !errors.As(res.UsageError, &apiErr) || apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("UsageError = %v, want API 500", res.UsageError)
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}
}

func TestService_Run_CleanupErrorJoinsPrimaryError(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{
		frames:       errorFrames,
		command:      "false",
		deleteStatus: http.StatusInternalServerError,
	})
	svc := newRunService(t, srv)

	_, err := svc.Run(context.Background(), RunRequest{Command: "false"})
	if err == nil {
		t.Fatal("Run returned nil error, want joined errors")
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Fatalf("error = %q, want command failure", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want cleanup failure", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("error = %v, want API 500 from cleanup", err)
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}
}

func TestService_Run_CleanupSuccessErrorIsReturned(t *testing.T) {
	t.Parallel()

	srv, _ := newRunServer(t, runServerConfig{deleteStatus: http.StatusInternalServerError})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"})
	if err == nil {
		t.Fatal("Run returned nil error, want cleanup failure")
	}
	if res == nil || res.SandboxID != "bx_1" {
		t.Fatalf("result = %+v, want partial result", res)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("error = %v, want API 500", err)
	}
}

func TestService_Run_CreateFailureDoesNotDelete(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{createStatus: http.StatusInternalServerError})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"})
	if err == nil {
		t.Fatal("Run returned nil error, want create failure")
	}
	if res != nil {
		t.Fatalf("result = %+v, want nil", res)
	}
	if calls.deleteCount() != 0 {
		t.Fatalf("delete calls = %d, want 0", calls.deleteCount())
	}
}

func TestService_Run_WaitFailureCleansUp(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{waitStatus: http.StatusInternalServerError})
	svc := newRunService(t, srv)

	res, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"})
	if err == nil {
		t.Fatal("Run returned nil error, want wait failure")
	}
	if res == nil || res.SandboxID != "bx_1" {
		t.Fatalf("result = %+v, want partial result", res)
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}
}

func TestService_Run_NilWriters(t *testing.T) {
	t.Parallel()

	srv, _ := newRunServer(t, runServerConfig{})
	svc := newRunService(t, srv)

	if _, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestService_CleanupSandboxUsesFreshContext(t *testing.T) {
	t.Parallel()

	srv, calls := newRunServer(t, runServerConfig{})
	svc := newRunService(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.cleanupSandbox(ctx, "bx_1"); err != nil {
		t.Fatalf("cleanupSandbox: %v", err)
	}
	if calls.deleteCount() != 1 {
		t.Fatalf("delete calls = %d, want 1", calls.deleteCount())
	}
}

func TestService_Run_RequiresCommand(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if _, err := svc.Run(context.Background(), RunRequest{}); err == nil {
		t.Fatal("Run returned nil error, want missing command error")
	}
}

func TestService_Run_RequiresServices(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if _, err := svc.Run(context.Background(), RunRequest{Command: "echo hi"}); err == nil {
		t.Fatal("Run returned nil error, want missing service error")
	}
	if _, err := svc.Run(context.Background(), RunRequest{
		Command:   "echo hi",
		SourceDir: ".",
	}); err == nil {
		t.Fatal("Run returned nil error, want missing files service error")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
