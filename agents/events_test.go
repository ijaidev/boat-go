package agents_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/ijaidev/boat-go/agents"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Events_SendsFiltersAndReadsPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/events",
		Header: http.Header{"Etag": []string{`"etag-2"`}},
		Body: `{"ok":true,"type":"events.list","id":"bx_example","events":[
			{"id":"prompt_1","type":"prompt","timestamp":1780347055370,"taskId":"prompt_1","conversationId":"conv_1","data":{"prompt":"run tests","status":"running"}},
			{"id":"response_1","type":"response","timestamp":1780347065650,"conversationId":"conv_1","data":{"content":"done"}}],
			"pageInfo":{"nextCursor":"cur_2","hasMore":true,"limit":2}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			q := r.URL.Query()
			if got := q.Get("type"); got != "prompt,response" {
				t.Fatalf("type = %q", got)
			}
			if got := q["conversation"]; len(got) != 2 || got[0] != "conv_1" || got[1] != "conv_2" {
				t.Fatalf("conversation = %v", got)
			}
			if got := q.Get("limit"); got != "2" {
				t.Fatalf("limit = %q", got)
			}
			if got := q.Get("cursor"); got != "cur_1" {
				t.Fatalf("cursor = %q", got)
			}
			if got := q.Get("sort"); got != "asc" {
				t.Fatalf("sort = %q", got)
			}
			if got := r.Header.Get("If-None-Match"); got != `"etag-1"` {
				t.Fatalf("If-None-Match = %q", got)
			}
		},
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.Events(context.Background(), "bx_example", agents.EventsRequest{
		Type:         []string{"prompt", "response"},
		Conversation: []string{"conv_1", "conv_2"},
		Limit:        2,
		Cursor:       "cur_1",
		Sort:         "asc",
		ETag:         `"etag-1"`,
	})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Type != "prompt" || page.Items[1].Type != "response" {
		t.Fatalf("items = %#v", page.Items)
	}
	if page.NextCursor != "cur_2" || !page.HasMore || page.ETag != `"etag-2"` {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_Events_NotModified(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/events",
		Status: http.StatusNotModified,
		Header: http.Header{"Etag": []string{`"etag-1"`}},
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("If-None-Match"); got != `"etag-1"` {
				t.Fatalf("If-None-Match = %q", got)
			}
		},
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.Events(context.Background(), "bx_example", agents.EventsRequest{ETag: `"etag-1"`, Sort: "asc"})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(page.Items) != 0 || page.HasMore || page.NextCursor != "" {
		t.Fatalf("page = %#v, want empty page", page)
	}
	if page.ETag != `"etag-1"` {
		t.Fatalf("etag = %q, want etag-1", page.ETag)
	}
}

func TestService_Events_AcceptsUnknownEventTypes(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/events",
		Body: `{"ok":true,"type":"events.list","id":"bx_example","events":[
			{"id":"evt_1","type":"future.event","timestamp":1780347055370,"conversationId":"conv_1",
			"data":{"somethingNew":true},"futureField":{"nested":1}}]}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.Events(context.Background(), "bx_example", agents.EventsRequest{})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %#v", page.Items)
	}
	ev := page.Items[0]
	if ev.Type != "future.event" {
		t.Fatalf("type = %q", ev.Type)
	}
	if ev.Data == nil || (*ev.Data)["somethingNew"] != true {
		t.Fatalf("data = %#v", ev.Data)
	}
	if got, ok := ev.AdditionalProperties["futureField"]; !ok {
		t.Fatalf("additional properties = %#v", ev.AdditionalProperties)
	} else if m, ok := got.(map[string]any); !ok || m["nested"] != float64(1) {
		t.Fatalf("futureField = %#v", got)
	}
}

func TestService_Events_MapsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		code   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, code: "unauthorized"},
		{name: "not found", status: http.StatusNotFound, code: "not_found"},
		{name: "payment required", status: http.StatusPaymentRequired, code: "billing_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodGet,
				Path:   "/sandboxes/bx_example/events",
				Status: tt.status,
				Body:   `{"ok":false,"type":"sandbox.error","status":` + strconv.Itoa(tt.status) + `,"code":"` + tt.code + `","message":"failed","requestId":"req_1"}`,
			})
			svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			_, err := svc.Events(context.Background(), "bx_example", agents.EventsRequest{})
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_Events_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.Events(context.Background(), "", agents.EventsRequest{}); err == nil {
		t.Fatal("Events = nil, want sandbox id error")
	}
}
