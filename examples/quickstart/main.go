// Command quickstart creates a sandbox, waits until it is ready, runs one command and stops the
// sandbox. Set BOAT_API_KEY before running it.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ijaidev/boat-go"
	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/sandboxes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client, err := boat.New(boat.Config{APIKey: os.Getenv("BOAT_API_KEY")})
	if err != nil {
		log.Fatal(err)
	}

	sandbox, err := client.Sandboxes.Create(ctx, sandboxes.CreateRequest{
		TTLSeconds: core.Some(1800),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.Sandboxes.Delete(cleanupCtx, sandbox.ID); err != nil {
			log.Printf("delete sandbox: %v", err)
		}
	}()

	ready, err := client.Sandboxes.WaitUntilReady(ctx, sandbox.ID, sandboxes.WaitOptions{})
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Commands.Run(ctx, ready.ID, commands.RunRequest{
		Command: "echo hi && uname -a",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("exit %d\n%s", deref(result.ExitCode), result.Stdout)
}

func deref(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}
