# Commands

Status: done

Owner: SDK maintainers

Package: `commands`

## 1. Purpose and scope

Wraps command execution in a Boat sandbox:

- synchronous execution with captured stdout, stderr and exit code
- streamed execution as newline-delimited JSON frames
- detached execution with status polling, stop, log reads and attach
- command and process history, stored output and live process lists
- `RunCommand`, a small follow helper that starts a detached command, attaches, and reconnects on
  the protocol's `reconnect` frames

Every method takes a `context.Context`, returns errors rather than panicking, and is safe for
concurrent use. A service instance is stateless beyond its transport.

## 2. Non-goals

- Creating, resuming or deleting sandboxes. Those live in `sandboxes`.
- Uploading files or opening interactive terminals. Those belong in `files` and other services.
- Create → push → exec → usage → delete orchestration. That belongs in `jobs/`.
- Automatic retry of commands, polling loops for detached status, or scheduler behavior. Callers
  decide when to poll, stop or retry.
- Deduplicating command history records. The API already records each command.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `New` | `New(core.Doer) *Service` | Wires a transport |
| `Run` | `Run(ctx, sandboxID, RunRequest, ...core.RequestOption) (*model.CommandResponse, error)` | Synchronous; never retried |
| `Stream` | `Stream(ctx, sandboxID, RunRequest, ...core.RequestOption) (core.Stream[model.CommandStreamFrame], error)` | `stream: true`; NDJSON |
| `Start` | `Start(ctx, sandboxID, RunRequest, ...core.RequestOption) (*model.CommandStartedResponse, error)` | `detached: true` |
| `Attach` | `Attach(ctx, sandboxID, processID, AttachRequest, ...core.RequestOption) (core.Stream[model.CommandAttachFrame], error)` | Resume output from byte offsets |
| `Status` | `Status(ctx, sandboxID, processID, StatusRequest, ...core.RequestOption) (*model.CommandStatusResponse, error)` | Tail or offset reads |
| `Stop` | `Stop(ctx, sandboxID, processID, ...core.RequestOption) (*model.CommandStopResponse, error)` | SIGTERM, then SIGKILL after 5 s |
| `History` | `History(ctx, sandboxID, HistoryRequest, ...core.RequestOption) (core.Page[model.CommandRecord], error)` | Cursor pagination |
| `GetHistory` | `GetHistory(ctx, sandboxID, commandID, ...core.RequestOption) (*model.CommandRecord, error)` | One record |
| `Logs` | `Logs(ctx, sandboxID, commandID, LogsRequest, ...core.RequestOption) (*model.CommandLogsResponse, error)` | One stored-output chunk |
| `Processes` | `Processes(ctx, sandboxID, ...core.RequestOption) (*model.ProcessListResponse, error)` | Live or last sample |
| `RunCommand` | `RunCommand(ctx, sandboxID, RunRequest, ...core.RequestOption) (core.Stream[Event], error)` | Detached start plus resumable follow |

Request options apply per call. `core.WithOrg` overrides the client organization. Command requests
set `core.Request.NoRetry`, so the transport never retries them even if a caller supplies
`core.WithIdempotencyKey`. The service does not generate a key.

Timeouts: a synchronous `Run` or `Stream` uses 30 seconds by default and accepts 1-600 seconds.
`Start` and `RunCommand` have no SDK timeout; an explicit `TimeoutSeconds` is forwarded.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `RunRequest` | `model.CommandRequest` | Hand-written; no `detached`/`stream` fields |
| `AttachRequest` | — | `StdoutOffset`, `StderrOffset` byte offsets; zero means the start |
| `StatusRequest` | — | Optional `TailBytes`, `StdoutOffset`, `StderrOffset` pointers |
| `HistoryRequest` | — | `Mode`, `Query` (`q`), `Limit`, `Cursor` |
| `LogsRequest` | — | `Stream`, `Offset`, `Limit`, `Encoding` |
| `Event` | `model.CommandAttachFrame` | Flattened event for `RunCommand` |
| `EventType` | — | `started`, `stdout`, `stderr`, `exit`, `error` |

Frames and responses otherwise reuse `model.*` directly. `Processes` returns
`*model.ProcessListResponse`; `History` returns `core.Page[model.CommandRecord]` with
`HasMore = nextCursor != null`.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Run` | `POST` | `/sandboxes/{sandboxId}/commands` | JSON result; 30 s default, 600 s cap |
| `Stream` | `POST` | `/sandboxes/{sandboxId}/commands` | `stream: true`; `application/x-ndjson` |
| `Start` | `POST` | `/sandboxes/{sandboxId}/commands` | `detached: true`; immediate process id |
| `Status` | `GET` | `/sandboxes/{sandboxId}/commands/{processId}` | `tailBytes`, `stdoutOffset`, `stderrOffset` |
| `Stop` | `DELETE` | `/sandboxes/{sandboxId}/commands/{processId}` | Detached commands only |
| `Attach` | `GET` | `/sandboxes/{sandboxId}/commands/{processId}/stream` | `stdoutOffset`, `stderrOffset` |
| `History` | `GET` | `/sandboxes/{sandboxId}/command-history` | `mode`, `q`, `limit`, `cursor` |
| `GetHistory` | `GET` | `/sandboxes/{sandboxId}/command-history/{commandId}` | |
| `Logs` | `GET` | `/sandboxes/{sandboxId}/command-history/{commandId}/logs` | `stream`, `offset`, `limit`, `encoding` |
| `Processes` | `GET` | `/sandboxes/{sandboxId}/processes` | Live or stored sample |

`RunCommand` calls `Start` and then `Attach`; it reissues `Attach` when a `reconnect` frame ends the
stream.

## 6. Error behavior

- `400 invalid_timeout`: `timeoutSeconds` is not an integer in 1-600.
- `401 unauthorized`: the key is missing, invalid or lacks the `exec` scope.
- `404 not_found`: the sandbox, process or history record does not exist.
- `409 boat_starting` / `sandbox_starting`: the sandbox is still restoring. It may be safe to retry
  the command, but only the caller knows whether the first attempt ran. The SDK does not retry.
- `409 sandbox_not_ready`: the sandbox is stopped or in error. Resume or recover it first. The API
  marks this `retryable: false`.
- `409 agent_outdated`: the sandbox agent is too old for offset reads.
- `429 rate_limited`: honor `core.RetryAfter`.
- `502 sandbox_direct_failed` with `sandbox_restoring`: the command provably did not run. The error
  carries `core.Retryable(err) == true`; retrying is the caller's decision.
- `502` without that hint: the command may already be running. Never retry automatically.

Command `POST` and `DELETE` requests set `core.Request.NoRetry`, so the transport never retries them,
with or without an Idempotency-Key. GET reads (status, history, logs, processes) are idempotent and
may be retried by the transport on 429, 500, 502 and 503.

Errors are `*core.APIError`; use `core.IsCode`, `core.Retryable` and `core.RetryAfter`, never string
matching.

## 7. Streaming and long-running behavior

`Stream` returns `model.CommandStreamFrame` values:

| Frame | Meaning |
|---|---|
| `started` | First frame; `commandId` names the history record |
| `stdout` / `stderr` | One output chunk in `data` |
| `exit` | Last frame on success; `exitCode`, `success`, `timedOut` |
| `error` | Last frame on failure; `error`, `message`, optional `retryable` |

`Attach` returns `model.CommandAttachFrame` values: `attached`, `stdout`, `stderr`, `heartbeat`,
`reconnect`, `exit`, `error`. A `reconnect` frame is the last line after ten minutes; reopen the
stream with its `stdoutOffset` and `stderrOffset` to continue with no gap or duplicate output.

Both streams:

- return `io.EOF` at the natural end
- close the response body exactly once, including on EOF, error, cancellation and repeated `Close`
- return `context.Canceled` or `context.DeadlineExceeded` promptly from `Next`
- treat malformed JSON as terminal and return the same error on later `Next` calls
- are not safe for concurrent `Next` calls

`RunCommand` hides `heartbeat` frames, emits one `EventStarted`, follows stdout and stderr as
separate events, reconnects on `reconnect` using the server offsets, then emits `EventExit` or
`EventError`. It never duplicates output across a reconnect. `Close` releases the attach body but
does not stop the detached command; call `Stop` for that.

## 8. Examples

```go
started, err := client.Commands.Start(ctx, sandboxID, commands.RunRequest{
    Command: "npm test",
})
if err != nil {
    return err
}

stream, err := client.Commands.Attach(ctx, sandboxID, started.ProcessID, commands.AttachRequest{})
if err != nil {
    return err
}
defer stream.Close()

for {
    frame, err := stream.Next(ctx)
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    switch frame.Type {
    case model.CommandAttachFrameTypeStdout:
        fmt.Print(*frame.Data)
    case model.CommandAttachFrameTypeReconnect:
        // Emit the event to the caller and reopen with the frame offsets.
    }
}
```

For most callers, `RunCommand` replaces the manual start/attach/reconnect loop:

```go
stream, err := client.Commands.RunCommand(ctx, sandboxID, commands.RunRequest{
    Command: "npm test",
})
if err != nil {
    return err
}
defer stream.Close()

for {
    event, err := stream.Next(ctx)
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    if event.Type == commands.EventStdout || event.Type == commands.EventStderr {
        fmt.Print(event.Data)
    }
}
```

## 9. Testing notes

- `commands/run_test.go` covers synchronous request shape, no Idempotency-Key, no retry on 502,
  `core.Retryable` mapping, detached start, every stream frame, the error frame and documented HTTP
  errors.
- `commands/stream_test.go` covers close-once, EOF, malformed frames, prompt cancellation, attach
  offsets, the attach error frame and argument validation.
- `commands/read_test.go` covers status tail/offset queries, stop, history pagination, history
  records, stored logs and process lists.
- `commands/follow_test.go` covers the `RunCommand` start/attach/reconnect path, asserting that the
  second attach uses the reconnect offsets and that output is not duplicated, plus error events and
  start/attach failures.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`; none call the live API.
- There are no live tests yet.

## 10. Open questions / known gaps

- `HistoryRequest.Mode` is a string, not a typed enum, because the API accepts a comma-separated
  list. Callers must use the documented values.
- `RunCommand` returns the attach error and leaves the detached command running when the first
  attach fails. The caller can call `Stop` with the known process id only if it started the command
  through `Start` instead.
- `RunCommand` reconnects only on explicit `reconnect` frames. A network drop or unexpected EOF is
  returned as an error; the caller decides whether to attach again with the last offsets.
- `Logs` reads one chunk per call. Callers loop with `Offset = NextOffset` until `Done`; the SDK
  does not hide that loop.
- `Stop` applies only to detached commands, matching the API.
