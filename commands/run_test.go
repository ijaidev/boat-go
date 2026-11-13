package commands_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Run_PostsSynchronousCommand(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var gotKey string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/commands",
		Body:   `{"ok":true,"type":"command.finished","success":true,"commandId":"cmd_1","exitCode":0,"stdout":"hi\n","stderr":"","timedOut":false}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotKey = r.Header.Get("Idempotency-Key")
			body = readJSONBody(t, r)
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	out, err := svc.Run(context.Background(), "bx_example", commands.RunRequest{
		Command:        "echo hi",
		Cwd:            core.Ptr("work"),
		TimeoutSeconds: core.Ptr(120),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotKey != "" {
		t.Fatalf("Idempotency-Key = %q, want none", gotKey)
	}
	if body["command"] != "echo hi" {
		t.Fatalf("command = %v", body["command"])
	}
	if body["cwd"] != "work" {
		t.Fatalf("cwd = %v", body["cwd"])
	}
	if body["timeoutSeconds"] != float64(120) {
		t.Fatalf("timeoutSeconds = %v", body["timeoutSeconds"])
	}
	if _, ok := body["detached"]; ok {
		t.Fatalf("detached = %v, want omitted", body["detached"])
	}
	if _, ok := body["stream"]; ok {
		t.Fatalf("stream = %v, want omitted", body["stream"])
	}
	if out.CommandID == nil || *out.CommandID != "cmd_1" {
		t.Fatalf("command id = %v", out.CommandID)
	}
	if out.ExitCode == nil || *out.ExitCode != 0 {
		t.Fatalf("exit code = %v", out.ExitCode)
	}
	if out.Stdout != "hi\n" {
		t.Fatalf("stdout = %q", out.Stdout)
	}
}

func TestService_Run_DoesNotRetryOrSendIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var gotKey string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/commands",
		Status: http.StatusBadGateway,
		Body:   `{"ok":false,"type":"sandbox.error","status":502,"code":"sandbox_direct_failed","message":"sandbox_restoring","requestId":"req_1","error":{"code":"sandbox_direct_failed","message":"sandbox_restoring","retryable":true}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
			gotKey = r.Header.Get("Idempotency-Key")
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Run(context.Background(), "bx_example", commands.RunRequest{Command: "echo hi"})
	if err == nil {
		t.Fatal("Run = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
	if gotKey != "" {
		t.Fatalf("Idempotency-Key = %q, want none", gotKey)
	}
	if !core.IsCode(err, "sandbox_direct_failed") {
		t.Fatalf("error = %v, want sandbox_direct_failed", err)
	}
	if !core.Retryable(err) {
		t.Fatalf("Retryable(%v) = false, want true", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *core.APIError", err)
	}
	if apiErr.Status != http.StatusBadGateway || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestService_Run_MapsDocumentedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		code   string
	}{
		{name: "bad request", status: http.StatusBadRequest, code: "invalid_timeout"},
		{name: "unauthorized", status: http.StatusUnauthorized, code: "unauthorized"},
		{name: "billing", status: http.StatusPaymentRequired, code: "billing_required"},
		{name: "forbidden", status: http.StatusForbidden, code: "forbidden"},
		{name: "not found", status: http.StatusNotFound, code: "sandbox_not_found"},
		{name: "conflict", status: http.StatusConflict, code: "sandbox_starting"},
		{name: "too large", status: http.StatusRequestEntityTooLarge, code: "output_too_large"},
		{name: "rate limited", status: http.StatusTooManyRequests, code: "rate_limited"},
		{name: "server error", status: http.StatusInternalServerError, code: "internal"},
		{name: "unavailable", status: http.StatusServiceUnavailable, code: "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPost,
				Path:   "/sandboxes/bx_example/commands",
				Status: tt.status,
				Body:   `{"ok":false,"type":"sandbox.error","code":"` + tt.code + `","message":"nope","requestId":"req_1"}`,
			})
			svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			_, err := svc.Run(context.Background(), "bx_example", commands.RunRequest{Command: "echo hi"})
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %q", err, tt.code)
			}
			var apiErr *core.APIError
			if !errors.As(err, &apiErr) || apiErr.Status != tt.status {
				t.Fatalf("api error = %#v, want status %d", apiErr, tt.status)
			}
		})
	}
}

func TestService_Run_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := commands.New(&testutil.Doer{})
	if _, err := svc.Run(context.Background(), "", commands.RunRequest{Command: "true"}); err == nil {
		t.Fatal("Run = nil, want id error")
	}
	if _, err := svc.Start(context.Background(), "", commands.RunRequest{Command: "true"}); err == nil {
		t.Fatal("Start = nil, want id error")
	}
}

func TestService_Start_Detaches(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/commands",
		Body:   `{"ok":true,"type":"command.started","success":true,"processId":42,"pid":42,"command":"sleep 1","startedAt":"2026-01-02T03:04:05Z","commandId":"cmd_2"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readJSONBody(t, r)
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	started, err := svc.Start(context.Background(), "bx_example", commands.RunRequest{Command: "sleep 1"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if body["detached"] != true {
		t.Fatalf("detached = %v, want true", body["detached"])
	}
	if _, ok := body["stream"]; ok {
		t.Fatalf("stream = %v, want omitted", body["stream"])
	}
	if _, ok := body["timeoutSeconds"]; ok {
		t.Fatalf("timeoutSeconds = %v, want omitted", body["timeoutSeconds"])
	}
	if started.ProcessID != 42 {
		t.Fatalf("process id = %d", started.ProcessID)
	}
	if started.CommandID == nil || *started.CommandID != "cmd_2" {
		t.Fatalf("command id = %v", started.CommandID)
	}
}

func TestService_Stream_DecodesEveryFrame(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var gotAccept string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/commands",
		Header: http.Header{"Content-Type": {"application/x-ndjson"}},
		Body: strings.Join([]string{
			`{"type":"started","commandId":"cmd_1"}`,
			`{"type":"stdout","data":"hi\n"}`,
			`{"type":"stderr","data":"err\n"}`,
			`{"type":"exit","success":true,"exitCode":0,"timedOut":false}`,
		}, "\n") + "\n",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotAccept = r.Header.Get("Accept")
			body = readJSONBody(t, r)
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{
		Command:        "echo hi",
		TimeoutSeconds: core.Ptr(60),
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	if gotAccept != "application/x-ndjson" {
		t.Fatalf("Accept = %q, want application/x-ndjson", gotAccept)
	}
	if body["stream"] != true {
		t.Fatalf("stream = %v, want true", body["stream"])
	}
	if _, ok := body["detached"]; ok {
		t.Fatalf("detached = %v, want omitted", body["detached"])
	}
	if body["timeoutSeconds"] != float64(60) {
		t.Fatalf("timeoutSeconds = %v", body["timeoutSeconds"])
	}

	want := []model.CommandStreamFrameType{
		model.CommandStreamFrameTypeStarted,
		model.CommandStreamFrameTypeStdout,
		model.CommandStreamFrameTypeStderr,
		model.CommandStreamFrameTypeExit,
	}
	for i, wantType := range want {
		frame, err := stream.Next(context.Background())
		if err != nil {
			t.Fatalf("Next %d: %v", i, err)
		}
		if frame.Type != wantType {
			t.Fatalf("frame %d type = %q, want %q", i, frame.Type, wantType)
		}
		switch wantType {
		case model.CommandStreamFrameTypeStarted:
			if frame.CommandID == nil || *frame.CommandID != "cmd_1" {
				t.Fatalf("started command id = %v", frame.CommandID)
			}
		case model.CommandStreamFrameTypeStdout:
			if frame.Data == nil || *frame.Data != "hi\n" {
				t.Fatalf("stdout data = %v", frame.Data)
			}
		case model.CommandStreamFrameTypeStderr:
			if frame.Data == nil || *frame.Data != "err\n" {
				t.Fatalf("stderr data = %v", frame.Data)
			}
		case model.CommandStreamFrameTypeExit:
			if frame.ExitCode == nil || *frame.ExitCode != 0 {
				t.Fatalf("exit code = %v", frame.ExitCode)
			}
		}
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after exit = %v, want io.EOF", err)
	}
}

func TestService_Stream_ReturnsErrorFrame(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/commands",
		Header: http.Header{"Content-Type": {"application/x-ndjson"}},
		Body:   `{"type":"started"}` + "\n" + `{"type":"error","error":"sandbox_not_ready","message":"resume first","retryable":false}` + "\n",
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	if _, err := stream.Next(context.Background()); err != nil {
		t.Fatalf("started frame: %v", err)
	}
	frame, err := stream.Next(context.Background())
	if err != nil {
		t.Fatalf("error frame: %v", err)
	}
	if frame.Type != model.CommandStreamFrameTypeError {
		t.Fatalf("type = %q", frame.Type)
	}
	if frame.Error == nil || *frame.Error != "sandbox_not_ready" {
		t.Fatalf("error = %v", frame.Error)
	}
	if frame.Message == nil || *frame.Message != "resume first" {
		t.Fatalf("message = %v", frame.Message)
	}
	if frame.Retryable == nil || *frame.Retryable {
		t.Fatalf("retryable = %v, want false", frame.Retryable)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after error frame = %v, want io.EOF", err)
	}
}
