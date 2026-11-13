//go:build live

// Package live contains opt-in tests that call the real Boat API.
//
// Run them with:
//
//	BOAT_API_KEY=boat_... go test -tags=live ./live
//
// They create the smallest sandbox with a short TTL and always delete it. They cost money.
package live_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ijaidev/boat-go"
	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/sandboxes"
)

func TestLive_ReadOnly(t *testing.T) {
	key := os.Getenv("BOAT_API_KEY")
	if key == "" {
		t.Skip("BOAT_API_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := boat.New(boat.Config{APIKey: key})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.Account.Me(ctx); err != nil {
		t.Fatalf("Me: %v", err)
	}
	if _, err := client.Account.Limits(ctx, account.LimitsRequest{}); err != nil {
		t.Fatalf("Limits: %v", err)
	}
	if _, err := client.Sandboxes.List(ctx, sandboxes.ListRequest{Limit: 1}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestLive_SandboxCreateRunDelete(t *testing.T) {
	key := os.Getenv("BOAT_API_KEY")
	if key == "" {
		t.Skip("BOAT_API_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := boat.New(boat.Config{APIKey: key})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sandbox, err := client.Sandboxes.Create(ctx, sandboxes.CreateRequest{
		Type:       "small",
		TTLSeconds: core.Some(120),
	})
	if err != nil {
		if core.IsCode(err, "billing_required") {
			t.Skipf("account cannot create sandboxes (no active plan): %v", err)
		}
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.Sandboxes.Delete(cleanupCtx, sandbox.ID); err != nil {
			t.Logf("delete sandbox %s: %v", sandbox.ID, err)
		}
	})

	ready, err := client.Sandboxes.WaitUntilReady(ctx, sandbox.ID, sandboxes.WaitOptions{
		PollInterval: 2 * time.Second,
		Timeout:      3 * time.Minute,
	})
	if err != nil {
		t.Fatalf("WaitUntilReady(%s): %v", sandbox.ID, err)
	}

	result, err := client.Commands.Run(ctx, ready.ID, commands.RunRequest{
		Command:        "echo boat-live-test",
		TimeoutSeconds: core.Ptr(30),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "boat-live-test") {
		t.Fatalf("stdout = %q, want boat-live-test", result.Stdout)
	}

	usage, err := client.Sandboxes.Usage(ctx, ready.ID, sandboxes.UsageRequest{})
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if usage.SandboxID != ready.ID {
		t.Fatalf("usage sandbox = %q, want %q", usage.SandboxID, ready.ID)
	}
}
