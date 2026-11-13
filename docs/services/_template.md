# <Service>

Status: planned | in progress | done | deprecated

Owner: <name or team>

Package: `<package>`

## 1. Purpose and scope

What this service owns. Which Boat API resources it wraps. What a caller can do with it.

## 2. Non-goals

What belongs in another service or in `jobs/`. List the tempting things that must not be added here.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `Create` | `Create(ctx, CreateRequest, ...core.RequestOption) (*model.X, error)` | |
| `Get` | `Get(ctx, id) (*model.X, error)` | |
| `List` | `List(ctx, ListRequest) (core.Page[model.X], error)` | |

Document options, defaults and concurrency safety.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `CreateRequest` | `model.CreateXRequest` | Alias |
| `X` | `model.X` | Alias |

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Create` | `POST` | `/x` | 202; async |
| `Get` | `GET` | `/x/{id}` | |
| `Delete` | `DELETE` | `/x/{id}` | Requires confirmation header |

## 6. Error behavior

- Which error codes are expected.
- Which are retryable.
- Which are terminal.
- Any idempotency requirements.
- Any special mapping (for example `sandbox_not_ready` → resume first).

## 7. Streaming and long-running behavior

- Frame types, if any.
- Reconnect semantics, if any.
- Cancellation and `Close()` behavior.
- Timeouts and defaults.

## 8. Examples

Link to a compiling example under `examples/`, or show a short fragment.

```go
// ...
```

## 9. Testing notes

- Fixtures in `testdata/`.
- Error cases covered.
- Streaming or concurrency cases.
- Live-test coverage, if any.

## 10. Open questions / known gaps

- Unsupported endpoints.
- Spec ambiguities.
- Planned changes.
