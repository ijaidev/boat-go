# Testing and TDD

This document defines how we test the SDK. The short version:

> If a unit test is feasible, write it first, watch it fail, then implement the code.

Tests are not a cleanup phase. They are the design tool that keeps the SDK small, predictable and
safe to change.

## 1. The TDD loop

For every new function, method or service behavior:

1. **Write the test** against the API you wish existed.
2. **Run it.** It must fail. For a brand-new type, a compile failure is the first red; add the
   minimal signature or stub and rerun until the failure is an assertion failure.
3. **Implement the minimum** that makes the test pass.
4. **Run the package tests**, then `go test ./...`.
5. **Refactor** with the tests green.
6. **Add edge cases**: errors, empty pages, cancellation, malformed input, nil optional fields.

If a unit test is genuinely not feasible, document why and cover the behavior with an integration,
compile or example test. Do not use "not feasible" to skip inconvenient tests.

## 2. Test pyramid

| Layer | What it tests | Speed | Runs in CI |
|---|---|---|---|
| Unit | Pure helpers, options, frame decoding, signature checks | fast | yes |
| Service + `httptest` | Request building, headers, JSON, envelopes, errors, pagination | fast | yes |
| Streaming | NDJSON frames, cancellation, close, reconnect semantics | fast | yes |
| Compile/example | Public API shape and documented examples | fast | yes |
| Live | Real Boat API behavior | slow, costs money | no, opt-in |

Most tests should be service + `httptest`. They validate the contract without a network dependency.

## 3. Layout and naming

- Tests live next to the code: `commands/stream.go` → `commands/stream_test.go`.
- Use external test packages (`package commands_test`) when testing only the public API; use the
  package under test when a private helper needs coverage.
- Name tests `TestService_Method_Behavior`:
  - `TestService_Create_SendsIdempotencyKey`
  - `TestService_Create_MapsConflict`
  - `TestStream_Next_ReturnsEOF`
- Use table-driven tests for variations.
- Use `t.Helper()`, `t.Cleanup()` and `t.Parallel()` where safe.
- No global mutable state.

## 4. HTTP tests

Use `httptest.Server` through `internal/testutil`. Assert on the request, not just the response.

```go
func TestService_Create_SendsIdempotencyKey(t *testing.T) {
    t.Parallel()

    srv := testutil.NewServer(t, testutil.Route{
        Method: http.MethodPost,
        Path:   "/sandboxes",
        Status: http.StatusAccepted,
        Body: `{
            "ok": true,
            "type": "sandbox.created",
            "sandbox": {"id": "bx_1", "state": "provisioning"}
        }`,
        Assert: func(t *testing.T, r *http.Request) {
            t.Helper()
            if got := r.Header.Get("Idempotency-Key"); got != "key-1" {
                t.Fatalf("Idempotency-Key = %q, want key-1", got)
            }
        },
    })

    svc := sandboxes.New(testutil.Transport(t, srv.URL, "boat_test"))
    _, err := svc.Create(context.Background(), sandboxes.CreateRequest{},
        core.WithIdempotencyKey("key-1"))
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
}
```

Rules:

- Assert method, path, query, headers and body where they matter.
- Return envelopes that match the real API, not simplified fakes.
- Test every documented error class the service can encounter.
- Never depend on a live sandbox id, IP or wall-clock timing.

## 5. Fixtures

- Put JSON fixtures in `testdata/` next to the package.
- Prefer small, synthetic fixtures derived from the OpenAPI examples. Scrub ids to `bx_example`
  and never use captured production payloads or signed URLs.
- Do not include secrets, live sandbox ids or signed URLs.
- A fixture may be inline for a one-off; use `testdata/` when it is reused.

## 6. Fakes and seams

- Use a fake `core.Doer` when the test is about service logic, not HTTP.
- Use `httptest` when the test is about the wire contract.
- Do not generate a mock for every method.
- Do not add an interface solely for mocking; define the narrow interface in the test if needed.
- A public `boattest` package may be added later. Keep it out of the core path until there is a
  real consumer.

## 7. Streaming tests

Streaming code is where most bugs hide. Every stream implementation needs:

- an `io.Pipe` or fixture-based test for `started`, `stdout`, `stderr`, `exit`, `error`
- an `io.EOF` test at the natural end
- a context cancellation test
- a `Close()` test that asserts the body is closed exactly once
- a malformed-frame test
- for `RunCommand`, a reconnect test that asserts no duplicated output

Example frame fixture:

```text
{"type":"started","processId":"proc_1"}
{"type":"stdout","data":"hello\n"}
{"type":"exit","exitCode":0}
```

## 8. Webhook tests

Use known HMAC vectors. Test:

- a valid signature over the exact raw body
- a body that was re-serialized after signing (must fail)
- an invalid signature
- a stale timestamp
- a missing header
- a duplicate delivery id
- a valid signature with the wrong event type

The implementation must use `hmac.Equal` or `subtle.ConstantTimeCompare`.

## 9. Live tests

Live tests are opt-in and never part of normal CI:

```go
//go:build live
```

Requirements:

- `BOAT_API_KEY` must be set.
- Create the smallest sandbox (`small`) with a short TTL.
- Always clean up with `t.Cleanup` (stop, then delete if the test created it).
- Print the sandbox id on failure for debugging, never the API key.
- Keep the number of live tests tiny; they cost money and are flaky by nature.

Run them explicitly:

```bash
BOAT_API_KEY=boat_... go test -tags=live ./...
```

## 10. Race and concurrency

Run `go test -race ./...` when touching:

- streaming
- pagination
- retries
- shared caches or clients
- anything with goroutines

The `Client` and all services must be safe for concurrent use. Do not add mutable package state.

## 11. Coverage

Coverage is a guide, not a target. The minimum bar:

- every exported method has a success test
- every exported method has at least one error-path test
- every retry branch has a test
- every streaming frame type has a test
- every webhook verification failure mode has a test

Do not chase 100% by testing generated code or trivial getters.

## 12. CI commands

```bash
gofmt -l .          # must be empty
go vet ./...
go test ./...
go test -race ./...
go generate ./...   # then check for a dirty tree
```

Optional once configured:

```bash
staticcheck ./...
golangci-lint run
```

## 13. Anti-patterns

- Testing implementation details instead of behavior.
- Asserting on `time.Sleep` durations.
- Using a live API in a unit test.
- One giant `client_test.go`.
- Mocking every service method instead of faking the transport.
- Ignoring `Close()` and body leaks.
- Re-serializing a webhook body before signature verification.
- Treating a compile failure as a substitute for a red assertion (fix the stub and rerun).
