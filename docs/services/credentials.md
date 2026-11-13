# Credentials

Status: done

Owner: SDK maintainers

Package: `credentials`

## 1. Purpose and scope

Wraps the Boat AI credentials API:

- credential profiles, their names, defaults and organization access
- API keys and sign-in credentials on a profile, addressed by `CredentialId`
- several accounts per credential, with active-account switching
- custom APIs exposed to sandboxes as environment variables
- the AI credentials proxy switch and its price
- account and sandbox AI usage
- blocking and unblocking a sandbox's AI placeholders

A credential profile is a named set of AI credentials. A sandbox that uses a profile can call model
providers without the caller copying keys into the sandbox. Credential ids are the generated
`model.CredentialID` values, such as `openai-api-key` or `claude-subscription`.

### Security notes

- API keys, subscription tokens and custom API values are secrets. They are write-only in the API and
  are never returned by it. The SDK never logs them and never adds them to error text.
- `SetKey`, `AddAccount` and `SetCustomAPI` accept secret values. Do not print request bodies, wrap
  them in logs, or copy them into telemetry.
- The AI proxy keeps real provider keys out of sandboxes. With the proxy on, a sandbox receives
  placeholders that point at Boat; Boat swaps in the real key per request. Turning the proxy on is
  billable.
- `SetProxy` and `SetSandboxAccess` are security-sensitive changes. They affect every sandbox that
  uses the account's profile, or the named sandbox, respectively. Restrict who can call them.

## 2. Non-goals

- Connecting sign-ins (`*-subscription`) or Bedrock (`*-bedrock`) credentials. Those flows are
  dashboard-only; this service can list and remove them, but not connect them.
- Proxying AI requests. The SDK manages the switch and reports usage; it does not forward provider
  traffic.
- Storing secrets locally, caching decrypted keys or reading them back. They are write-only.
- Sandbox creation, environment management and credential-profile assignment to environments. Those
  belong in `sandboxes` and `environments`.
- Billing, wallet and plan changes. Usage is read-only.
- Cross-service flows such as "create a sandbox with this profile and wait for usage". Those belong
  in `jobs/`.
- Retrying mutations. Account and key changes are not idempotent and are never retried.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `New` | `New(core.Doer) *Service` | Wires a transport |
| `ListProfiles` | `ListProfiles(ctx, ...core.RequestOption) ([]model.CredentialProfile, error)` | No cursor pagination |
| `CreateProfile` | `CreateProfile(ctx, CreateProfileRequest, ...core.RequestOption) (*model.CredentialProfile, error)` | 201; never retried |
| `UpdateProfile` | `UpdateProfile(ctx, profile string, UpdateProfileRequest, ...core.RequestOption) (*model.CredentialProfile, error)` | Rename, default, org access |
| `DeleteProfile` | `DeleteProfile(ctx, profile string, ...core.RequestOption) error` | 409 `is_default` / `in_use` |
| `SetKey` | `SetKey(ctx, profile string, credential model.CredentialID, SetKeyRequest, ...core.RequestOption) (*model.CredentialProfile, error)` | API keys only |
| `DeleteKey` | `DeleteKey(ctx, profile string, credential model.CredentialID, ...core.RequestOption) (*model.CredentialProfile, error)` | Also removes sign-ins |
| `ListAccounts` | `ListAccounts(ctx, profile string, credential model.CredentialID, ...core.RequestOption) ([]model.CredentialAccount, error)` | Labels only, no secrets |
| `AddAccount` | `AddAccount(ctx, profile string, credential model.CredentialID, SetKeyRequest, ...core.RequestOption) ([]model.CredentialAccount, error)` | 201; API keys only |
| `SwitchAccount` | `SwitchAccount(ctx, profile string, credential model.CredentialID, account string, ...core.RequestOption) ([]model.CredentialAccount, error)` | Applies to running sandboxes at once |
| `DeleteAccount` | `DeleteAccount(ctx, profile string, credential model.CredentialID, account string, ...core.RequestOption) ([]model.CredentialAccount, error)` | Returns the accounts left |
| `SetCustomAPI` | `SetCustomAPI(ctx, profile, name string, SetCustomAPIRequest, ...core.RequestOption) (*model.CredentialProfile, error)` | Adds or replaces |
| `DeleteCustomAPI` | `DeleteCustomAPI(ctx, profile, name string, ...core.RequestOption) (*model.CredentialProfile, error)` | |
| `GetProxy` | `GetProxy(ctx, GetProxyRequest, ...core.RequestOption) (*model.AiProxySettings, error)` | Optional `org` query |
| `SetProxy` | `SetProxy(ctx, SetProxyRequest, ...core.RequestOption) (*model.AiProxySettings, error)` | Billable, security-sensitive |
| `Usage` | `Usage(ctx, UsageRequest, ...core.RequestOption) (*model.AccountAiUsageResponse, error)` | Read-only |
| `SandboxUsage` | `SandboxUsage(ctx, sandboxID string, SandboxUsageRequest, ...core.RequestOption) (*model.SandboxAiUsageResponse, error)` | Read-only |
| `SetSandboxAccess` | `SetSandboxAccess(ctx, sandboxID string, SetSandboxAccessRequest, ...core.RequestOption) (*model.SandboxAiAccessResponse, error)` | Blocks or unblocks placeholders |

Every method takes a `context.Context` first and accepts per-call `core.RequestOption`s such as
`core.WithOrg`. Services are safe for concurrent use; there is no mutable package state.

Every mutation sets `core.Request.NoRetry`, so the transport never retries it, even when the caller
supplies `core.WithIdempotencyKey`. The service never generates an idempotency key. GET reads are
retried by the transport on `429`, `500`, `502` and `503`.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `CreateProfileRequest` | — | `Name` required; empty `Org`/`CopyFrom` are omitted |
| `UpdateProfileRequest` | — | `IsDefault *bool` sends `true` only; `Access` for org profiles |
| `SetKeyRequest` | — | Body of `SetKey` and `AddAccount`; `KeepCurrent *bool` |
| `SetCustomAPIRequest` | — | `Value` and `BaseURL` required |
| `GetProxyRequest` | — | `Org` query only |
| `SetProxyRequest` | — | `Enabled` body, `Org` query |
| `UsageRequest` | — | `Days`, `Profile`, `Org` query filters |
| `SandboxUsageRequest` | — | `Days` query filter |
| `SetSandboxAccessRequest` | — | `Blocked` body |
| `model.CredentialID` | — | Generated credential enum |
| `model.CredentialProfile` | — | Profile, owner, connected ids, names only |
| `model.CredentialAccount` | — | Account label and `Active` flag |
| `model.AiProxySettings` | — | Switch, price and proxy URL |
| `model.AccountAiUsageResponse` | — | Per-day, per-credential usage and totals |
| `model.SandboxAiUsageResponse` | — | One sandbox's usage and blocked state |
| `model.SandboxAiAccessResponse` | — | New blocked state |

`UsageRequest.Days` and `SandboxUsageRequest.Days` accept 1-365. Zero omits the parameter and uses
the server default of 30. Any other value is forwarded as-is; the SDK does not range-check it, and the
API rejects one outside 1-365.

`SetProxyRequest.Org` is a query parameter, not a JSON field. `GetProxyRequest.Org` is the same
billing-wallet selector. Both accept an organization id, an organization name, or `personal`, and
win over the `X-Boat-Org` header set by `core.WithOrg`.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `ListProfiles` | `GET` | `/credential-profiles` | |
| `CreateProfile` | `POST` | `/credential-profiles` | `201`; `org`, `copyFrom` in the body |
| `UpdateProfile` | `PUT` | `/credential-profiles/{profile}` | `name`, `isDefault`, `access` |
| `DeleteProfile` | `DELETE` | `/credential-profiles/{profile}` | |
| `SetKey` | `PUT` | `/credential-profiles/{profile}/keys/{credential}` | `value`, `keepCurrent` |
| `DeleteKey` | `DELETE` | `/credential-profiles/{profile}/keys/{credential}` | |
| `ListAccounts` | `GET` | `/credential-profiles/{profile}/accounts/{credential}` | |
| `AddAccount` | `POST` | `/credential-profiles/{profile}/accounts/{credential}` | `201`; `value`, `keepCurrent` |
| `SwitchAccount` | `PUT` | `/credential-profiles/{profile}/accounts/{credential}/{account}/active` | No body |
| `DeleteAccount` | `DELETE` | `/credential-profiles/{profile}/accounts/{credential}/{account}` | |
| `SetCustomAPI` | `PUT` | `/credential-profiles/{profile}/custom-apis/{name}` | `value`, `baseUrl` |
| `DeleteCustomAPI` | `DELETE` | `/credential-profiles/{profile}/custom-apis/{name}` | |
| `GetProxy` | `GET` | `/ai-proxy` | `org` |
| `SetProxy` | `PUT` | `/ai-proxy` | `org`; `enabled` body |
| `Usage` | `GET` | `/ai-usage` | `days`, `profile`, `org` |
| `SandboxUsage` | `GET` | `/sandboxes/{sandboxId}/ai-usage` | `days` |
| `SetSandboxAccess` | `PUT` | `/sandboxes/{sandboxId}/ai-access` | `blocked` body |

A profile path segment is either a profile id or a profile name. Account ids are the opaque `id`
values returned by `ListAccounts`.

`ListProfiles` and the account list endpoints return every item in one response; the v1 API has no
cursor for them.

## 6. Error behavior

- `400 invalid_*`: malformed profile, custom API or access body. Fix the request; retrying cannot
  help.
- `400 not_an_api_key` on `SetKey` or `AddAccount`: the credential is a sign-in
  (`*-subscription`) or Bedrock (`*-bedrock`). Connect or add those on the dashboard; use
  `DeleteKey` to remove one.
- `401 unauthorized`: the key is missing, invalid or lacks the required scope.
- `403 forbidden`: the key lacks `account.admin`, the caller only uses an organization profile, or
  the caller is not the organization owner for a proxy change.
- `404 not_found` / `credential_profile_not_found`: the profile, credential, account or sandbox does
  not exist.
- `409 is_default`: a default profile cannot be deleted. Make another profile the default first.
- `409 in_use`: an environment names the profile. Remove that reference first.
- `409 ambiguous_org`: the `org` query matched two organizations by name. Pass an organization id.
- `429 rate_limited`: `core.RetryAfter` carries the server hint. GETs are retried by the transport;
  mutations are not.
- `500`, `502`, `503`: GETs are retried by the transport. Mutations surface the first error.

All errors are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID`, `Details` and
`RetryAfter`. Use `errors.As`, `core.IsCode` and `core.RetryAfter`; never match error strings.

Secret values are write-only. The API does not echo them, and the SDK never puts request bodies,
headers or credentials into error text.

## 7. Streaming and long-running behavior

- No endpoint in this service streams. There is no NDJSON, long-poll or SSE response.
- `Usage` and `SandboxUsage` are read-only snapshots. The SDK does not poll them or wait for a
  billing period to close.
- `SetProxy` changes the setting for the account. Running sandboxes switch at their next start.
- `SwitchAccount` applies to every sandbox that uses the profile at once, including running ones,
  with no restart.
- `SetSandboxAccess` works while the sandbox is running or stopped; the proxy starts refusing or
  accepting that sandbox's placeholder requests immediately.
- Every call honors context cancellation and deadlines through the shared transport.

## 8. Examples

Create a profile, set an API key, and switch back to a previous key. The fragment assumes a
`*credentials.Service` in `svc`; see §10 for root-client wiring:

```go
profile, err := svc.CreateProfile(ctx, credentials.CreateProfileRequest{
    Name: "work",
    CopyFrom: "default",
})
if err != nil {
    return err
}

profile, err = svc.SetKey(ctx, profile.ID, model.OpenaiAPIKey,
    credentials.SetKeyRequest{
        Value:       key,
        KeepCurrent: core.Ptr(true),
    })
if err != nil {
    return err
}

accounts, err := svc.ListAccounts(ctx, profile.ID, model.OpenaiAPIKey)
if err != nil {
    return err
}
_, err = svc.SwitchAccount(ctx, profile.ID, model.OpenaiAPIKey, accounts[0].ID)
```

Turn the proxy on for an organization and block one sandbox:

```go
settings, err := svc.SetProxy(ctx, credentials.SetProxyRequest{
    Enabled: true,
    Org:     "team_example",
})
if err != nil {
    return err
}
_ = settings.ProxyURL

_, err = svc.SetSandboxAccess(ctx, sandboxID,
    credentials.SetSandboxAccessRequest{Blocked: true})
if err != nil {
    return err
}
```

`settings.ProxyURL` is a public base URL, not a secret. Never log key values, custom API values or
request bodies from `SetKey`, `AddAccount` or `SetCustomAPI`.

## 9. Testing notes

- `credentials/profiles_test.go` covers list, create (including omitted optional fields), update
  with `isDefault` and organization access, and delete.
- `credentials/keys_test.go` covers key bodies, `keepCurrent`, sign-in removal and proof that a key
  mutation is not retried even with a caller-supplied `Idempotency-Key`.
- `credentials/accounts_test.go` covers account listing, adding a key, active-account switching and
  deleting an account.
- `credentials/custom_apis_test.go` covers custom API upsert and removal.
- `credentials/proxy_test.go` covers the `org` query, omitted `org`, and the `enabled` body with no
  leaked query fields.
- `credentials/usage_test.go` covers `days`, `profile` and `org` filters, omitted defaults, and the
  sandbox usage response.
- `credentials/access_test.go` covers blocking and unblocking.
- `credentials/errors_test.go` maps documented 400, 401, 403, 404, 409 and 429 responses to
  `*core.APIError`, preserves `RequestID`, carries `Retry-After`, and asserts that no secret value
  appears in error text.
- `credentials/service_test.go` covers nil transports, required arguments (no request is sent), and
  `NoRetry` on every mutation.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`; none call the live API.
  There is no live-test coverage yet.

## 10. Open questions / known gaps

- The root `boat.Client` does not expose `Credentials` yet. Construct the service with
  `credentials.New(transport)` until the facade is wired.
- The SDK validates required identity and secret fields locally but leaves the `days` range, the
  custom API name pattern and the base URL format to the API.
- There is no cursor pagination on profile or account lists because the v1 API has none.
- Sign-in and Bedrock credentials cannot be connected through the API; only removal is available.
- `Usage` rows are keyed by credential string (`CredentialId` or `custom:<NAME>`), not by a typed
  enum, because custom APIs share the field.
- No live tests. A safe live test would need an account with `account.admin`, would create and
  delete a temporary profile, and must never print a key value.
