# Account

Status: done

Owner: SDK maintainers

Package: `account`

## 1. Purpose and scope

Wraps the account-level Boat endpoints that are not tied to a sandbox:

- the authenticated user and account status (`GET /me`);
- the wallets the account can bill and the account-wide active wallet (`GET /orgs`,
  `PATCH /orgs/active`);
- creation, start-window and concurrency limits for a wallet (`GET /limits`);
- API key inventory, scoped key creation, rotation, revocation and usage (`/api-keys`);
- zero-data-retention and account-wide snapshot/monitoring defaults
  (`/account/data-retention`);
- asynchronous deletion operation polling (`/deletion-operations/{operationId}`);
- the legacy account sandbox secret setup (`/secrets`).

This service is for inspection and account configuration. It does not create sandboxes; the
limits it reads are the same ones a create would check.

## 2. Non-goals

- Renaming an organization or changing its new-sandbox visibility (`PATCH /orgs/{org}`). Not
  exposed here; it is an owner-only dashboard operation in the current scope.
- Billing, wallet top-ups, seats and plan changes. Those are dashboard features, not part of v1.
- Sandbox starts and batch scheduling. Use `sandboxes.Create` or `sandboxes.CreateMany`.
- Webhook endpoints. Use `webhooks`.
- Environments and their secrets. The legacy `/secrets` endpoint here is account-wide setup, not
  the environments service.
- Polling helpers. `DeletionOperation` returns the current state; callers poll it.
- Caching or storing API key secrets, environment secrets or data-retention choices. The service is
  stateless.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `Me` | `Me(ctx, ...core.RequestOption) (*model.MeResponse, error)` | Current user and account status |
| `Orgs` | `Orgs(ctx, ...core.RequestOption) (*model.OrgListResponse, error)` | Personal account first, then organizations |
| `SetActiveOrg` | `SetActiveOrg(ctx, org string, ...core.RequestOption) (*model.ActiveOrgResponse, error)` | Empty string sends `null` for personal |
| `Limits` | `Limits(ctx, LimitsRequest, ...core.RequestOption) (*model.LimitsResponse, error)` | Read-only; wallet selection |
| `APIKeys` | `APIKeys(ctx, ...core.RequestOption) (*model.APIKeysResponse, error)` | Metadata and scope catalog |
| `CreateScopedAPIKey` | `CreateScopedAPIKey(ctx, CreateScopedAPIKeyRequest, ...core.RequestOption) (*model.APIKeySecretResponse, error)` | Secret shown once |
| `RotateAPIKey` | `RotateAPIKey(ctx, apiKeyID string, ...core.RequestOption) (*model.APIKeySecretResponse, error)` | Replacement secret shown once |
| `RevokeAPIKey` | `RevokeAPIKey(ctx, apiKeyID string, ...core.RequestOption) (*model.APIKeyRevokeResponse, error)` | Immediate |
| `APIKeyUsage` | `APIKeyUsage(ctx, apiKeyID string, ...core.RequestOption) (*model.APIKeyUsageResponse, error)` | 30-day window and created resources |
| `DataRetention` | `DataRetention(ctx, ...core.RequestOption) (*model.DataRetentionPolicyResponse, error)` | Account-wide; API keys get 403 |
| `UpdateDataRetention` | `UpdateDataRetention(ctx, UpdateDataRetentionRequest, ...core.RequestOption) (*model.DataRetentionPolicyResponse, error)` | Browser session required |
| `DeletionOperation` | `DeletionOperation(ctx, operationID string, ...core.RequestOption) (*model.DeletionOperation, error)` | Async; poll until settled |
| `Secrets` | `Secrets(ctx, ...core.RequestOption) (*model.SecretsResponse, error)` | Plaintext account secrets |
| `UpdateSecrets` | `UpdateSecrets(ctx, UpdateSecretsRequest, ...core.RequestOption) (*model.SecretsResponse, error)` | Full replacement |

Every I/O method takes `context.Context` first and accepts per-call options. `core.WithOrg` sets the
`X-Boat-Org` header for one request; `SetActiveOrg` changes the account-wide default. Services are
safe for concurrent use and hold no mutable state.

`Limits` and `SetActiveOrg` are the only methods with wallet semantics. `LimitsRequest.Org` is the
`org` query parameter; `LimitsRequest.TeamID` is the legacy `teamId` alias. The API gives `teamId`
precedence over `org` and `X-Boat-Org`. `SetActiveOrg` takes an organization id or name, `personal`,
or an empty string for the personal account (`"org": null`).

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `LimitsRequest` | — | Hand-written; `Org`, `TeamID` map to query parameters |
| `CreateScopedAPIKeyRequest` | — | Hand-written body; `Name`, `TTL`, `Preset`, `Actions`, `SandboxIDs`, `EnvironmentIDs` |
| `UpdateDataRetentionRequest` | — | Hand-written body with `*bool` fields for explicit false |
| `UpdateSecretsRequest` | — | Hand-written body; `EnvContents`, `SecretFiles` |
| `DataRetentionConfirmation` | — | Constant phrase `delete archived sandbox data` |
| `model.MeResponse` | generated | User fields are pointers when the API may omit them |
| `model.OrgListResponse` | generated | Includes `ActiveWalletChosen` |
| `model.LimitsResponse` | generated | Start windows, wallet, machine types and capacity |
| `model.APIKeysResponse` | generated | Includes the catalog |
| `model.APIKeySecretResponse` | generated | `Secret` is the one-time raw key |
| `model.APIKeyUsageResponse` | generated | Includes `CreatedResources` |
| `model.DataRetentionPolicyResponse` | generated | Flags and update counters |
| `model.DeletionOperation` | generated | Async purge state |
| `model.SecretsResponse` | generated | Plaintext environment and file secrets |

`CreateScopedAPIKeyRequest` always sends only the fields that are set. The method rejects a request
with no preset, actions, sandbox ids and environment ids before sending it, because the API treats
an unknown field as a 400 and an empty scope must not silently become full access.

`UpdateDataRetentionRequest` uses pointers so that `false` is distinguishable from "leave alone".
`UpdateSecretsRequest` has no pointers: both JSON fields are always sent, and a nil `SecretFiles`
slice is serialized as `[]`, because `POST /secrets` is a full replacement.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Me` | `GET` | `/me` | |
| `Orgs` | `GET` | `/orgs` | |
| `SetActiveOrg` | `PATCH` | `/orgs/active` | Body `{"org": string \| null}` |
| `Limits` | `GET` | `/limits` | `org`, `teamId`; `X-Boat-Org` via option |
| `APIKeys` | `GET` | `/api-keys` | |
| `CreateScopedAPIKey` | `POST` | `/api-keys/scoped` | `201`; secret shown once |
| `RotateAPIKey` | `POST` | `/api-keys/{apiKeyId}/rotate` | Secret shown once |
| `RevokeAPIKey` | `DELETE` | `/api-keys/{apiKeyId}` | |
| `APIKeyUsage` | `GET` | `/api-keys/{apiKeyId}/usage` | |
| `DataRetention` | `GET` | `/account/data-retention` | Never cached |
| `UpdateDataRetention` | `PATCH` | `/account/data-retention` | Browser session only |
| `DeletionOperation` | `GET` | `/deletion-operations/{operationId}` | Async |
| `Secrets` | `GET` | `/secrets` | Legacy setup |
| `UpdateSecrets` | `POST` | `/secrets` | Full replacement |

Paths escape ids with `url.PathEscape`. Empty API key and deletion operation ids are rejected
locally, before any request.

## 6. Error behavior

All API failures are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID` and, when the
API supplies one, `RetryAfter`. Use `core.IsCode(err, code)` or `errors.As`; never match messages.

- `400`: a malformed body or query parameter. `Limits` names an unknown query parameter, and
  `CreateScopedAPIKey` rejects an unknown field so a misspelled scope cannot create a full key.
- `401 unauthorized`: missing or invalid API key.
- `403 forbidden`: the caller may not read or mutate the requested key or inventory.
- `403 not_org_member`: `SetActiveOrg` named an organization the account does not belong to.
- `403` on `/account/data-retention`: an API key or legacy permanent token cannot access the
  account-wide retention setting. `UpdateDataRetention` needs an interactive browser session; the
  SDK does not attempt a workaround.
- `404 not_found`: the API key, usage record or deletion operation does not exist or is not owned
  by the authenticated account.
- `409 ambiguous_org`: several organizations share the requested name. Pass the id.
- `409 api_key_expired`: an expired scoped key cannot be rotated.
- `409` on `UpdateDataRetention`: the requested retention change conflicts with an accepted
  deletion operation or current account state.
- `503` on `CreateScopedAPIKey`: scoped credential creation is not activated. The shared error
  parser reports the status; see §10 for the typed-body limitation.

Retries:

- GETs (`Me`, `Orgs`, `Limits`, `APIKeys`, `APIKeyUsage`, `DataRetention`, `DeletionOperation`,
  `Secrets`) are retried on `429`, `500`, `502` and `503` by the transport.
- `CreateScopedAPIKey`, `RotateAPIKey`, `RevokeAPIKey` and `UpdateDataRetention` set
  `core.Request.NoRetry`. A lost `201` can already have created a key, rotation invalidates the old
  secret, and retention changes are destructive.
- `SetActiveOrg` and `UpdateSecrets` are POST/PATCH requests without an idempotency key, so the
  transport does not retry them.

## 7. Streaming and long-running behavior

There is no streaming endpoint in this service. Long-running behavior:

- **Deletion operations.** A sandbox or snapshot delete returns an operation id. `DeletionOperation`
  reads its current state; poll until `DeletionOperationStatusCompleted` or
  `DeletionOperationStatusBlocked`. `Stage` explains what the background purge is doing, and
  `ExpectedBy` is set while upload URLs are still valid. The target is already hidden from lists,
  restores and forks once the operation exists.
- **Data retention.** Enabling queues archived sandboxes for deletion; `QueuedSandboxes` reports
  how many this update queued. Disabling affects future archives only and cannot cancel accepted
  deletion operations.
- **API key secrets.** `CreateScopedAPIKey` and `RotateAPIKey` return the raw secret once. There is
  no endpoint to read it again. The SDK never logs or stores it; the caller must persist it in a
  secret manager before dropping the response.
- **Cancellation.** Every method honors `ctx`. A cancelled request returns the context error through
  the transport.

## 8. Examples

Reading the active wallet and checking limits before a batch:

```go
acct := account.New(d) // d is the shared core.Doer transport
limits, err := acct.Limits(ctx, account.LimitsRequest{Org: "example-org"})
if err != nil {
    return err
}
if !limits.CanStart {
    return fmt.Errorf("cannot start: %s", limits.BillingStatus)
}
```

Creating a key and storing its secret:

```go
created, err := acct.CreateScopedAPIKey(ctx, account.CreateScopedAPIKeyRequest{
    Name:   "ci",
    TTL:    "24h",
    Preset: "ci",
})
if err != nil {
    return err
}
// created.Secret is shown once. Store it in your secret manager now.
```

Switching the default wallet for the account:

```go
active, err := acct.SetActiveOrg(ctx, "example-org")
if err != nil {
    return err
}
fmt.Println(active.Active.ID)
```

Scoping a single request only:

```go
limits, err := acct.Limits(ctx, account.LimitsRequest{}, core.WithOrg("example-org"))
```

Inline fragments are illustrative.

## 9. Testing notes

- `account/account_test.go`: the API error table covers 400/401/403/404/409 across every method,
  including both 403 retention paths, plus a nil-transport guard.
- `account/me_test.go`: `Me` decoding, login and zero-data-retention flags.
- `account/orgs_test.go`: `Orgs` wallets, `SetActiveOrg` body (`org` and explicit `null`) and
  `409 ambiguous_org`.
- `account/limits_test.go`: exact `org` and `teamId` query parameters, `X-Boat-Org`, response
  windows, and the empty-query case.
- `account/apikeys_test.go`: inventory, scoped creation body and one-time secret, local scope
  validation, 503 status, rotation, revocation, usage, empty-id guards, and `NoRetry` on every
  key mutation and retention update.
- `account/retention_test.go`: policy decoding, the exact confirmation phrase, explicit `false`
  serialization, and the browser-only 403.
- `account/deletion_test.go`: operation state mapping and empty-id guard.
- `account/secrets_test.go`: legacy secret setup, full-replacement body, and empty-array
  serialization.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`. None call the live
  API and no fixture contains a real key, organization or secret.

## 10. Open questions / known gaps

- `PATCH /orgs/{org}` (rename, new-sandbox visibility) is not exposed. It is owner-only and not in
  this scope.
- `UpdateDataRetention` cannot work from an API key by design. Automation that changes retention
  must use a browser/CLI session; the SDK deliberately does not try to bypass that.
- The spec's `503` body for scoped key creation uses `error` as a string. The shared error parser
  expects an error object, so the API error carries `Status == 503` with code `http_503` instead
  of `scoped_api_key_creation_disabled`. Check the status and the catalog's
  `ScopedCreationEnabled` flag.
- There is no `WaitDeletionOperation` helper. Callers poll `DeletionOperation`; a shared polling
  utility may arrive with `jobs`.
- `/secrets` returns plaintext environment values and file contents. That is the API contract, so
  the SDK cannot redact them; callers must treat the response as secret.
- The API key scope union (`"*"` or a list) is generated as `json.RawMessage`; callers inspect it
  manually until a small accessor is added.
- No pagination exists on any account endpoint, so no page types are exposed.
