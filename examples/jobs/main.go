// Command jobs runs one create, push, exec, usage and delete workflow. Set BOAT_API_KEY before
// running it. The command runs inside the sandbox and its output is streamed to the terminal.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ijaidev/boat-go"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/jobs"
	"github.com/ijaidev/boat-go/sandboxes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	client, err := boat.New(boat.Config{APIKey: os.Getenv("BOAT_API_KEY")})
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Jobs.Run(ctx, jobs.RunRequest{
		Create: sandboxes.CreateRequest{
			Type:       "small",
			TTLSeconds: core.Some(900),
		},
		SourceDir: ".",
		RemoteDir: "workspace",
		Command:   "cd workspace && ls -la",
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sandbox %s exited with %d", result.SandboxID, result.ExitCode)
	if result.Usage != nil {
		log.Printf("cost: $%.4f", result.Usage.Dollars)
	}
}
