<img src="assets/favicon.svg" width="64" height="64" alt="Boat Go SDK">

# boat-go

An unofficial, idiomatic Go SDK for the [Boat](https://docs.boat.dev/) public API v1.

The SDK is in the design phase. Start with the rulebook and architecture:

- [Agent and contributor rules](AGENTS.md)
- [Architecture](docs/architecture.md)
- [Testing and TDD](docs/testing.md)
- [Service docs](docs/services/README.md)
- [Boat API reference](https://docs.boat.dev/api/v1)

## Status

No Go module or SDK code exists yet. The first milestone is `core`, `model`, `internal/transport`,
`sandboxes`, `commands` and the root facade.

## Development principles

- Test first: failing test, then implementation.
- Small files and domain-scoped packages.
- Concrete services; small interfaces only at seams.
- Generated models, hand-written public API.
- No live API calls in normal tests.
