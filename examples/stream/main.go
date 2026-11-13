// Command stream runs a command with live output. Set BOAT_API_KEY and BOAT_SANDBOX_ID before
// running it.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ijaidev/boat-go"
	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/model"
)

func main() {
	ctx := context.Background()

	client, err := boat.New(boat.Config{APIKey: os.Getenv("BOAT_API_KEY")})
	if err != nil {
		log.Fatal(err)
	}

	stream, err := client.Commands.Stream(ctx, os.Getenv("BOAT_SANDBOX_ID"), commands.RunRequest{
		Command: "npm test",
		Cwd:     nil,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	for {
		frame, err := stream.Next(ctx)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			log.Fatal(err)
		}
		switch frame.Type {
		case model.CommandStreamFrameTypeStdout:
			fmt.Print(derefString(frame.Data))
		case model.CommandStreamFrameTypeStderr:
			fmt.Fprint(os.Stderr, derefString(frame.Data))
		case model.CommandStreamFrameTypeExit:
			fmt.Printf("\nexit %d\n", derefInt(frame.ExitCode))
		case model.CommandStreamFrameTypeError:
			log.Fatalf("command error: %s", derefString(frame.Message))
		}
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}
