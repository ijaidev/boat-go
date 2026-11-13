package commands_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Stream_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	d, body := streamDoer(http.StatusOK, strings.NewReader(`{"type":"started"}`+"\n"))
	svc := commands.New(d)

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_Stream_EOFClosesBodyOnce(t *testing.T) {
	t.Parallel()

	d, body := streamDoer(http.StatusOK, strings.NewReader(`{"type":"started"}`+"\n"))
	svc := commands.New(d)

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if _, err := stream.Next(context.Background()); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after end = %v, want io.EOF", err)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close after EOF: %v", err)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes after Close = %d, want 1", got)
	}
}

func TestService_Stream_MalformedFrameIsTerminal(t *testing.T) {
	t.Parallel()

	d, body := streamDoer(http.StatusOK, strings.NewReader(`{"type":"started"}`+"\nnot-json\n"+`{"type":"exit"}`+"\n"))
	svc := commands.New(d)

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if _, err := stream.Next(context.Background()); err != nil {
		t.Fatalf("first Next: %v", err)
	}
	_, firstErr := stream.Next(context.Background())
	if firstErr == nil {
		t.Fatal("Next = nil, want decode error")
	}
	if !strings.Contains(firstErr.Error(), "frame 2") {
		t.Fatalf("error = %v, want frame number", firstErr)
	}
	_, secondErr := stream.Next(context.Background())
	if secondErr == nil || secondErr.Error() != firstErr.Error() {
		t.Fatalf("error after failure = %v, want %v", secondErr, firstErr)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_Stream_OpenMapsAPIError(t *testing.T) {
	t.Parallel()

	d, body := streamDoer(http.StatusConflict, strings.NewReader(`{"ok":false,"type":"sandbox.error","status":409,"code":"sandbox_not_ready","message":"resume first","requestId":"req_1"}`))
	svc := commands.New(d)

	_, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if !core.IsCode(err, "sandbox_not_ready") {
		t.Fatalf("error = %v, want sandbox_not_ready", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
	if got := body.CloseCount(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestService_Stream_ContextCancellationIsPrompt(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	signals := newSignalingReader(reader)
	d, _ := streamDoer(http.StatusOK, signals)
	svc := commands.New(d)

	stream, err := svc.Stream(context.Background(), "bx_example", commands.RunRequest{Command: "true"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := stream.Next(ctx)
		result <- err
	}()

	select {
	case <-signals.started:
	case <-time.After(time.Second):
		t.Fatal("read did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Next = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Next did not return after cancellation")
	}
}

func TestService_Stream_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := commands.New(&testutil.Doer{})
	if _, err := svc.Stream(context.Background(), "", commands.RunRequest{Command: "true"}); err == nil {
		t.Fatal("Stream = nil, want id error")
	}
}

func TestService_Attach_DecodesEveryFrameAndSendsOffsets(t *testing.T) {
	t.Parallel()

	var gotAccept string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/commands/77/stream",
		Header: http.Header{"Content-Type": {"application/x-ndjson"}},
		Body: strings.Join([]string{
			`{"type":"attached","processId":77,"stdoutOffset":10,"stderrOffset":20}`,
			`{"type":"stdout","data":"out","nextOffset":13}`,
			`{"type":"stderr","data":"err","nextOffset":23}`,
			`{"type":"heartbeat"}`,
			`{"type":"reconnect","stdoutOffset":13,"stderrOffset":23}`,
			`{"type":"exit","status":"exited","exitCode":0,"stdoutOffset":13,"stderrOffset":23}`,
		}, "\n") + "\n",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotAccept = r.Header.Get("Accept")
			requireQuery(t, r, "stdoutOffset", "10")
			requireQuery(t, r, "stderrOffset", "20")
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	stream, err := svc.Attach(context.Background(), "bx_example", 77, commands.AttachRequest{StdoutOffset: 10, StderrOffset: 20})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	if gotAccept != "application/x-ndjson" {
		t.Fatalf("Accept = %q, want application/x-ndjson", gotAccept)
	}

	want := []model.CommandAttachFrameType{
		model.CommandAttachFrameTypeAttached,
		model.CommandAttachFrameTypeStdout,
		model.CommandAttachFrameTypeStderr,
		model.CommandAttachFrameTypeHeartbeat,
		model.CommandAttachFrameTypeReconnect,
		model.CommandAttachFrameTypeExit,
	}
	for i, wantType := range want {
		frame, err := stream.Next(context.Background())
		if err != nil {
			t.Fatalf("Next %d: %v", i, err)
		}
		if frame.Type != wantType {
			t.Fatalf("frame %d type = %q, want %q", i, frame.Type, wantType)
		}
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after exit = %v, want io.EOF", err)
	}
}

func TestService_Attach_ReturnsErrorFrame(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/commands/77/stream",
		Header: http.Header{"Content-Type": {"application/x-ndjson"}},
		Body:   `{"type":"error","error":"agent_outdated","message":"update the sandbox agent","stdoutOffset":0,"stderrOffset":0}` + "\n",
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotQuery = r.URL.RawQuery
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	stream, err := svc.Attach(context.Background(), "bx_example", 77, commands.AttachRequest{})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	if gotQuery != "" {
		t.Fatalf("query = %q, want empty", gotQuery)
	}
	frame, err := stream.Next(context.Background())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if frame.Type != model.CommandAttachFrameTypeError {
		t.Fatalf("type = %q", frame.Type)
	}
	if frame.Error == nil || *frame.Error != "agent_outdated" {
		t.Fatalf("error = %v", frame.Error)
	}
	if frame.Message == nil || *frame.Message != "update the sandbox agent" {
		t.Fatalf("message = %v", frame.Message)
	}
	if frame.StdoutOffset == nil || *frame.StdoutOffset != 0 {
		t.Fatalf("stdout offset = %v", frame.StdoutOffset)
	}
}

func TestService_Attach_RejectsInvalidArguments(t *testing.T) {
	t.Parallel()

	svc := commands.New(&testutil.Doer{})
	tests := []struct {
		name      string
		sandboxID string
		processID int
		req       commands.AttachRequest
	}{
		{name: "missing sandbox", sandboxID: "", processID: 1},
		{name: "missing process", sandboxID: "bx_example", processID: 0},
		{name: "negative stdout offset", sandboxID: "bx_example", processID: 1, req: commands.AttachRequest{StdoutOffset: -1}},
		{name: "negative stderr offset", sandboxID: "bx_example", processID: 1, req: commands.AttachRequest{StderrOffset: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := svc.Attach(context.Background(), tt.sandboxID, tt.processID, tt.req); err == nil {
				t.Fatal("Attach = nil, want error")
			}
		})
	}
}
