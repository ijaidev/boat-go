# Environments

Status: done

Owner: SDK maintainers

Package: `environments`

## 1. Purpose and scope

Wraps the Boat sandbox environment and repository endpoints:

- list, create, update, soft-delete and upgrade environments
- set and remove individual environment variables
- write and remove individual secret files
- add and remove repositories on an environment
- list GitHub repositories and select the repository used by future sandboxes

An environment is a reusable start configuration: repositories, environment variables, secret
files, setup behavior and credential toggles. Editing an environment mints a new immutable version.
Sandboxes already running stay pinned to the version they started on until `Upgrade` moves them.

The `/environments` routes accept the spec's `org` query parameter. They do **not** use the
`X-Boat-Org` header, so `core.WithOrg` has no effect there. Use the `Org` field on a request type or
`environments.WithOrg`. The `/repos` routes have no org parameter; they operate on the default
environment.

## 2. Non-goals

- Starting sandboxes. Use `sandboxes.Create` with `Environment` or `Environment` plus `NoEnv`.
- Credential profiles and AI credential configuration. Those belong in `credentials/`.
- Organization membership and billing. Those belong in `account/`.
- Git-aware clone, commit, push and setup orchestration. Those belong in `jobs/`.
- Pagination for `/environments`. The endpoint returns every non-deleted environment in one page.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `List` | `List(ctx, ListRequest, ...core.RequestOption) ([]model.SandboxEnvironment, error)` | No cursor pagination |
| `Create` | `Create(ctx, CreateRequest, ...core.RequestOption) ([]model.SandboxEnvironment, error)` | Returns the full list; never retried |
| `Update` | `Update(ctx, environmentID string, UpdateRequest, ...core.RequestOption) (*model.SandboxEnvironment, error)` | Mints a new version |
| `Delete` | `Delete(ctx, environmentID string, ...core.RequestOption) (*model.SandboxEnvironment, error)` | Soft delete; may be nil |
| `Upgrade` | `Upgrade(ctx, environmentID string, UpgradeRequest, ...core.RequestOption) (*model.UpgradeSandboxEnvironmentResponse, error)` | All or selected sandboxes |
| `SetVar` | `SetVar(ctx, environmentID, key, value string, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | Value is a secret |
| `DeleteVar` | `DeleteVar(ctx, environmentID, key string, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | |
| `SetSecretFile` | `SetSecretFile(ctx, environmentID string, SecretFileRequest, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | Contents is a secret |
| `DeleteSecretFile` | `DeleteSecretFile(ctx, environmentID, path string, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | `path` is a query parameter |
| `AddRepo` | `AddRepo(ctx, environmentID string, AddRepoRequest, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | Adds or updates the base branch |
| `RemoveRepo` | `RemoveRepo(ctx, environmentID, repositoryID string, ...core.RequestOption) (*model.EnvironmentItemChangeResponse, error)` | |
| `ListRepos` | `ListRepos(ctx, ListReposRequest, ...core.RequestOption) (*model.ReposResponse, error)` | Cursor pagination in `PageInfo` |
| `SelectRepo` | `SelectRepo(ctx, SelectRepoRequest, ...core.RequestOption) (*model.RepoSelectionResponse, error)` | Idempotent; future sandboxes only |

Every method takes `context.Context` first and accepts per-call `core.RequestOption`s. Services are
safe for concurrent use and hold no mutable package state. `WithOrg(org)` sets the `org` query
parameter for methods that do not take an `Org` field, such as `Delete`, `SetVar` and `RemoveRepo`.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `ListRequest` | — | `Org` |
| `CreateRequest` | — | Body `name`; `Org` is query-only |
| `UpdateRequest` | — | Hand-written for optional flags, nullable `credentialProfileId`, full-list replacement |
| `RepositorySelection` | — | One entry of `UpdateRequest.Repositories` |
| `UpgradeRequest` | — | `AgentIDs` maps to `agentIds` |
| `SecretFileRequest` | — | Body `path` and `contents`; `Org` is query-only |
| `AddRepoRequest` | — | Body `repositoryId`, optional `baseBranch` |
| `ListReposRequest` | — | `sync`, `limit`, `cursor`, `sort`, `q`, `selected` |
| `SelectRepoRequest` | — | Body `repositoryId`, optional `baseBranch` |

`UpdateRequest.CredentialProfileID` is a tri-state `*core.Optional[string]`: nil omits the field,
`core.Some(value)` sets a profile, and `core.Null[string]()` clears it to the owner's default.
`SecretFiles` and `Repositories` are pointers so an empty slice can replace the current list.
All other fields use pointers or omit-empty values so an omitted field is not sent.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `List` | `GET` | `/environments` | `org` |
| `Create` | `POST` | `/environments` | `org`; `200`; returns full list |
| `Update` | `PUT` | `/environments/{environmentId}` | `org`; `200` |
| `Delete` | `DELETE` | `/environments/{environmentId}` | `org`; `200`; soft delete |
| `Upgrade` | `POST` | `/environments/{environmentId}/upgrade` | `org`; body `agentIds` optional |
| `SetVar` | `PUT` | `/environments/{environmentId}/vars/{key}` | body `value` |
| `DeleteVar` | `DELETE` | `/environments/{environmentId}/vars/{key}` | |
| `SetSecretFile` | `PUT` | `/environments/{environmentId}/secret-files` | body `path`, `contents` |
| `DeleteSecretFile` | `DELETE` | `/environments/{environmentId}/secret-files` | query `path` |
| `AddRepo` | `POST` | `/environments/{environmentId}/repos` | body `repositoryId`, `baseBranch` |
| `RemoveRepo` | `DELETE` | `/environments/{environmentId}/repos/{repositoryId}` | |
| `ListRepos` | `GET` | `/repos` | `sync`, `limit`, `cursor`, `sort`, `q`, `selected` |
| `SelectRepo` | `POST` | `/repos` | body `repositoryId`, `baseBranch` |

Environment ids, variable keys, repository ids and secret-file paths are escaped as path segments.
`Query` in `ListReposRequest` is sent as the spec's `q` parameter. `List` omits `org` when empty,
so the account's active wallet applies. On `/environments` routes only the query parameter selects
an organization.

## 6. Error behavior

- `400 invalid_*`: invalid environment name, variable key, secret-file path, repository selection
  or upgrade selector. Fix the request; retrying cannot help.
- `401 unauthorized`: invalid or missing key.
- `402 billing_required`: the account cannot create or edit environments.
- `403 forbidden`: the caller may read but not change the organization's environments.
- `404 not_found`: the environment, variable or repository does not exist.
- `409 conflict`: duplicate environment name, deleting the default environment, deleting the last
  environment, or `ambiguous_org` when two organizations share a name.
- `429 rate_limited`: `core.RetryAfter` carries the wait. `List` and `ListRepos` are retried by the
  transport; mutations surface the first error.
- `500`, `502`, `503`: retried for `List` and `ListRepos`; mutations surface the first error.

`Create`, `Update`, `Delete`, `Upgrade`, `SetVar`, `DeleteVar`, `SetSecretFile`,
`DeleteSecretFile`, `AddRepo` and `RemoveRepo` set `core.Request.NoRetry`, because a retry could
duplicate a side effect or mint another version. `SelectRepo` is idempotent and is not marked
`NoRetry`; like every POST, the transport still only retries it when the caller supplies an
`Idempotency-Key`.

All API failures are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID`, `Details` and
`RetryAfter`. Use `errors.As`, `core.IsCode` and `core.RetryAfter`. Environment variable values and
secret file contents never appear in SDK error text or logs.

## 7. Streaming and long-running behavior

No stream and no polling helper. Every call is a normal JSON request and returns when the server
answers. `Upgrade` may take longer because it hot-pushes the new configuration to running
sandboxes; the response reports `upgraded`, `upgradedLive`, `upgradedStopped` and `failed` counts.
The SDK does not wait for a sandbox to pick up a new version. Cancellation is handled by the
request context.

## 8. Examples

These fragments assume `svc` is an `*environments.Service`.

```go
envs, err := svc.List(ctx, environments.ListRequest{Org: "team_example"})
if err != nil {
    return err
}

_, err = svc.SetVar(ctx, envID, "API_TOKEN", token,
    environments.WithOrg("team_example"))
if err != nil {
    return err
}

repos, err := svc.ListRepos(ctx, environments.ListReposRequest{
    Query: "web",
})
if err != nil {
    return err
}
for _, installation := range repos.Installations {
    for _, repo := range *installation.Repositories {
        // repo.DatabaseID is the repositoryId used by AddRepo and SelectRepo.
        _ = repo
    }
}
```

Find a created environment by name, because `Create` returns the server's full list:

```go
envs, err := svc.Create(ctx, environments.CreateRequest{
    Name: "customer-demos",
})
if err != nil {
    return err
}
for _, env := range envs {
    if env.Name == "customer-demos" {
        envID = env.ID.String()
    }
}
```

## 9. Testing notes

- Tests live next to the code: `environments/lifecycle_test.go`, `vars_test.go`, `secrets_test.go`,
  `repos_test.go` and `errors_test.go`.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`; none call the live API.
- Request shape is asserted: method, path escaping, `org` and repository query names, headers,
  exact JSON bodies, omitted fields, explicit `credentialProfileId: null`, empty lists and the
  optional upgrade selector.
- Error coverage maps `400`, `401`, `402`, `403`, `404`, `409` and `429` to `*core.APIError` and
  checks the request id.
- Retry coverage asserts that `List` retries a transient `503` and that every mutation reaches the
  server exactly once even when the caller supplies an `Idempotency-Key`.
- Secret tests assert that variable and secret-file values do not appear in error text.
- No live-test coverage. Environment changes would need a real account and are covered by the
  compiled request contract instead.

## 10. Open questions / known gaps

- The root `boat.Client` does not expose `Environments` yet; wiring it is a separate facade change.
- `Create` returns the full environment list because that is the wire response. Callers locate the
  new environment by `Name`.
- `List` has no cursor. If the API later paginates `/environments`, this service will need a page
  type and a breaking signature change.
- `/repos` and `SelectRepo` have no `org` parameter and always operate on the default environment.
  Use `AddRepo`, `Update` or `RemoveRepo` to edit an organization environment directly.
- `Delete` returns `nil` when the response omits the deleted environment; the delete still
  succeeded when no error is returned.
- The spec describes `Update` flags and contents as a version change but does not mark the body
  required. The SDK sends `{}` for an empty update and lets the server decide.
