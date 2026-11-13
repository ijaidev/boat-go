package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
)

func TestOptional_MarshalJSON_ThreeStates(t *testing.T) {
	t.Parallel()

	type request struct {
		TTLSeconds *core.Optional[int] `json:"ttlSeconds,omitempty"`
	}

	tests := []struct {
		name string
		req  request
		want string
	}{
		{name: "omitted", req: request{}, want: `{}`},
		{name: "null", req: request{TTLSeconds: core.Null[int]()}, want: `{"ttlSeconds":null}`},
		{name: "value", req: request{TTLSeconds: core.Some(3600)}, want: `{"ttlSeconds":3600}`},
		{name: "zero value", req: request{TTLSeconds: core.Some(0)}, want: `{"ttlSeconds":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Marshal = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestOptional_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	var got struct {
		TTLSeconds *core.Optional[int] `json:"ttlSeconds"`
	}
	if err := json.Unmarshal([]byte(`{"ttlSeconds":null}`), &got); err != nil {
		t.Fatalf("Unmarshal null: %v", err)
	}
	if got.TTLSeconds != nil {
		t.Fatalf("pointer null decoded as %#v, want nil", got.TTLSeconds)
	}

	if err := json.Unmarshal([]byte(`{"ttlSeconds":42}`), &got); err != nil {
		t.Fatalf("Unmarshal value: %v", err)
	}
	if got.TTLSeconds == nil || !got.TTLSeconds.Valid || got.TTLSeconds.Value != 42 {
		t.Fatalf("value decoded as %#v, want 42", got.TTLSeconds)
	}

	var direct struct {
		TTLSeconds core.Optional[int] `json:"ttlSeconds"`
	}
	if err := json.Unmarshal([]byte(`{"ttlSeconds":null}`), &direct); err != nil {
		t.Fatalf("Unmarshal direct null: %v", err)
	}
	if direct.TTLSeconds.Valid {
		t.Fatalf("direct null decoded as %#v, want invalid", direct.TTLSeconds)
	}
}

func TestOptional_Get(t *testing.T) {
	t.Parallel()

	if v, ok := core.Some("x").Get(); !ok || v != "x" {
		t.Fatalf("Some().Get() = %q, %v", v, ok)
	}
	if _, ok := core.Null[string]().Get(); ok {
		t.Fatal("Null().Get() ok = true, want false")
	}
}

func TestApply_OptionsInOrder(t *testing.T) {
	t.Parallel()

	req := &core.Request{}
	core.Apply(req,
		core.WithHeader("X-Test", "one"),
		core.WithOrg("org-a"),
		core.WithQuery("limit", "10"),
		core.WithQuery("limit", "20"),
	)
	if got := req.Header.Get("X-Test"); got != "one" {
		t.Fatalf("X-Test = %q, want one", got)
	}
	if got := req.Header.Get("X-Boat-Org"); got != "org-a" {
		t.Fatalf("X-Boat-Org = %q, want org-a", got)
	}
	if got := req.Query["limit"]; len(got) != 2 || got[0] != "10" || got[1] != "20" {
		t.Fatalf("limit = %v, want [10 20]", got)
	}
}

func TestAPIError_Error_IncludesCodeAndRequestID(t *testing.T) {
	t.Parallel()

	err := &core.APIError{
		Status:    409,
		Code:      "sandbox_not_ready",
		Message:   "sandbox is not ready",
		RequestID: "req_123",
	}
	want := "boat: http 409 sandbox_not_ready: sandbox is not ready (request req_123)"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestIsCode_Unwraps(t *testing.T) {
	t.Parallel()

	base := &core.APIError{Status: 404, Code: "not_found"}
	err := errors.Join(errors.New("context"), base)
	if !core.IsCode(err, "not_found") {
		t.Fatal("IsCode = false, want true")
	}
	if core.IsCode(err, "other") {
		t.Fatal("IsCode(other) = true, want false")
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()

	err := &core.APIError{RetryAfter: 2 * time.Second}
	if got := core.RetryAfter(err); got != 2*time.Second {
		t.Fatalf("RetryAfter = %v, want 2s", got)
	}
	if got := core.RetryAfter(errors.New("x")); got != 0 {
		t.Fatalf("RetryAfter(non-API) = %v, want 0", got)
	}
}

func TestRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "true", err: &core.APIError{Details: map[string]any{"retryable": true}}, want: true},
		{name: "false", err: &core.APIError{Details: map[string]any{"retryable": false}}, want: false},
		{name: "missing", err: &core.APIError{Details: map[string]any{}}, want: false},
		{name: "not API", err: errors.New("x"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := core.Retryable(tt.err); got != tt.want {
				t.Fatalf("Retryable = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEach_WalksPages(t *testing.T) {
	t.Parallel()

	pages := map[string]core.Page[int]{
		"":  {Items: []int{1, 2}, NextCursor: "a", HasMore: true},
		"a": {Items: []int{3}, NextCursor: "b", HasMore: true},
		"b": {Items: []int{4, 5}, HasMore: false},
	}
	var got []int
	var cursors []string
	err := core.Each(context.Background(), func(_ context.Context, cursor string) (core.Page[int], error) {
		cursors = append(cursors, cursor)
		return pages[cursor], nil
	}, func(v int) error {
		got = append(got, v)
		return nil
	})
	if err != nil {
		t.Fatalf("Each: %v", err)
	}
	if len(got) != 5 || got[0] != 1 || got[4] != 5 {
		t.Fatalf("items = %v, want [1 2 3 4 5]", got)
	}
	if len(cursors) != 3 || cursors[0] != "" || cursors[1] != "a" || cursors[2] != "b" {
		t.Fatalf("cursors = %v, want [ a b]", cursors)
	}
}

func TestEach_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	err := core.Each(context.Background(), func(context.Context, string) (core.Page[int], error) {
		return core.Page[int]{HasMore: true}, nil
	}, func(int) error { return nil })
	if err == nil {
		t.Fatal("Each = nil, want missing cursor error")
	}
}

func TestEach_RejectsRepeatedCursor(t *testing.T) {
	t.Parallel()

	err := core.Each(context.Background(), func(context.Context, string) (core.Page[int], error) {
		return core.Page[int]{NextCursor: "same", HasMore: true}, nil
	}, func(int) error { return nil })
	if err == nil {
		t.Fatal("Each = nil, want repeated cursor error")
	}
}

func TestEach_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := core.Each(ctx, func(context.Context, string) (core.Page[int], error) {
		t.Fatal("fetch called after cancel")
		return core.Page[int]{}, nil
	}, func(int) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Each = %v, want context.Canceled", err)
	}
}

func TestEach_StopsOnYieldError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stop")
	err := core.Each(context.Background(), func(context.Context, string) (core.Page[int], error) {
		return core.Page[int]{Items: []int{1}}, nil
	}, func(int) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("Each = %v, want sentinel", err)
	}
}

func TestNewIdempotencyKey_IsUUIDv4(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]struct{}, 64)
	for range 64 {
		key, err := core.NewIdempotencyKey()
		if err != nil {
			t.Fatalf("NewIdempotencyKey: %v", err)
		}
		if !pattern.MatchString(key) {
			t.Fatalf("key %q is not a UUIDv4", key)
		}
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate key %q", key)
		}
		seen[key] = struct{}{}
	}
}
