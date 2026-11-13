<img src="assets/favicon.svg" width="64" height="64" alt="Boat Go SDK">

# boat-go

An unofficial, idiomatic Go SDK for the [Boat](https://docs.boat.dev/) public API v1.

```bash
go get github.com/ijaidev/boat-go
```

```go
client, err := boat.New(boat.Config{APIKey: os.Getenv("BOAT_API_KEY")})
if err != nil {
	return err
}

sb, err := client.Sandboxes.Create(ctx, sandboxes.CreateRequest{
	TTLSeconds: core.Some(3600),
})
if err != nil {
	return err
}
_, err = client.Sandboxes.WaitUntilReady(ctx, sb.ID, sandboxes.WaitOptions{})
```

## Status

The SDK is under active development. The kernel, transport, generated `model` package, root facade
and the `account`, `agents`, `ci`, `commands`, `credentials`, `environments`, `files`, `sandboxes`,
`snapshots`, `telemetry` and `webhooks` services exist today. `jobs` is next.

The API surface is intentionally small and concrete. There is no CLI, server or generated client
leaking into the public API.

## Design

- Context-first, errors-not-panics, concrete services with small interfaces at seams.
- Generated `model` package; hand-written ergonomic services and request types.
- Test-first, `httptest` and fake transports, no live API calls in normal tests.
- Cursor pagination, NDJSON streams and webhook verification designed in from the start.

Start with the rulebook and architecture:

- [Agent and contributor rules](AGENTS.md)
- [Architecture](docs/architecture.md)
- [Testing and TDD](docs/testing.md)
- [Service docs](docs/services/README.md)
- [Boat API reference](https://docs.boat.dev/api/v1)
