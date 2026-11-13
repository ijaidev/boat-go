package commands_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_RunCommand_ReconnectsWithoutDuplicates(t *testing.T) {
	t.Parallel()

	const firstStream = `{"type":"attached","processId":42,"stdoutOffset":0,"stderrOffset":0}
{"type":"stdout","data":"hello","nextOffset":5}
{"type":"stderr","data":"oops","nextOffset":4}
{"type":"heartbeat"}
{"type":"reconnect","stdoutOffset":5,"stderrOffset":4}
`
	const secondStream = `{"type":"attached","processId":42,"stdoutOffset":5,"stderrOffset":4}
{"type":"stdout","data":" world","nextOffset":11}
{"type":"exit","status":"exited","exitCode":0,"signal":null,"stdoutOffset":11,"stderrOffset":4}
`

	var starts, attaches atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sandboxes/bx_example/commands":
			starts.Add(1)
			var body map[string]any
			if err := decodeJSON(r.Body, &body); err != nil {
				t.Errorf("decode start body: %v", err)
			}
			if body["detached"] != true {
				t.Errorf("detached = %v, want true", body["detached"])
			}
			if body["command"] != "echo hi" {
				t.Errorf("command = %v", body["command"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true,"type":"command.started","success":true,"processId":42,"pid":42,"command":"echo hi","startedAt":"2026-01-02T03:04:05Z","commandId":"cmd_1"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/sandboxes/bx_example/commands/42/stream":
			attaches.Add(1)
			offset := r.URL.Query().Get("stdoutOffset")
			w.Header().Set("Content-Type", "application/x-ndjson")
			if offset == "" {
				_, _ = io.WriteString(w, firstStream)
				return
			}
			if offset != "5" {
				t.Errorf("stdoutOffset = %q, want 5", offset)
			}
			if got := r.URL.Query().Get("stderrOffset"); got != "4" {
				t.Errorf("stderrOffset = %q, want 4", got)
			}
			_, _ = io.WriteString(w, secondStream)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))
	stream, err := svc.RunCommand(context.Background(), "bx_example", commands.RunRequest{Command: "echo hi"})
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	var got []commands.Event
	for {
		event, err := stream.Next(context.Background())
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		got = append(got, event)
	}

	wantTypes := []commands.EventType{
		commands.EventStarted,
		commands.EventStdout,
		commands.EventStderr,
		commands.EventStdout,
		commands.EventExit,
	}
	if len(got) != len(wantTypes) {
		t.Fatalf("events = %#v, want %d", got, len(wantTypes))
	}
	for i, want := range wantTypes {
		if got[i].Type != want {
			t.Fatalf("event %d type = %q, want %q", i, got[i].Type, want)
		}
	}
	if got[0].ProcessID != 42 || got[0].CommandID != "cmd_1" {
		t.Fatalf("started event = %#v", got[0])
	}
	if got[1].Data != "hello" || got[2].Data != "oops" || got[3].Data != " world" {
		t.Fatalf("output events = %#v", got)
	}
	if got[4].ExitCode == nil || *got[4].ExitCode != 0 || got[4].Status != model.CommandAttachFrameStatusExited {
		t.Fatalf("exit event = %#v", got[4])
	}
	if starts.Load() != 1 {
		t.Fatalf("starts = %d, want 1", starts.Load())
	}
	if attaches.Load() != 2 {
		t.Fatalf("attaches = %d, want 2", attaches.Load())
	}
}

func TestService_RunCommand_ReturnsErrorEvent(t *testing.T) {
	t.Parallel()

	started := model.CommandStartedResponse{
		ProcessID: 9,
		CommandID: core.Ptr("cmd_9"),
	}
	body := newCloseCounter(strings.NewReader(
		`{"type":"attached","processId":9}` + "\n" +
			`{"type":"error","error":"agent_outdated","message":"update the sandbox agent"}` + "\n"))
	d := &testutil.Doer{
		DoFunc: func(_ context.Context, _ *core.Request, out any, _ ...core.RequestOption) error {
			*out.(*model.CommandStartedResponse) = started
			return nil
		},
		RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
		},
	}
	svc := commands.New(d)

	stream, err := svc.RunCommand(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	event, err := stream.Next(context.Background())
	if err != nil {
		t.Fatalf("started event: %v", err)
	}
	if event.Type != commands.EventStarted || event.CommandID != "cmd_9" {
		t.Fatalf("started event = %#v", event)
	}
	event, err = stream.Next(context.Background())
	if err != nil {
		t.Fatalf("error event: %v", err)
	}
	if event.Type != commands.EventError || event.Error != "agent_outdated" || event.Message != "update the sandbox agent" {
		t.Fatalf("error event = %#v", event)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after error = %v, want io.EOF", err)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_RunCommand_StartError(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{
		DoFunc: func(context.Context, *core.Request, any, ...core.RequestOption) error {
			return &core.APIError{Status: http.StatusConflict, Code: "sandbox_not_ready", Message: "resume first"}
		},
	}
	svc := commands.New(d)

	_, err := svc.RunCommand(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if !core.IsCode(err, "sandbox_not_ready") {
		t.Fatalf("error = %v, want sandbox_not_ready", err)
	}
}

func TestService_RunCommand_AttachError(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{
		DoFunc: func(_ context.Context, _ *core.Request, out any, _ ...core.RequestOption) error {
			*out.(*model.CommandStartedResponse) = model.CommandStartedResponse{ProcessID: 9}
			return nil
		},
		RawFunc: func(context.Context, *core.Request, ...core.RequestOption) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusConflict,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"ok":false,"code":"agent_outdated","message":"update"}`)),
			}, nil
		},
	}
	svc := commands.New(d)

	_, err := svc.RunCommand(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if !core.IsCode(err, "agent_outdated") {
		t.Fatalf("error = %v, want agent_outdated", err)
	}
}
