package sandboxes_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/sandboxes"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestService_Create_SendsDefaultIdempotencyKeyAndBody(t *testing.T) {
	t.Parallel()

	var gotKey string
	var gotBody map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.created","sandbox":{"id":"bx_example","name":"n","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotKey = r.Header.Get("Idempotency-Key")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(body, &gotBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	sb, err := svc.Create(context.Background(), sandboxes.CreateRequest{
		Name:       "n",
		TTLSeconds: core.Some(60),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sb.ID != "bx_example" {
		t.Fatalf("sandbox id = %q", sb.ID)
	}
	if !uuidPattern.MatchString(gotKey) {
		t.Fatalf("Idempotency-Key = %q, want UUIDv4", gotKey)
	}
	if gotBody["name"] != "n" {
		t.Fatalf("body name = %v", gotBody["name"])
	}
	if gotBody["ttlSeconds"] != float64(60) {
		t.Fatalf("body ttlSeconds = %v", gotBody["ttlSeconds"])
	}
}

func TestService_Create_OverridesIdempotencyKey(t *testing.T) {
	t.Parallel()

	var gotKey string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.created","sandbox":{"id":"bx_example","name":"n","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotKey = r.Header.Get("Idempotency-Key")
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Create(context.Background(), sandboxes.CreateRequest{}, core.WithIdempotencyKey("key-1")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotKey != "key-1" {
		t.Fatalf("Idempotency-Key = %q, want key-1", gotKey)
	}
}

func TestService_Create_ExplicitNullTTL(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.created","sandbox":{"id":"bx_example","name":"n","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &gotBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Create(context.Background(), sandboxes.CreateRequest{TTLSeconds: core.Null[int]()}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	value, ok := gotBody["ttlSeconds"]
	if !ok || value != nil {
		t.Fatalf("ttlSeconds = %v (present %v), want explicit null", value, ok)
	}
}

func TestService_Create_MapsConflict(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"sandbox.error","status":409,"code":"idempotency_in_progress","message":"in progress","requestId":"req_1"}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Create(context.Background(), sandboxes.CreateRequest{})
	if !core.IsCode(err, "idempotency_in_progress") {
		t.Fatalf("error = %v, want idempotency_in_progress", err)
	}
}

func TestService_CreateMany_ReturnsBatch(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Status: http.StatusAccepted,
		Body: `{"ok":true,"type":"sandbox.created","sandbox":{"id":"bx_1","name":"a","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false},"sandboxes":[
			{"id":"bx_1","name":"a","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false},
			{"id":"bx_2","name":"a","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false}]}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.CreateMany(context.Background(), sandboxes.CreateRequest{From: "base", Count: 2})
	if err != nil {
		t.Fatalf("CreateMany: %v", err)
	}
	if len(got) != 2 || got[0].ID != "bx_1" || got[1].ID != "bx_2" {
		t.Fatalf("sandboxes = %#v", got)
	}
}

func TestService_Get_ReturnsSandbox(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example",
		Body:   `{"ok":true,"type":"sandbox.info","sandbox":{"id":"bx_example","name":"n","state":"ready","desktopAvailable":true,"snapshotAvailable":true}}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	sb, err := svc.Get(context.Background(), "bx_example")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb.State != model.SandboxStateReady {
		t.Fatalf("state = %q", sb.State)
	}
}

func TestService_Get_RequiresID(t *testing.T) {
	t.Parallel()

	svc := sandboxes.New(&testutil.Doer{})
	if _, err := svc.Get(context.Background(), ""); err == nil {
		t.Fatal("Get = nil, want id error")
	}
}

func TestService_List_SendsFiltersAndReadsPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes",
		Body: `{"ok":true,"type":"sandbox.list","sandboxes":[
			{"id":"bx_1","name":"a","state":"ready","desktopAvailable":false,"snapshotAvailable":false}],
			"pageInfo":{"nextCursor":"cur_2","hasMore":true,"limit":1}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			q := r.URL.Query()
			if got := q.Get("limit"); got != "1" {
				t.Fatalf("limit = %q", got)
			}
			if got := q.Get("state"); got != "ready,idle" {
				t.Fatalf("state = %q", got)
			}
			if got := q.Get("sort"); got != "asc" {
				t.Fatalf("sort = %q", got)
			}
			if got := q.Get("include"); got != "cost" {
				t.Fatalf("include = %q", got)
			}
			if got := q["label"]; len(got) != 2 || got[0] != "env=prod" || got[1] != "team=a" {
				t.Fatalf("label = %v", got)
			}
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.List(context.Background(), sandboxes.ListRequest{
		Limit:   1,
		Sort:    "asc",
		State:   []string{"ready", "idle"},
		Label:   []string{"env=prod", "team=a"},
		Include: []string{"cost"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 || page.NextCursor != "cur_2" || !page.HasMore {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_List_EmptyPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes",
		Body:   `{"ok":true,"type":"sandbox.list","sandboxes":[]}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.List(context.Background(), sandboxes.ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 0 || page.HasMore || page.NextCursor != "" {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_Update_SendsTTLStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  *core.Optional[int]
		want any
	}{
		{name: "omitted", ttl: nil, want: nil},
		{name: "null", ttl: core.Null[int](), want: nil},
		{name: "value", ttl: core.Some(7200), want: float64(7200)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var body map[string]any
			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPatch,
				Path:   "/sandboxes/bx_example",
				Body:   `{"ok":true,"type":"sandbox.info","sandbox":{"id":"bx_example","name":"n","state":"ready","desktopAvailable":false,"snapshotAvailable":false}}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					raw, _ := io.ReadAll(r.Body)
					if err := json.Unmarshal(raw, &body); err != nil {
						t.Fatalf("decode body: %v", err)
					}
				},
			})
			svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			if _, err := svc.Update(context.Background(), "bx_example", sandboxes.UpdateRequest{TTLSeconds: tt.ttl}); err != nil {
				t.Fatalf("Update: %v", err)
			}
			value, ok := body["ttlSeconds"]
			if tt.name == "omitted" {
				if ok {
					t.Fatalf("ttlSeconds present = %v, want omitted", value)
				}
				return
			}
			if !ok || value != tt.want {
				t.Fatalf("ttlSeconds = %v (present %v), want %v", value, ok, tt.want)
			}
		})
	}
}

func TestService_Stop_SendsForceAndReturnsOperation(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/stop",
		Body:   `{"ok":true,"type":"sandbox.stopping","id":"bx_example","status":"archiving","operation":{"id":"stop_1","status":"pending"}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	op, err := svc.Stop(context.Background(), "bx_example", sandboxes.StopOptions{Force: core.Ptr(true)})
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if op.ID != "stop_1" {
		t.Fatalf("stop id = %q", op.ID)
	}
	if body["force"] != true {
		t.Fatalf("body force = %v", body["force"])
	}
}

func TestService_GetStop(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/stops/stop_1",
		Body:   `{"ok":true,"type":"sandbox.stop","id":"bx_example","stop":{"id":"stop_1","status":"completed"}}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	op, err := svc.GetStop(context.Background(), "bx_example", "stop_1")
	if err != nil {
		t.Fatalf("GetStop: %v", err)
	}
	if op.Status != model.StopOperationStatusCompleted {
		t.Fatalf("status = %q", op.Status)
	}
}

func TestService_Resume_ReturnsSandbox(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/resume",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.resuming","id":"bx_example","status":"resuming","sandbox":{"id":"bx_example","name":"n","state":"provisioning","desktopAvailable":false,"snapshotAvailable":true}}`,
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	sb, err := svc.Resume(context.Background(), "bx_example", sandboxes.ResumeRequest{})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if sb.ID != "bx_example" {
		t.Fatalf("id = %q", sb.ID)
	}
}

func TestService_Fork_SendsIdempotencyKey(t *testing.T) {
	t.Parallel()

	var gotKey string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/fork",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.forking","id":"bx_new","status":"forking","sandbox":{"id":"bx_new","name":"n","state":"provisioning","desktopAvailable":false,"snapshotAvailable":false}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			gotKey = r.Header.Get("Idempotency-Key")
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	sb, err := svc.Fork(context.Background(), "bx_example", sandboxes.ForkRequest{})
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if sb.ID != "bx_new" {
		t.Fatalf("id = %q", sb.ID)
	}
	if !uuidPattern.MatchString(gotKey) {
		t.Fatalf("Idempotency-Key = %q, want UUIDv4", gotKey)
	}
}

func TestService_Delete_SendsConfirmationHeader(t *testing.T) {
	t.Parallel()

	var got string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/sandboxes/bx_example",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"sandbox.deleted","operation":{"id":"del_1","status":"pending"}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			got = r.Header.Get("X-Ascii-Confirm-Delete")
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	op, err := svc.Delete(context.Background(), "bx_example")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got != "bx_example" {
		t.Fatalf("confirm header = %q", got)
	}
	if op.ID != "del_1" {
		t.Fatalf("operation id = %q", op.ID)
	}
}

func TestService_Usage_SendsWindow(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	until := since.Add(time.Hour)
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/usage",
		Body:   `{"ok":true,"type":"sandbox.usage","sandboxId":"bx_example","seconds":60,"dollars":0.001,"running":false,"billingMultiplier":1}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("since"); got != "2026-01-02T03:04:05Z" {
				t.Fatalf("since = %q", got)
			}
			if got := r.URL.Query().Get("until"); got != "2026-01-02T04:04:05Z" {
				t.Fatalf("until = %q", got)
			}
		},
	})
	svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	usage, err := svc.Usage(context.Background(), "bx_example", sandboxes.UsageRequest{Since: &since, Until: &until})
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if usage.SandboxID != "bx_example" || usage.Seconds != 60 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestService_WaitUntilReady_Polls(t *testing.T) {
	t.Parallel()

	states := []model.SandboxState{model.SandboxStateProvisioning, model.SandboxStateReady}
	var calls int
	d := &testutil.Doer{DoFunc: func(_ context.Context, _ *core.Request, out any, _ ...core.RequestOption) error {
		state := states[calls]
		if calls < len(states)-1 {
			calls++
		}
		*out.(*model.SandboxInfoResponse) = model.SandboxInfoResponse{
			Sandbox: model.Sandbox{ID: "bx_example", State: state},
		}
		return nil
	}}
	svc := sandboxes.New(d)

	sb, err := svc.WaitUntilReady(context.Background(), "bx_example", sandboxes.WaitOptions{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("WaitUntilReady: %v", err)
	}
	if sb.State != model.SandboxStateReady {
		t.Fatalf("state = %q", sb.State)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 2", calls+1)
	}
}

func TestService_WaitUntilReady_TerminalState(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{DoFunc: func(_ context.Context, _ *core.Request, out any, _ ...core.RequestOption) error {
		*out.(*model.SandboxInfoResponse) = model.SandboxInfoResponse{
			Sandbox: model.Sandbox{ID: "bx_example", State: model.SandboxStateError},
		}
		return nil
	}}
	svc := sandboxes.New(d)

	_, err := svc.WaitUntilReady(context.Background(), "bx_example", sandboxes.WaitOptions{PollInterval: time.Millisecond})
	if err == nil {
		t.Fatal("WaitUntilReady = nil, want terminal state error")
	}
}

func TestService_WaitUntilReady_ContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := &testutil.Doer{DoFunc: func(ctx context.Context, _ *core.Request, _ any, _ ...core.RequestOption) error {
		return ctx.Err()
	}}
	svc := sandboxes.New(d)

	_, err := svc.WaitUntilReady(ctx, "bx_example", sandboxes.WaitOptions{PollInterval: time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
