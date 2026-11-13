# Services

This folder documents the behavior and contracts of each SDK service. One service, one file.

A service is **not done** until:

- its code and tests exist,
- its doc exists here,
- it is linked from the status table below,
- and its public API is covered by at least one success and one error test.

## Status

| Service | Package | Status | Doc |
|---|---|---|---|
| Sandboxes | `sandboxes` | done | [sandboxes.md](sandboxes.md) |
| Commands | `commands` | done | [commands.md](commands.md) |
| Files | `files` | done | [files.md](files.md) |
| Agents | `agents` | done | [agents.md](agents.md) |
| Snapshots | `snapshots` | done | [snapshots.md](snapshots.md) |
| Environments | `environments` | done | [environments.md](environments.md) |
| Credentials | `credentials` | done | [credentials.md](credentials.md) |
| Account | `account` | done | [account.md](account.md) |
| Webhooks | `webhooks` | done | [webhooks.md](webhooks.md) |
| Telemetry | `telemetry` | done | [telemetry.md](telemetry.md) |
| CI | `ci` | done | [ci.md](ci.md) |
| Jobs | `jobs` | planned | — |

Status values: `planned`, `in progress`, `done`, `deprecated`.

## Doc contract

Each service doc must contain:

1. **Purpose and scope** — what the service owns.
2. **Non-goals** — what belongs elsewhere.
3. **Public API** — methods, signatures, options.
4. **Types and aliases** — public types and their `model` counterparts.
5. **Endpoints and mapping** — SDK method → HTTP endpoint.
6. **Error behavior** — expected codes, retries, terminal cases.
7. **Streaming and long-running behavior** — frames, reconnects, cancellation.
8. **Examples** — compiling examples under `examples/` where practical.
9. **Testing notes** — fixtures, edge cases, live-test coverage.
10. **Open questions / known gaps** — honest limitations.

## Rules

- Do not duplicate the official API reference. Link to it and document our behavior.
- Keep examples short; move compilable examples to `examples/`.
- Update the doc in the same change as the behavior.
- If a service is split or renamed, update this table and the package docs.
- Use the [template](_template.md) when adding a service.
