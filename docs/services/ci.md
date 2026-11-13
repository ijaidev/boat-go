# CI

Status: done

Owner: SDK maintainers

Package: `ci`

## 1. Purpose and scope

Wraps the Boat CI endpoints for GitHub Actions jobs with `runs-on: boat`:

- the CI overview: setup steps and the next one, jobs Boat did not run, linked GitHub accounts and
  their repositories, 7-day counts and stats, and the jobs
- reading one job and its GitHub log
- connecting GitHub by returning the App install or authorization link
- setting whether an account's public repositories may run jobs and which users may start them

Boat starts a sandbox runner; GitHub still owns the workflow, trigger and job state. The service is a
read-only view of that work except for `Connect` and `PublicRepos`, which change the GitHub
connection policy.

Every method takes `context.Context` first, returns errors rather than panicking, and is safe for
concurrent use. A service instance is stateless beyond its transport.

## 2. Non-goals

- Managing GitHub workflows, workflow files, runners, repositories or secrets. The service never calls
  the GitHub API directly.
- Dashboard-only billing, plan, wallet or seat changes. `402` is surfaced as an error, not managed.
- Polling loops. `JobLog` is empty before a job starts; the caller decides when to poll `Job`,
  `JobLog` or `Overview`.
- Streaming or tailing logs. The pinned spec defines a JSON response for the job log, so the SDK
  decodes it instead of returning a body.
- A separate job-list endpoint. Jobs come from `Overview`; there is no cursor pagination.
- Create → push → exec → usage → delete orchestration. That belongs in `jobs/`.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `New` | `New(core.Doer) *Service` | Wires a transport |
| `Overview` | `Overview(ctx, OverviewRequest, ...core.RequestOption) (*model.CiOverviewResponse, error)` | `account`, `repo`, `state`, `limit` filters |
| `Connect` | `Connect(ctx, ConnectRequest, ...core.RequestOption) (*model.CiConnectResponse, error)` | Never retried; returns a 30-minute install link |
| `Job` | `Job(ctx, jobID string, ...core.RequestOption) (*model.CiJob, error)` | One GitHub workflow job |
| `JobLog` | `JobLog(ctx, jobID string, ...core.RequestOption) (*model.CiJobLogResponse, error)` | JSON log, last 1 MB |
| `PublicRepos` | `PublicRepos(ctx, PublicReposRequest, ...core.RequestOption) (*model.CiPublicReposResponse, error)` | Sets the public-repository policy |

Request options apply per call. `core.WithOrg` overrides the client organization. `Connect` sets
`core.Request.NoRetry`, so even a caller-supplied `core.WithIdempotencyKey` cannot make the transport
retry it. `PublicRepos` is a set operation and needs no idempotency key.

`Overview` accepts `Limit` 0-500: zero omits the parameter and uses the server default of 200. An
unknown `State` or an out-of-range `Limit` is rejected before any request is sent.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `State` | — | `StateAll`, `StateActive`, `StateFailed`; empty means the server default |
| `OverviewRequest` | — | Query filters; not a request body |
| `ConnectRequest` | — | Hand-written body; all fields optional |
| `PublicReposRequest` | — | Hand-written body; `Account` is required |
| `Service` | — | Concrete struct; no interface |

Responses reuse the generated models: `model.CiOverviewResponse`, `model.CiConnectResponse`,
`model.CiJob`, `model.CiJobLogResponse` and `model.CiPublicReposResponse`. Job fields use
`model.CiJobState`, `model.CiJobMachineClass`, `model.CiJobSandboxState` and `model.CiRepositoryStatus`.

`PublicReposRequest.Users` has three states: nil keeps the current list, a pointer to an empty slice
allows anyone, and a non-empty slice allows only those GitHub users. The spec also accepts a
comma-separated string; the SDK exposes the list form only.

`CiConnectResponse.InstallURL` is a one-time authorization link valid for 30 minutes. Treat it as a
credential: never log it or persist it.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Overview` | `GET` | `/ci` | `account`, `repo`, `state`, `limit` |
| `Connect` | `POST` | `/ci/connect` | Body: `org`, `account`, `orgApp` |
| `PublicRepos` | `POST` | `/ci/public` | Body: `account`, `allow`, `users`, `dryRun`, `org` |
| `Job` | `GET` | `/ci/jobs/{jobId}` | `jobId` is path-escaped |
| `JobLog` | `GET` | `/ci/jobs/{jobId}/logs` | JSON `log` string |

`/ci/public` is POST-only in the pinned spec. There is no standalone GET for the public-repository
setting; read `allowPublicRepos` and `publicRepoUsers` from the overview's `installations` instead.

`Overview` sends `state=all` only when the caller sets it explicitly. An empty `State` omits the
parameter so the server default applies.

## 6. Error behavior

- `400`: an invalid filter, an unknown state or a public-repository user GitHub does not know. Fix the
  request; retrying cannot help.
- `401 unauthorized`: the key is missing, invalid or lacks `account.read` / `account.admin`.
- `402`: a billing or plan limit. Surface it to the caller; the service does not change plans.
- `403`: the caller did not link the GitHub account and is not an owner or admin of its organization.
- `404 not_found`: no linked GitHub account or job matches.
- `409`: a state conflict or concurrent change reported by the API.
- `429 rate_limited`: honor `core.RetryAfter`.
- `500`, `502`, `503`: server or upstream GitHub failures.

The pinned spec documents `400`, `401`, `403` and `404` for the CI endpoints. `402`, `409`, `429` and
the `5xx` family still map to the same `*core.APIError` when the API or gateway emits them.

GET requests (`Overview`, `Job`, `JobLog`) are retried by the transport on `429`, `500`, `502` and
`503`, respecting `Retry-After`. `Connect` is never retried. `PublicRepos` is a POST and is not
retried unless the caller supplies an `Idempotency-Key`; the operation sets a value, so repeating it
is harmless but unnecessary.

Errors are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID`, `Details` and
`RetryAfter`. Use `errors.As`, `core.IsCode` and `core.RetryAfter`; never match error strings. Error
text never contains the install URL, repository credentials or log output.

## 7. Streaming and long-running behavior

The service has no streams. `JobLog` decodes the spec's JSON payload into `CiJobLogResponse.Log`:

- the log has one timestamped line per output line, capped at the last 1 MB by the API
- it is empty before the job starts; poll `JobLog` while the job runs
- the response is buffered by the transport, bounded by its 32 MiB envelope limit, not by an
  unbounded reader

All calls honor context cancellation through the shared HTTP client. `Connect` returns the install or
authorization URL immediately; browser authorization happens outside the SDK. Repeating `Connect`
creates a new link, which is why it is never retried.

## 8. Examples

```go
overview, err := svc.Overview(ctx, ci.OverviewRequest{
    Account: "acme",
    Repo:    "acme/api",
    State:   ci.StateActive,
    Limit:   50,
})
if err != nil {
    return err
}
for _, job := range overview.Jobs {
    fmt.Println(job.ID, job.JobName, job.State)
}

link, err := svc.Connect(ctx, ci.ConnectRequest{Account: "acme"})
if err != nil {
    return err
}
// Send link.InstallURL to the account owner; never log it.
sendToAccountOwner(link.InstallURL)

log, err := svc.JobLog(ctx, jobID)
if err != nil {
    return err
}
// Consume log.Log; do not log the whole body.
_ = log.Log
```

## 9. Testing notes

- `ci/ci_test.go` covers the shared error mapping (400, 401, 402, 403, 404, 409, 429, 500, 502, 503),
  request ids, and a nil transport.
- `ci/overview_test.go` covers every filter, omission of unset filters, explicit `state=all`, response
  parsing, and rejection of an unknown state or an out-of-range limit without a request.
- `ci/connect_test.go` covers the install body, the authorize-existing body, `NoRetry` on the request,
  and a single attempt on `503` even with a caller-supplied idempotency key.
- `ci/jobs_test.go` covers job and log parsing, an empty log, a 1 MiB log, path escaping, and missing
  job ids.
- `ci/public_repos_test.go` covers the full body, omitted optional fields, empty-list clearing, the
  response, and the required account.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`; none call the live API.
- There are no live tests.

## 10. Open questions / known gaps

- `JobLog` returns JSON because the pinned spec defines `application/json`. If the API later streams
  text or binary logs, this method needs a new body-returning shape and a `Close` contract.
- `PublicRepos` exposes `users` as a list only. The spec's comma-separated string form is left to the
  caller.
- `Overview` is the only job-list surface and has no cursor. Jobs beyond `limit` (at most 500) require
  narrowing by `account`, `repo` or `state`.
- `Connect` always creates a fresh authorization link; the SDK does not cache it.
- The overview's setup steps, problems and stats are generated structs with pointer fields. A field
  the API omits is nil; callers must handle that.
