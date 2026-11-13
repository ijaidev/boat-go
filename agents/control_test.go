package agents_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/agents"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Conversations_ListsSandboxConversations(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/conversations",
		Body: `{"ok":true,"type":"conversation.list","id":"bx_example","conversations":[
			{"id":"conv_1","createdAt":"2026-09-08T10:05:00Z","lastPromptAt":"2026-09-08T10:05:00Z",
			 "prompts":1,"running":true,"lastHarness":"pi","lastModel":"claude-sonnet-5",
			 "lastPromptPreview":"Investigate the flaky CI job","current":true},
			{"id":"conv_2","createdAt":"2026-09-08T09:40:00Z","lastPromptAt":"2026-09-08T10:03:00Z",
			 "prompts":2,"running":false,"lastHarness":"claude","lastModel":null,
			 "lastPromptPreview":"Now add tests","current":false}]}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Conversations(context.Background(), "bx_example")
	if err != nil {
		t.Fatalf("Conversations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("conversations = %#v", got)
	}
	if got[0].ID != "conv_1" || !got[0].Current || !got[0].Running || got[0].Prompts != 1 {
		t.Fatalf("first conversation = %#v", got[0])
	}
	if got[1].LastModel != nil {
		t.Fatalf("last model = %#v, want nil", got[1].LastModel)
	}
}

func TestService_Conversations_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.Conversations(context.Background(), ""); err == nil {
		t.Fatal("Conversations = nil, want sandbox id error")
	}
}

func TestService_Steer_SendsMessage(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/steer",
		Body: `{"ok":true,"type":"prompt.steered","id":"bx_example","conversationId":"conv_1",
			"promptId":"prompt_456","native":true,"mode":"native","status":"steered"}`,
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

	res, err := svc.Steer(context.Background(), "bx_example", agents.SteerRequest{
		Conversation: "conv_1",
		Message:      "skip the integration tests",
	})
	if err != nil {
		t.Fatalf("Steer: %v", err)
	}
	if res.PromptID != "prompt_456" || !res.Native || res.Mode == nil || *res.Mode != model.SteerResponseModeNative {
		t.Fatalf("response = %#v", res)
	}
	if got["conversation"] != "conv_1" || got["message"] != "skip the integration tests" {
		t.Fatalf("body = %#v", got)
	}
}

func TestService_Steer_NoRunningTurn(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/steer",
		Status: http.StatusConflict,
		Body: `{"ok":false,"type":"sandbox.error","status":409,"code":"no_running_turn","message":"nothing is running",
			"requestId":"req_3","error":{"code":"no_running_turn","message":"nothing is running","status":409,
			"details":{"reason":"turn_ended","lastTurn":{"status":"finished"}}}}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Steer(context.Background(), "bx_example", agents.SteerRequest{Message: "more"})
	if !core.IsCode(err, "no_running_turn") {
		t.Fatalf("error = %v, want no_running_turn", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *core.APIError", err)
	}
	if got := apiErr.Details["reason"]; got != "turn_ended" {
		t.Fatalf("reason = %v", got)
	}
}

func TestService_Steer_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.Steer(context.Background(), "", agents.SteerRequest{}); err == nil {
		t.Fatal("Steer = nil, want sandbox id error")
	}
}

func TestService_Interrupt_ScopesConversation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		conversation string
		wantQuery    string
		responseID   string
	}{
		{name: "one conversation", conversation: "conv_1", wantQuery: "conv_1", responseID: "conv_1"},
		{name: "whole sandbox", conversation: "", wantQuery: "", responseID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPost,
				Path:   "/sandboxes/bx_example/interrupt",
				Body: `{"ok":true,"type":"sandbox.interrupted","id":"bx_example","status":"interrupted",
					"conversationId":` + quoteOrNull(tt.responseID) + `}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					if got := r.URL.Query().Get("conversation"); got != tt.wantQuery {
						t.Fatalf("conversation = %q, want %q", got, tt.wantQuery)
					}
					if r.URL.Query().Has("conversation") != (tt.wantQuery != "") {
						t.Fatalf("conversation presence = %v", r.URL.Query().Has("conversation"))
					}
				},
			})
			svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			res, err := svc.Interrupt(context.Background(), "bx_example", tt.conversation)
			if err != nil {
				t.Fatalf("Interrupt: %v", err)
			}
			if res.Type != "sandbox.interrupted" || res.Status != "interrupted" {
				t.Fatalf("response = %#v", res)
			}
			if tt.wantQuery == "" && res.ConversationID != nil {
				t.Fatalf("conversation id = %q, want nil", *res.ConversationID)
			}
		})
	}
}

func TestService_Interrupt_Conflict(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_example/interrupt",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"sandbox.error","status":409,"code":"interrupt_failed","message":"failed","requestId":"req_4"}`,
	})
	svc := agents.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Interrupt(context.Background(), "bx_example", "")
	if !core.IsCode(err, "interrupt_failed") {
		t.Fatalf("error = %v, want interrupt_failed", err)
	}
}

func TestService_Interrupt_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := agents.New(&testutil.Doer{})
	if _, err := svc.Interrupt(context.Background(), "", ""); err == nil {
		t.Fatal("Interrupt = nil, want sandbox id error")
	}
}

func quoteOrNull(s string) string {
	if s == "" {
		return "null"
	}
	return `"` + s + `"`
}
