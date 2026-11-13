# Webhooks

Status: done

Owner: SDK maintainers

Package: `webhooks`

## 1. Purpose and scope

Wraps the Boat webhook management API and verifies inbound sandbox lifecycle deliveries:

- list, create, read, update and delete account-wide webhook endpoints,
- rotate an endpoint's signing secret,
- verify the HMAC-SHA256 signature of a delivery and return its parsed event.

Boat sends the same lifecycle events for every subscribed endpoint and retries failed deliveries
with exponential backoff. An endpoint must be public HTTPS on port 443 and resolve only to public
addresses. Redirects are not followed. An account can register at most 10 unique endpoint URLs.

Official reference: <https://docs.boat.dev/api/v1>.

## 2. Non-goals

- Delivering events, queueing them, scheduling retries or observing delivery attempts. Boat owns the
  delivery pipeline; this package only manages registrations and verifies what arrives.
- Storing signing secrets. The caller must persist the one-time secret returned by `Create` and
  `Rotate` in a secret manager.
- Deduplication, replay storage or ordering. Deliveries are at-least-once and may arrive out of
  order; the library keeps no state.
- Emitting sandbox lifecycle events. `sandboxes` owns the lifecycle operations.
- Cross-domain flows such as create → subscribe → process. Those belong in `jobs/`.
- Fetching event history. The v1 API has no delivery-history endpoint.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `List` | `List(ctx, ...core.RequestOption) ([]model.Webhook, error)` | No pagination; secret never included |
| `Create` | `Create(ctx, CreateRequest, ...core.RequestOption) (*SecretResult, error)` | `201`; returns the one-time secret |
| `Get` | `Get(ctx, webhookID string, ...core.RequestOption) (*model.Webhook, error)` | Metadata only |
| `Update` | `Update(ctx, webhookID string, UpdateRequest, ...core.RequestOption) (*model.Webhook, error)` | Keeps the existing secret |
| `Delete` | `Delete(ctx, webhookID string, ...core.RequestOption) (string, error)` | Returns the deleted id |
| `Rotate` | `Rotate(ctx, webhookID string, ...core.RequestOption) (*SecretResult, error)` | Returns the new one-time secret |
| `Verify` | `Verify(rawBody []byte, headers http.Header, secret string, opts VerifyOptions) (Event, error)` | Stateless signature check |

Every I/O method takes `context.Context` first and accepts per-call `core.RequestOption`s such as
`core.WithOrg`. `Service` is safe for concurrent use and has no mutable package state. The `webhook`
id is required by `Get`, `Update`, `Delete` and `Rotate`; an empty id is rejected locally.

`Create` and `Rotate` send no `Idempotency-Key`, so the transport does not retry them. A caller may
pass `core.WithIdempotencyKey`, but the v1 webhook endpoints do not document idempotency; prefer
`List` to reconcile after an ambiguous failure.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `CreateRequest` | — | Hand-written for the nullable `name` |
| `UpdateRequest` | — | Hand-written optional and nullable fields |
| `SecretResult` | `model.Webhook` | `Webhook` plus the one-time `Secret` |
| `Event` | embeds `model.WebhookEvent` | Parsed body plus delivery headers |
| `VerifyOptions` | — | `Tolerance`, `Now` |
| `DefaultTolerance` | — | `5 * time.Minute` |
| `Header*` constants | — | The five `X-Ascii-*` delivery headers |

`CreateRequest.URL` and `CreateRequest.Events` are required. `Name` has three JSON states: nil omits
the field, `core.Some("name")` sets it, and `core.Null[string]()` sends an explicit null. The API
also accepts null as "no name".

`UpdateRequest` sends only the fields that are set. `Name` uses the same three states. `URL` is a
pointer and `Events` is omitted when empty; the API requires at least one changed field, so an empty
`UpdateRequest` produces `{}` and a `400`.

`SecretResult.Secret` is the only copy the API returns. `List`, `Get` and `Update` never include it.
`SecretResult` implements `fmt.Stringer` and `fmt.GoStringer`; both redact the secret so `%v` and
`%#v` do not leak it. Direct access to the field is still possible, so do not log it yourself.

`Event` embeds the parsed `model.WebhookEvent` and adds the delivery metadata:

| Field | Source | Meaning |
|---|---|---|
| `DeliveryID` | `X-Ascii-Delivery` | Stable idempotency key across retries |
| `Type` (embedded) | `X-Ascii-Event` + body `type` | Header and body must agree |
| `ID` (embedded) | body `id` | Event id; may repeat across endpoints |
| `Timestamp` | `X-Ascii-Timestamp` | Unix seconds, normalized to UTC |
| `Attempt` | `X-Ascii-Attempt` | 1-based attempt number, 1–8 |
| `RawBody` | request body | Exact signed bytes; aliases the input slice |

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `List` | `GET` | `/webhooks` | Account-wide |
| `Create` | `POST` | `/webhooks` | `201`; secret in response |
| `Get` | `GET` | `/webhooks/{webhookId}` | |
| `Update` | `PATCH` | `/webhooks/{webhookId}` | `name`, `url`, `events` |
| `Delete` | `DELETE` | `/webhooks/{webhookId}` | Queued deliveries are discarded |
| `Rotate` | `POST` | `/webhooks/{webhookId}/rotate` | New secret in response |

Deliveries are `POST` requests to the registered URL with these headers:

| Header | Value |
|---|---|
| `X-Ascii-Event` | Event type, such as `sandbox.ready` |
| `X-Ascii-Delivery` | `evt_` plus 32 lowercase hex characters |
| `X-Ascii-Timestamp` | Unix seconds as ASCII digits |
| `X-Ascii-Signature` | `v1=` plus 64 lowercase hex characters |
| `X-Ascii-Attempt` | Integer 1–8 |

The signature is `HMAC-SHA256(secret, deliveryID + "." + timestamp + "." + rawBody)`. The event and
attempt headers are not part of the signed message. `Verify` checks the event header against the
signed body type, but `Event.Attempt` is delivery metadata rather than an authenticated value.

## 6. Error behavior

Management errors are `*core.APIError`:

- `400`: invalid URL, empty or unknown event list, or an empty update. Fix the request.
- `401 unauthorized`: invalid or missing API key.
- `404 not_found`: the webhook id does not exist.
- `409 conflict`: the URL is already registered or the account reached the 10-endpoint limit.
- `429`, `500`, `502`, `503`: retried by the transport for `GET` only. `POST`, `PATCH` and `DELETE`
  are not retried automatically because the SDK sends no idempotency key.
- `Delete` can still race an in-flight delivery; a handler must tolerate an event for an endpoint
  that was just removed.

`Verify` returns sentinel errors for `errors.Is`:

| Error | Cause |
|---|---|
| `ErrMissingSecret` | The secret argument is empty |
| `ErrMissingHeader` | A required `X-Ascii-*` header is absent |
| `ErrInvalidHeader` | Bad timestamp, delivery id or attempt shape |
| `ErrInvalidSignature` | Wrong version prefix, malformed hex, wrong length or digest mismatch |
| `ErrStaleTimestamp` | Timestamp older or newer than the tolerance |
| `ErrInvalidEvent` | Unknown event type, or header and body disagree |
| `ErrInvalidPayload` | Verified body is not a valid `WebhookEvent` JSON object |

Verification errors never include the secret, the expected signature or the raw body. The signature
comparison uses `hmac.Equal`, and JSON is parsed only after the signature and timestamp are valid.

## 7. Streaming and long-running behavior

There is no stream and no long-poll. Webhook management is request/response.

Delivery is at-least-once and separate events may arrive out of order. Return any `2xx` within five
seconds to acknowledge an event; Boat retries other responses with exponential backoff. A duplicate
delivery is expected, not an error.

Deduplication is caller-owned. The library keeps no state and does not cache delivery ids. Store
`Event.DeliveryID` in durable storage and skip a delivery id that is already present. Apply the
effect and record the id in the same transaction where possible; recording before processing can
lose an event after a crash, and processing before recording can repeat it.

`Verify` accepts timestamps within `VerifyOptions.Tolerance`, five minutes by default, in either
direction. Set a shorter tolerance only when both clocks are tightly synchronized. For deterministic
tests, override `VerifyOptions.Now`.

`Verify` needs the exact request body bytes. Read the body once, bound it with
`http.MaxBytesReader`, and pass those bytes unchanged; decoding into a struct and re-serializing
breaks the signature. After a rotation, keep accepting the old secret for attempts already in
flight, then remove it.

## 8. Examples

Create an endpoint and keep the secret out of logs. `svc` is a `*webhooks.Service`:

```go
res, err := svc.Create(ctx, webhooks.CreateRequest{
    URL:    "https://example.com/hooks/boat",
    Events: []model.WebhookEventType{model.SandboxReady, model.SandboxError},
})
if err != nil {
    return err
}
if err := secrets.Put(ctx, "boat/webhooks/"+res.Webhook.ID, res.Secret); err != nil {
    return err
}
```

Verify a delivery:

```go
func handleDelivery(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    event, err := webhooks.Verify(body, r.Header, signingSecret, webhooks.VerifyOptions{})
    if err != nil {
        http.Error(w, "invalid delivery", http.StatusUnauthorized)
        return
    }
    if alreadyProcessed(event.DeliveryID) {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if event.Type == model.SandboxReady {
        handleReady(event.Data.Sandbox.ID)
    }
    w.WriteHeader(http.StatusNoContent)
}
```

The fragment assumes `signingSecret` was loaded from a secret manager and `alreadyProcessed` uses
caller-owned durable state.

## 9. Testing notes

- `webhooks/verify_test.go` uses a fixed HMAC-SHA256 vector computed independently from the
  implementation, and asserts every verification failure mode: exact vs re-serialized body, wrong
  secret, malformed hex, missing version prefix, short digest, missing headers, invalid timestamp,
  stale and future timestamps, custom tolerance, unknown and mismatched event types, and invalid
  JSON.
- The duplicate-delivery test calls `Verify` twice with the same delivery id and asserts both
  succeed, documenting that deduplication belongs to the caller.
- `webhooks/read_test.go` and `webhooks/manage_test.go` cover request method, path, headers and JSON
  body for all six endpoints, the three `name` states, secret redaction, and `400`, `401`, `404`,
  `409` and `503` error paths.
- All tests use `httptest` through `internal/testutil`; none call the live API. Secrets are fake
  `whsec_test_...` values.
- No live-test coverage: a live webhook test needs a public receiver and would be inherently flaky.

## 10. Open questions / known gaps

- `List` has no pagination because the v1 endpoint has none; the account limit is 10 endpoints.
- A lost `Create` or `Rotate` response loses the only copy of that secret. Rotate again to recover;
  the previous secret is invalidated.
- Rotation overlap is caller-managed. The SDK does not store old secrets or expose a multi-secret
  verifier.
- `Create` and `Rotate` have no documented idempotency key. An ambiguous failure is reconciled by
  listing endpoints and rotating if necessary.
- Replay protection is the five-minute timestamp window plus caller-owned delivery-id deduplication.
  The library keeps no nonce cache.
- `Verify` buffers the whole raw body. A receiver should cap the request size before calling it.
- Event data stays in the generated nested struct; a richer typed payload per event type may be
  added later without breaking `Event`.
