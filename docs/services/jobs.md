# Jobs

Status: done

Owner: SDK maintainers

Package: `jobs`

## 1. Purpose and scope

`jobs` owns workflows that span several Boat services. It is the only package allowed to import
multiple domain services; services never import `jobs`.

It provides:

- `Run`: create a sandbox, wait for it, optionally push a local git work tree, run a command, read
  usage and delete the sandbox;
- `Push`: build a git-aware archive, upload it through a signed URL and extract it in a sandbox;
- `Pull`: stream one sandbox file through a signed download URL;
- `PullSnapshot`: stream one file, folder or whole snapshot from snapshot storage.

Every method takes a `context.Context` first, returns errors instead of panicking, and keeps signed
URLs out of errors. Command exit codes are results, not SDK errors.

## 2. Non-goals

- Replacing the domain services. `jobs` composes them; it does not reimplement their endpoints.
- Automatic retries of commands. A lost command response may still have run.
- Sandbox lifecycle management beyond one `Run` call. Use `sandboxes` directly for resume, fork,
  stop or long-lived sandboxes.
- Archive extraction, chunk reassembly or tar parsing for downloads. `Pull` and `PullSnapshot`
  stream the bytes the API returns.
- A local `.git` directory transfer. `Push` sends the working tree, not repository history.
- Interactive terminals, port forwarding, agents or CI setup. Those stay in their services.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `New` | `New(Config) *Service` | Wires the services and HTTP client |
| `Run` | `Run(ctx, RunRequest) (*RunResult, error)` | Create → wait → optional push → exec → usage → delete |
| `Push` | `Push(ctx, sandboxID, sourceDir, remoteDir string, ...core.RequestOption) error` | Git-aware archive upload and extraction |
| `Pull` | `Pull(ctx, sandboxID, remotePath string, w io.Writer, ...core.RequestOption) error` | Signed URL, streamed to `w` |
| `PullSnapshot` | `PullSnapshot(ctx, snapshotID, path string, w io.Writer, ...core.RequestOption) error` | Snapshot file stream, closed once |

`Run` has no per-call request options; configure the organization on the transport passed to the
services. `Push`, `Pull` and `PullSnapshot` forward `core.RequestOption`s to their Boat API calls,
not to the signed URL request. `Config.HTTPClient` is used for the signed PUT/GET and defaults to
`http.DefaultClient`.

`Run` behavior:

- A command exit code, including a non-zero one, is returned in `RunResult.ExitCode` with a nil
  error.
- An attach error frame or a stream failure is returned as an error.
- Usage is best effort. A failed usage request leaves `RunResult.Usage` nil, stores the error in
  `RunResult.UsageError` and still returns the command result.
- The sandbox is deleted on success unless `Keep` is true.
- On failure it is deleted unless `Keep` or `KeepOnError` is true.
- Cleanup uses `context.WithoutCancel` plus a 30-second timeout, so a canceled caller still cleans
  up. A cleanup failure is joined with the primary error with `errors.Join`.

`Push` requires git on the local machine and `tar` inside the sandbox.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `Config` | — | `Sandboxes`, `Commands`, `Files`, `Snapshots`, `HTTPClient` |
| `RunRequest` | — | Wraps `sandboxes.CreateRequest`; command, cwd, source dir, writers and keep flags |
| `RunResult` | `model.SandboxUsageResponse` | Sandbox id, exit code, signal, usage, pushed flag and `UsageError` |

`RunRequest.SourceDir` is optional; an empty value skips `Push`. `RemoteDir` is the extraction
directory and defaults to `.`. `Stdout` and `Stderr` may be nil. `Keep` keeps the sandbox on every
outcome; `KeepOnError` keeps it only when a step after creation fails.

`RunResult.SandboxID` is set as soon as creation succeeds, including on a later error.
`RunResult.UsageError` is informational and does not fail the run.

## 5. Endpoints and mapping

`jobs` has no endpoints of its own. It orchestrates existing service calls:

| SDK method | Services and calls | Notes |
|---|---|---|
| `Run` | `sandboxes.Create`, `sandboxes.WaitUntilReady`, optional `Push`, `commands.RunCommand`, `sandboxes.Usage`, `sandboxes.Delete` | Start uses `detached: true`; attach streams NDJSON |
| `Push` | `git ls-files`, `files.UploadURL`, signed `PUT`, `commands.Run` | Archive is `.boat-push-<random>.tgz` under `/home/user` |
| `Pull` | `files.DownloadURL`, signed `GET` | No 5 MiB inline cap |
| `PullSnapshot` | `snapshots.File` | Empty path means the whole snapshot |

`Push` runs:

```sh
git ls-files -z --cached --others --exclude-standard
```

It includes tracked and untracked-not-ignored files, skips deleted paths and submodule
directories, and preserves regular-file mode bits and symlinks without following links. The tar and
gzip archive is built in Go and written to a temporary file that is removed afterward. Extraction
runs:

```sh
mkdir -p <remote> && tar xzf <archive> -C <remote> && rm <archive>
```

Every path is single-quoted with embedded quotes escaped. The extraction command has a 600-second
timeout.

## 6. Error behavior

- Command exit codes are results. An error frame, malformed stream, transport failure or output
  writer failure is an SDK error.
- Usage failures are best effort and never replace a successful command result.
- Signed upload and download URLs are secrets. `jobs` never includes them in error text, and signed
  request failures use a generic message while preserving the underlying error for `errors.Is` and
  `errors.As`.
- A signed URL response with a non-2xx status reports only the status code.
- `Push` reports git failures, invalid source paths, upload failures and a non-zero extraction exit
  code as errors.
- `Pull` and `PullSnapshot` require a non-nil writer and fail before making a Boat API call without
  one.
- On `Run` failure, cleanup errors are joined with the primary error; on success, a cleanup failure
  is returned with the partial result.
- The transport's normal retry policy applies to the individual service calls. `jobs` adds no
  command retry.

## 7. Streaming and long-running behavior

`Run` uses `commands.RunCommand`, which starts a detached command, attaches to its NDJSON output,
hides heartbeat frames and reconnects on protocol `reconnect` frames. It sets no command time limit
beyond the caller's context. `stdout` and `stderr` events are written incrementally to the provided
writers.

`Pull` and `PullSnapshot` stream with `io.Copy`; they never buffer the whole file. `Pull` closes the
signed download body after the copy. `PullSnapshot` closes the snapshot body exactly once, including
on copy failure.

Cancelling the caller's context aborts the current API call or stream. Cleanup after a `Run`
failure still uses a fresh bounded context.

## 8. Examples

```go
result, err := client.Jobs.Run(ctx, jobs.RunRequest{
    Create:    sandboxes.CreateRequest{TTLSeconds: core.Some(900)},
    Command:   "go test ./...",
    SourceDir: ".",
    Stdout:    os.Stdout,
    Stderr:    os.Stderr,
})
if err != nil {
    return err
}
fmt.Println("exit code:", result.ExitCode)
```

Uploading and extracting without running a command:

```go
if err := client.Jobs.Push(ctx, sandboxID, ".", "workspace"); err != nil {
    return err
}
```

Downloading a snapshot file:

```go
f, err := os.Create("result.json")
if err != nil {
    return err
}
defer f.Close()
if err := client.Jobs.PullSnapshot(ctx, snapshotID, "workspace/result.json", f); err != nil {
    return err
}
```

Inline fragments are illustrative. The root client does not expose `Jobs` yet; wire a `jobs.Service`
from a `jobs.Config` until the facade follows.

## 9. Testing notes

- `jobs/run_test.go`: full success sequence, non-zero exit, nil writers, usage failure, cleanup
  error joining, create failure with no delete, wait failure cleanup and the `Keep`/`KeepOnError`
  matrix.
- `jobs/push_test.go`: a temporary git repository with tracked, untracked, ignored, deleted,
  executable, symlink and gitlink entries; uploaded gzip/tar contents and modes; remote path shell
  quoting; extraction failure; upload failure and temp-file removal; git and configuration errors.
- `jobs/pull_test.go`: signed download streaming, body close counts, signed URL redaction, download
  errors and snapshot streaming/error close paths.
- All tests use `httptest` and temporary directories. None call the live Boat API, and no live tests
  exist yet.

## 10. Open questions / known gaps

- The root `boat.Client` does not have a `Jobs` field in this change. A follow-up wires
  `jobs.New` into the facade.
- `Run` has no per-call `core.RequestOption` argument; organization scope must come from the
  transport configuration.
- The cleanup timeout is a fixed 30 seconds. It is not configurable yet.
- If extraction fails, the remote archive is left behind because the `&& rm` does not run. The
  sandbox is normally deleted unless the caller keeps it.
- `Push` sends the working tree only. It does not transfer `.git`, submodules or files ignored by
  git, and it does not apply `.gitattributes` export rules.
- `Pull` does not resume a partial download; callers can use `files.DownloadURL` directly with a
  `Range` header when they need offsets.
- `PullSnapshot` streams tar bytes for folders and whole snapshots. It does not extract or reassemble
  chunk manifests.
- `docs/services/README.md` still lists Jobs as planned because this change is scoped to the package
  and its own doc.
