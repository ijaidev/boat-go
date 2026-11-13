# Sandboxes

Status: done

Owner: SDK maintainers

Package: `sandboxes`

## 1. Purpose and scope

Wraps the Boat sandbox lifecycle: list, create, read, update, stop, resume, fork, share, unshare,
recover, delete and usage. A sandbox is the unit every other service hangs off.

## 2. Non-goals

- Command execution, files, agents and snapshots. Those live in their own services.
- Create → push → exec → usage → delete flows. Those belong in `jobs/`.
- Waiting for a specific agent turn or hydration. `WaitUntilReady` only waits for the API state.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `Create` | `Create(ctx, CreateRequest, ...core.RequestOption) (*model.Sandbox, error)` | Generated Idempotency-Key by default |
| `CreateMany` | `CreateMany(ctx, CreateRequest, ...core.RequestOption) ([]model.Sandbox, error)` | Batch from a saved snapshot |
| `Get` | `Get(ctx, id, ...core.RequestOption) (*model.Sandbox, error)` | |
| `List` | `List(ctx, ListRequest, ...core.RequestOption) (core.Page[model.Sandbox], error)` | Cursor pagination |
| `Update` | `Update(ctx, id, UpdateRequest, ...core.RequestOption) (*model.Sandbox, error)` | Name, TTL, subdomain |
| `Stop` | `Stop(ctx, id, StopOptions, ...core.RequestOption) (*model.StopOperation, error)` | Stop is archive, not pause |
| `GetStop` | `GetStop(ctx, id, stopID, ...core.RequestOption) (*model.StopOperation, error)` | |
| `Resume` | `Resume(ctx, id, ResumeRequest, ...core.RequestOption) (*model.Sandbox, error)` | Uses the newest snapshot |
| `Fork` | `Fork(ctx, id, ForkRequest, ...core.RequestOption) (*model.Sandbox, error)` | Generated Idempotency-Key by default |
| `Delete` | `Delete(ctx, id, ...core.RequestOption) (*model.DeletionOperation, error)` | Sends `X-Ascii-Confirm-Delete` |
| `Share` | `Share(ctx, id, ...core.RequestOption) (*ShareResult, error)` | |
| `Unshare` | `Unshare(ctx, id, ...core.RequestOption) (*UnshareResult, error)` | |
| `Recover` | `Recover(ctx, id, ...core.RequestOption) (*RecoverResult, error)` | Deprecated |
| `Usage` | `Usage(ctx, id, UsageRequest, ...core.RequestOption) (*model.SandboxUsageResponse, error)` | Also works after stop |
| `WaitUntilReady` | `WaitUntilReady(ctx, id, WaitOptions) (*model.Sandbox, error)` | Polls until ready, idle or running |

Services are safe for concurrent use. Request options apply per call; `core.WithOrg` overrides the
client organization and `core.WithIdempotencyKey` overrides the generated key.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `CreateRequest` | — | Hand-written for the nullable `ttlSeconds` |
| `ResumeRequest` | — | `TTLSeconds *core.Optional[int]` |
| `ForkRequest` | — | `TTLSeconds *core.Optional[int]` |
| `UpdateRequest` | — | `TTLSeconds *core.Optional[int]` |
| `CreateFile` | — | Inline file written before ready |
| `ShareResult` | — | Wraps the inline share response |
| `UnshareResult` | — | Wraps the inline unshare response |
| `RecoverResult` | — | Wraps the inline recover response |

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `List` | `GET` | `/sandboxes` | `limit`, `cursor`, `sort`, `state`, `name`, `namePrefix`, `stoppedBefore`, `dataExpired`, `label`, `include` |
| `Create` | `POST` | `/sandboxes` | `202`; async |
| `Get` | `GET` | `/sandboxes/{id}` | |
| `Update` | `PATCH` | `/sandboxes/{id}` | |
| `Delete` | `DELETE` | `/sandboxes/{id}` | Confirmation header is the exact id |
| `Stop` | `POST` | `/sandboxes/{id}/stop` | Optional `force` |
| `GetStop` | `GET` | `/sandboxes/{id}/stops/{stopId}` | |
| `Resume` | `POST` | `/sandboxes/{id}/resume` | `202`; async |
| `Fork` | `POST` | `/sandboxes/{id}/fork` | `202`; async |
| `Share` | `POST` | `/sandboxes/{id}/share` | |
| `Unshare` | `POST` | `/sandboxes/{id}/unshare` | |
| `Recover` | `POST` | `/sandboxes/{id}/recover` | Deprecated |
| `Usage` | `GET` | `/sandboxes/{id}/usage` | `since`, `until` |

## 6. Error behavior

- `400 invalid_*`: fix the request. `Create` surfaces them unchanged.
- `401 unauthorized`: invalid or missing key.
- `402 billing_required`: the account cannot create or operate sandboxes.
- `403 machine_class_plan_required`: the size needs a paid plan.
- `404 not_found`: the sandbox or stop does not exist.
- `409 sandbox_stopping`, `sandbox_resuming`, `resume_failed`, `fork_failed`,
  `snapshot_data_expired`, `idempotency_in_progress`, `idempotency_key_reused`: state conflicts.
  `idempotency_in_progress` is safe to retry with the same key; `idempotency_key_reused` is terminal.
- `429 rate_limited` / `limit_reached`: start limits. `core.RetryAfter` carries the wait.
- `503 no_ready_machine`: retry or fall back; a refused start does not count against the limits.

The transport retries only safe methods or calls carrying an Idempotency-Key. `Create` and `Fork`
always carry one, so a lost `202` cannot create a second billable sandbox.

## 7. Streaming and long-running behavior

No stream. `WaitUntilReady` polls `Get` every second by default and returns when the state is
`ready`, `idle` or `running`. It stops with an error on `error`, `cancelled` or `archived`, or when
the context or `WaitOptions.Timeout` ends.

## 8. Examples

```go
sb, err := client.Sandboxes.Create(ctx, sandboxes.CreateRequest{
    TTLSeconds: core.Some(3600),
})
if err != nil {
    return err
}
sb, err = client.Sandboxes.WaitUntilReady(ctx, sb.ID, sandboxes.WaitOptions{})
```

## 9. Testing notes

- `sandboxes/sandboxes_test.go` covers request shape, generated and overridden idempotency keys,
  explicit null TTL, pagination, stop/delete headers, usage windows and wait behavior.
- All tests use `httptest` or a fake `core.Doer`; none call the live API.
- Live coverage is a short create/get/stop/delete flow behind the `live` build tag.

## 10. Open questions / known gaps

- `Create` returns the first sandbox when `Count > 1`; use `CreateMany` for the batch.
- Batch create cannot be combined with `failFast` or `hostedPorts`; that is an API rule, not an SDK
  restriction.
- `Recover` is kept for API parity but marked deprecated.
