package agents_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/agents"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func eventResponse(status int, etag, body string) *http.Response {
	header := make(http.Header)
	if etag != "" {
		header.Set("ETag", etag)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestService_StreamEvents_FollowsCursor(t *testing.T) {
	t.Parallel()

	pages := []string{
		`{"ok":true,"type":"events.list","id":"bx_example","events":[
			{"id":"e1","type":"prompt","timestamp":1},
			{"id":"e2","type":"response","timestamp":2}],
			"pageInfo":{"nextCursor":"c1","hasMore":false,"limit":100}}`,
		`{"ok":true,"type":"events.list","id":"bx_example","events":[
			{"id":"e3","type":"response","timestamp":3}],
			"pageInfo":{"nextCursor":"c2","hasMore":false,"limit":100}}`,
	}
	var d *testutil.Doer
	d = &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		call := len(d.Calls) - 1
		if req.Method != http.MethodGet || req.Path != "/sandboxes/bx_example/events" {
			t.Errorf("request = %s %s", req.Method, req.Path)
		}
		if got := req.Query.Get("sort"); got != "asc" {
			t.Errorf("sort = %q, want asc", got)
		}
		switch call {
		case 0:
			if req.Query.Get("cursor") != "" || req.Header.Get("If-None-Match") != "" {
				t.Errorf("first call cursor/etag = %q/%q", req.Query.Get("cursor"), req.Header.Get("If-None-Match"))
			}
			return eventResponse(http.StatusOK, `"t1"`, pages[0]), nil
		case 1:
			if got := req.Query.Get("cursor"); got != "c1" {
				t.Errorf("cursor = %q, want c1", got)
			}
			if got := req.Header.Get("If-None-Match"); got != `"t1"` {
				t.Errorf("If-None-Match = %q, want t1", got)
			}
			return eventResponse(http.StatusOK, `"t2"`, pages[1]), nil
		default:
			t.Errorf("unexpected call %d", call)
			return eventResponse(http.StatusNotModified, `"t2"`, ""), nil
		}
	}}
	svc := agents.New(d)

	stream, err := svc.StreamEvents(context.Background(), "bx_example", agents.EventsRequest{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	for _, want := range []string{"e1", "e2", "e3"} {
		ev, err := stream.Next(context.Background())
		if err != nil {
			t.Fatalf("Next(%s): %v", want, err)
		}
		if ev.ID == nil || *ev.ID != want {
			t.Fatalf("event = %#v, want id %s", ev, want)
		}
	}
	if len(d.Calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(d.Calls))
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after Close = %v, want io.EOF", err)
	}
}

func TestService_StreamEvents_FetchesNextPageWithoutWaiting(t *testing.T) {
	t.Parallel()

	pages := []string{
		`{"ok":true,"type":"events.list","id":"bx_example","events":[{"id":"e1","type":"prompt","timestamp":1}],
			"pageInfo":{"nextCursor":"c1","hasMore":true,"limit":1}}`,
		`{"ok":true,"type":"events.list","id":"bx_example","events":[{"id":"e2","type":"response","timestamp":2}],
			"pageInfo":{"nextCursor":"c2","hasMore":false,"limit":1}}`,
	}
	var d *testutil.Doer
	d = &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		call := len(d.Calls) - 1
		if call == 1 && req.Query.Get("cursor") != "c1" {
			t.Errorf("cursor = %q, want c1", req.Query.Get("cursor"))
		}
		if call >= len(pages) {
			t.Errorf("unexpected call %d", call)
			return eventResponse(http.StatusNotModified, "", ""), nil
		}
		return eventResponse(http.StatusOK, "", pages[call]), nil
	}}
	svc := agents.New(d)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// A one-hour poll interval proves the second page is fetched without waiting.
	stream, err := svc.StreamEvents(ctx, "bx_example", agents.EventsRequest{PollInterval: time.Hour})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = stream.Close() }()

	for _, want := range []string{"e1", "e2"} {
		ev, err := stream.Next(ctx)
		if err != nil {
			t.Fatalf("Next(%s): %v", want, err)
		}
		if ev.ID == nil || *ev.ID != want {
			t.Fatalf("event = %#v, want id %s", ev, want)
		}
	}
}

func TestService_StreamEvents_NotModifiedWaitsAndHonorsCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var d *testutil.Doer
	d = &testutil.Doer{RawFunc: func(_ context.Context, req *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		call := len(d.Calls) - 1
		switch call {
		case 0:
			return eventResponse(http.StatusOK, `"t1"`, `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
		case 1:
			if got := req.Header.Get("If-None-Match"); got != `"t1"` {
				t.Errorf("If-None-Match = %q, want t1", got)
			}
			cancel()
			return eventResponse(http.StatusNotModified, `"t1"`, ""), nil
		default:
			t.Errorf("unexpected call %d", call)
			return eventResponse(http.StatusNotModified, "", ""), nil
		}
	}}
	svc := agents.New(d)

	stream, err := svc.StreamEvents(ctx, "bx_example", agents.EventsRequest{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = stream.Close() }()

	if _, err := stream.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next = %v, want context.Canceled", err)
	}
}

func TestService_StreamEvents_CancelWhileWaiting(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
	}}
	svc := agents.New(d)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	stream, err := svc.StreamEvents(ctx, "bx_example", agents.EventsRequest{PollInterval: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = stream.Close() }()

	if _, err := stream.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Next = %v, want context.DeadlineExceeded", err)
	}
	if len(d.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(d.Calls))
	}
}

func TestService_StreamEvents_BacksOffOnEmptyPages(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
	}}
	svc := agents.New(d)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	stream, err := svc.StreamEvents(ctx, "bx_example", agents.EventsRequest{PollInterval: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = stream.Close() }()

	if _, err := stream.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Next = %v, want context.DeadlineExceeded", err)
	}
	// With backoff (20ms, 40ms, ...) a 120ms window allows a handful of polls, never a spin.
	if got := len(d.Calls); got < 2 || got > 8 {
		t.Fatalf("calls = %d, want 2..8", got)
	}
}

func TestService_StreamEvents_FirstFetchError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("transport down")
	d := &testutil.Doer{RawFunc: func(context.Context, *core.Request, ...core.RequestOption) (*http.Response, error) {
		return nil, sentinel
	}}
	svc := agents.New(d)

	_, err := svc.StreamEvents(context.Background(), "bx_example", agents.EventsRequest{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want transport down", err)
	}
}

func TestService_StreamEvents_CloseBeforeNext(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
	}}
	svc := agents.New(d)

	stream, err := svc.StreamEvents(context.Background(), "bx_example", agents.EventsRequest{})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next = %v, want io.EOF", err)
	}
	if len(d.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(d.Calls))
	}
}

func TestService_StreamEvents_CloseUnblocksNext(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
	}}
	svc := agents.New(d)

	stream, err := svc.StreamEvents(context.Background(), "bx_example", agents.EventsRequest{PollInterval: time.Hour})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := stream.Next(context.Background())
		result <- err
	}()

	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("Next = %v, want io.EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Next did not return after Close")
	}
}

func TestService_StreamEvents_ConstructorContextCancel(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[]}`), nil
	}}
	svc := agents.New(d)

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := svc.StreamEvents(ctx, "bx_example", agents.EventsRequest{PollInterval: time.Hour})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	t.Cleanup(func() { _ = stream.Close() })

	cancel()
	if _, err := stream.Next(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next = %v, want context.Canceled", err)
	}
}

func TestService_StreamEvents_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.StreamEvents(context.Background(), "", agents.EventsRequest{}); err == nil {
		t.Fatal("StreamEvents = nil, want sandbox id error")
	}
}

func TestService_StreamEvents_SkipsUnknownEventTypes(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{RawFunc: func(_ context.Context, _ *core.Request, _ ...core.RequestOption) (*http.Response, error) {
		return eventResponse(http.StatusOK, "", `{"ok":true,"type":"events.list","id":"bx_example","events":[
			{"id":"e1","type":"future.event","timestamp":1,"extra":{"a":1}}],
			"pageInfo":{"nextCursor":"c1","hasMore":false,"limit":100}}`), nil
	}}
	svc := agents.New(d)

	stream, err := svc.StreamEvents(context.Background(), "bx_example", agents.EventsRequest{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = stream.Close() }()

	ev, err := stream.Next(context.Background())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if ev.Type != "future.event" || ev.AdditionalProperties["extra"] == nil {
		t.Fatalf("event = %#v", ev)
	}
}
