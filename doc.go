// Package boat is an unofficial, idiomatic Go client for the Boat public API v1.
//
// Create a client with New, then use the service fields:
//
//	client, err := boat.New(boat.Config{APIKey: os.Getenv("BOAT_API_KEY")})
//	if err != nil {
//		return err
//	}
//	sandbox, err := client.Sandboxes.Create(ctx, sandboxes.CreateRequest{})
//
// Every I/O method takes a context.Context. Services are safe for concurrent use.
package boat
