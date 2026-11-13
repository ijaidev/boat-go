package agents_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/agents"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Prompt_SendsBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  agents.PromptRequest
		want map[string]any
	}{
		{
			name: "continue conversation",
			req: agents.PromptRequest{
				Provider:        model.Codex,
				Model:           "gpt-5.4",
				ReasoningEffort: "medium",
				Fast:            core.Ptr(false),
				Prompt:          "run the tests",
				ConversationID:  "conv_1",
			},
			want: map[string]any{
				"provider":        "codex",
				"model":           "gpt-5.4",
				"reasoningEffort": "medium",
				"fast":            false,
				"prompt":          "run the tests",
				"conversationId":  "conv_1",
			},
		},
		{
			name: "new conversation",
			req: agents.PromptRequest{
				Provider: model.Claude,
				Prompt:   "start a second task",
				New:      true,
			},
			want: map[string]any{
				"provider": "claude",
				"prompt":   "start a second task",
				"new":      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got map[string]any
			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPost,
				Path:   "/sandboxes/bx_example/prompt",
				Status: http.StatusAccepted,
				Body: `{"ok":true,"type":"prompt.queued","id":"bx_example","promptId":"prompt_1",
					"conversationId":"conv_1",
					"promptRun":{"id":"prompt_1","promptId":"prompt_1","sandboxId":"bx_example","status":"queued","done":false,"conversationId":"conv_1"},
					"status":"queued","provider":"codex","model":"gpt-5.4","reasoningEffort":"medium","fast":false}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("read body: %v", err)
					}
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Fatalf("decode body: %v", err)
					}
				},
			})
			svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			res, err := svc.Prompt(context.Background(), "bx_example", tt.req)
			if err != nil {
				t.Fatalf("Prompt: %v", err)
			}
			if res.PromptID != "prompt_1" || res.PromptRun.Status != model.PromptRunStatusQueued {
				t.Fatalf("response = %#v", res)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("body = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestService_Prompt_ProviderNotConfigured(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/prompt",
		Status: http.StatusConflict,
		Body: `{"ok":false,"type":"sandbox.error","status":409,"code":"provider_not_configured",
			"message":"Prompting is locked until Codex is configured.","requestId":"req_1",
			"error":{"code":"provider_not_configured","message":"Prompting is locked until Codex is configured.","status":409,
			"details":{"setupUrl":"https://example.com/agents/setup"}}}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Prompt(context.Background(), "bx_example", agents.PromptRequest{
		Provider: model.Codex,
		Prompt:   "run the tests",
	})
	if !core.IsCode(err, "provider_not_configured") {
		t.Fatalf("error = %v, want provider_not_configured", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *core.APIError", err)
	}
	if apiErr.Status != http.StatusConflict || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
	if got := apiErr.Details["setupUrl"]; got != "https://example.com/agents/setup" {
		t.Fatalf("setupUrl = %v", got)
	}
}

func TestService_Prompt_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.Prompt(context.Background(), "", agents.PromptRequest{}); err == nil {
		t.Fatal("Prompt = nil, want sandbox id error")
	}
}

func TestService_PromptStatus_ReturnsRun(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/prompts/prompt_1",
		Body: `{"ok":true,"type":"prompt.run","id":"bx_example",
			"promptRun":{"id":"prompt_1","promptId":"prompt_1","sandboxId":"bx_example","status":"running","done":false,"conversationId":"conv_1"}}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	run, err := svc.PromptStatus(context.Background(), "bx_example", "prompt_1")
	if err != nil {
		t.Fatalf("PromptStatus: %v", err)
	}
	if run.Status != model.PromptRunStatusRunning || run.Done || run.ConversationID == nil || *run.ConversationID != "conv_1" {
		t.Fatalf("run = %#v", run)
	}
}

func TestService_PromptStatus_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/prompts/missing",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"sandbox.error","status":404,"code":"not_found","message":"prompt not found","requestId":"req_2"}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.PromptStatus(context.Background(), "bx_example", "missing")
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}

func TestService_PromptStatus_RequiresPromptID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.PromptStatus(context.Background(), "bx_example", ""); err == nil {
		t.Fatal("PromptStatus = nil, want prompt id error")
	}
}

func TestService_WaitForPrompt_PollsUntilDone(t *testing.T) {
	t.Parallel()

	runs := []model.PromptRun{
		{ID: "prompt_1", PromptID: "prompt_1", SandboxID: "bx_example", Status: model.PromptRunStatusRunning},
		{ID: "prompt_1", PromptID: "prompt_1", SandboxID: "bx_example", Status: model.PromptRunStatusFinished, Done: true},
	}
	var calls int
	d := &testutil.Doer{DoFunc: func(_ context.Context, req *core.Request, out any, _ ...core.RequestOption) error {
		if req.Method != http.MethodGet || req.Path != "/sandboxes/bx_example/prompts/prompt_1" {
			t.Errorf("request = %s %s", req.Method, req.Path)
		}
		*out.(*model.PromptRunResponse) = model.PromptRunResponse{PromptRun: runs[calls]}
		calls++
		return nil
	}}
	svc := agents.New(d)

	run, err := svc.WaitForPrompt(context.Background(), "bx_example", "prompt_1", agents.WaitOptions{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("WaitForPrompt: %v", err)
	}
	if !run.Done || run.Status != model.PromptRunStatusFinished {
		t.Fatalf("run = %#v", run)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestService_WaitForPrompt_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := &testutil.Doer{DoFunc: func(ctx context.Context, _ *core.Request, _ any, _ ...core.RequestOption) error {
		return ctx.Err()
	}}
	svc := agents.New(d)

	_, err := svc.WaitForPrompt(ctx, "bx_example", "prompt_1", agents.WaitOptions{PollInterval: time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
