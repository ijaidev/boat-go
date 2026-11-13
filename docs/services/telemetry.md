# Telemetry

Status: done

Owner: SDK maintainers

Package: `telemetry`

## 1. Purpose and scope

Wraps the account's OpenTelemetry export configuration:

- reading the configured OTLP/HTTP destination, its saved header names and its last export result;
- setting or replacing that destination and returning the result of the first test export;
- removing the destination and its queued deliveries.

Boat sends account and sandbox metrics to `<endpoint>/v1/metrics` every 60 seconds and sandbox
lifecycle events to `<endpoint>/v1/logs`. The API does **not** send traces, and it never sends agent
token data. The SDK itself emits no telemetry; this service only manages where Boat exports.

Export headers may carry vendor credentials such as an API key. The API returns header names only,
and the service never logs values or puts them in error text.

## 2. Non-goals

- Emitting spans, metrics, logs or any telemetry from the SDK. Boat owns collection and delivery.
- Running or configuring an OTLP collector. The caller supplies an HTTPS endpoint it controls.
- Sending traces. Boat's export is metrics and lifecycle logs only.
- Sending agent token data. It is not part of the export contract.
- Storing header values in the SDK. Only the caller's request and Boat hold them.
- Retrying `Delete` or retrying `Set` on the caller's behalf. See §6.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `New` | `New(core.Doer) *Service` | Wires a transport |
| `Get` | `Get(ctx, ...core.RequestOption) (*model.TelemetryConfig, error)` | `nil, nil` when no export is set |
| `Set` | `Set(ctx, model.TelemetrySetRequest, ...core.RequestOption) (*model.TelemetrySetResponse, error)` | Replaces the export; returns the test result |
| `Delete` | `Delete(ctx, ...core.RequestOption) error` | Removes the export and queued deliveries |

Every method takes `context.Context` first and accepts per-call options such as `core.WithOrg`.
Services are safe for concurrent use and keep no mutable package state besides the transport.

`Set` validates the spec's shape before sending: the endpoint must be non-empty, at most 20 headers
may be supplied, header names must be non-empty, and each value may be at most 8192 characters.
Header values are never included in a validation error.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `model.TelemetryConfig` | generated | Endpoint, header **names**, last export time, last error, creation time |
| `model.TelemetrySetRequest` | generated | `Endpoint` is required; `Headers` is an optional map of values |
| `model.TelemetrySetResponse` | generated | Saved configuration plus `Test` |
| `model.TelemetryTestResult` | generated | `Ok`, optional HTTP `Status`, optional collector `Message` |

`model.TelemetrySetRequest.Headers` is a `*map[string]string`: `nil` omits the field, while an empty
map sends `{}`. `model.TelemetryConfig.Headers` is `[]string` containing names only, so a caller can
never read a saved credential back through the SDK.

`Set` returns the full `model.TelemetrySetResponse` so the test export result is preserved. Inspect
`Test.Ok`, `Test.Status` and `Test.Message`; a failed collector test is not an HTTP error.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Get` | `GET` | `/telemetry` | `telemetry` may be `null` |
| `Set` | `PUT` | `/telemetry` | Replaces the single destination; `200` |
| `Delete` | `DELETE` | `/telemetry` | `404` when no export exists |

The response envelope is decoded by the shared transport. `Delete` ignores the success payload; the
spec's `telemetry` field is `null` after removal.

## 6. Error behavior

All failures are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID` and, when supplied,
`RetryAfter`. Use `core.IsCode` and `errors.As`; do not match messages.

- `400 invalid_endpoint` (or another `invalid_*`): the endpoint is not an HTTPS URL on port 443 or
  4318, or the body is otherwise invalid. Fix the request.
- `401 unauthorized`: the key is missing or invalid.
- `402 credit_required`: the account cannot use telemetry export on its current plan.
- `403 forbidden`: the key lacks `account.admin` or cannot manage this account.
- `404 not_found` on `Delete`: no export is configured. A retry after a lost success also lands
  here.
- `409` conflicts: the account's telemetry configuration changed concurrently. Re-read with `Get`
  before deciding whether to retry.

Retry behavior:

- `Get` is a safe GET and is retried on `429`, `500`, `502` and `503` with jitter and `Retry-After`.
- `Set` is idempotent but PUT is not a safe method, so it is **not** retried by default. Pass
  `core.WithIdempotencyKey` to explicitly opt in to those retries; the same key is reused for every
  attempt.
- `Delete` sets `core.Request.NoRetry`, so it is never retried, even with an idempotency key. A
  retry could turn a successful deletion into a misleading `404`.

## 7. Streaming and long-running behavior

No stream and no long-running call. `Set` returns after Boat performs one synchronous test export,
so its latency is bounded by the collector's response time. After that, Boat exports metrics every
60 seconds and lifecycle events as they occur, independently of the SDK.

Cancelling `ctx` aborts `Get`, `Set` or `Delete` and is returned as `context.Canceled` or
`context.DeadlineExceeded`; the SDK does not retry after cancellation.

## 8. Examples

```go
cfg, err := client.Telemetry.Get(ctx)
if err != nil {
    return err
}
if cfg == nil {
    // No export is configured.
}

result, err := client.Telemetry.Set(ctx, model.TelemetrySetRequest{
    Endpoint: "https://otlp.example.com",
    Headers: &map[string]string{
        "dd-api-key": os.Getenv("DATADOG_API_KEY"),
    },
})
if err != nil {
    return err
}
if !result.Test.Ok {
    // The configuration is saved, but the collector rejected the test export.
    log.Printf("telemetry test failed: %v", result.Test.Message)
}

if err := client.Telemetry.Delete(ctx); err != nil {
    return err
}
```

Inline fragments are illustrative. Do not log the header value or put it in an error.

## 9. Testing notes

- `telemetry/telemetry_test.go` covers `Get` with a configuration and with `null`, `Set` request
  shape and the preserved test result, omitted nil headers, local validation, and `Delete`.
- Header redaction is covered by a validation test with a credential-sized value: the error names
  the header but never contains its value.
- Error tables cover `400`, `401`, `402`, `403`, `404` and `409`.
- Retry tests assert that `Set` makes one attempt by default, two when the caller supplies an
  idempotency key, and that `Delete` makes one attempt even with a key.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`. None call the live API.
- No live tests: changing telemetry would send real exports to a real collector.

## 10. Open questions / known gaps

- Local validation mirrors only the documented shape limits (`20` headers, `8192` characters per
  value, non-empty endpoint). HTTPS, port and public-address checks are left to the API.
- `Set` returns the full envelope rather than a narrow `(config, test, error)` tuple, matching other
  services that expose rich response bodies.
- Header values cannot be read back through the SDK by design; to rotate one, call `Set` again with
  the full header map.
- No helper waits for a later metrics export or inspects historical exports. The API exposes only
  `LastExportAt` and `LastError`.
