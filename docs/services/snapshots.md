# Snapshots

Status: done

Owner: SDK maintainers

Package: `snapshots`

## 1. Purpose and scope

Wraps captured sandbox state: the snapshots Boat takes automatically, and the saved snapshots a
caller names and keeps. A snapshot is a frozen disk image; a saved snapshot is a stable name that
points at one, independent of the source sandbox's later life.

The service covers:

- listing snapshots for the account or for one sandbox;
- reading the latest snapshot of a sandbox;
- browsing the captured file tree;
- streaming one file, a folder or the whole snapshot as a tar;
- reading the signed chunk manifest used to reconstruct a filesystem;
- deleting one snapshot or every snapshot of a sandbox;
- listing, saving, reading and deleting saved snapshots;
- creating, listing and revoking saved-snapshot share codes.

## 2. Non-goals

- The spec's `named-snapshots` endpoints. They are undocumented in the main API reference and are
  an open question (see §10).
- Creating sandboxes from a saved snapshot. Use `sandboxes.Create` with `CreateRequest.From`, or
  `sandboxes.CreateMany` for a batch.
- Stop, resume and fork. Those belong to `sandboxes`; `DeleteForSandbox` does not stop the sandbox.
- Polling helpers. `Save` and both delete methods are asynchronous; callers poll `GetSaved` or the
  account deletion-operation endpoint.
- Reassembling downloaded chunks or extracting tars. That is caller or CLI work, not SDK behavior.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `List` | `List(ctx, ListRequest, ...core.RequestOption) (core.Page[model.SnapshotSummary], error)` | Account-wide, cursor paginated |
| `ListForSandbox` | `ListForSandbox(ctx, sandboxID, ListRequest, ...core.RequestOption) (core.Page[model.SnapshotSummary], error)` | One sandbox |
| `Latest` | `Latest(ctx, sandboxID, ...core.RequestOption) (*model.SnapshotSummary, error)` | `nil, nil` when the sandbox has no snapshot |
| `Tree` | `Tree(ctx, snapshotID, ...core.RequestOption) (*model.SnapshotTreeResponse, error)` | Flat path listing |
| `File` | `File(ctx, snapshotID, path, ...core.RequestOption) (*Download, error)` | Binary stream; empty path means the whole snapshot |
| `Download` | `Download(ctx, snapshotID, ...core.RequestOption) (*Download, error)` | Whole snapshot as a tar |
| `DownloadManifest` | `DownloadManifest(ctx, snapshotID, ...core.RequestOption) (*model.SnapshotDownloadResponse, error)` | Signed chunk URLs |
| `Delete` | `Delete(ctx, snapshotID, ...core.RequestOption) (*model.DeletionOperation, error)` | Async; sends the confirmation header |
| `DeleteForSandbox` | `DeleteForSandbox(ctx, sandboxID, ...core.RequestOption) ([]model.DeletionOperation, error)` | One operation per snapshot |
| `ListSaved` | `ListSaved(ctx, SavedListRequest, ...core.RequestOption) (core.Page[model.SavedSnapshot], error)` | One page; the endpoint has no cursor |
| `Save` | `Save(ctx, sandboxID, name, ...core.RequestOption) (*model.SavedSnapshot, error)` | `202`; async, returns `saving` |
| `GetSaved` | `GetSaved(ctx, name, ...core.RequestOption) (*model.SavedSnapshot, error)` | Poll target after `Save` |
| `DeleteSaved` | `DeleteSaved(ctx, name, ...core.RequestOption) error` | Removes the artifact and frees storage |
| `CreateShare` | `CreateShare(ctx, name, ...core.RequestOption) (*model.NamedSnapshotShareCreatedResponse, error)` | The only response carrying the full code |
| `ListShares` | `ListShares(ctx, name, ...core.RequestOption) ([]model.NamedSnapshotShare, error)` | Prefixes only |
| `RevokeShare` | `RevokeShare(ctx, name, shareID, ...core.RequestOption) (*model.NamedSnapshotShareRevokedResponse, error)` | Code stops deploying |

Services are safe for concurrent use. Every I/O method takes `context.Context` first and returns
errors. Request options apply per call; `core.WithOrg` scopes a request to an organization.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `ListRequest` | — | Hand-written; `limit`, `cursor`, `sort` |
| `SavedListRequest` | — | Hand-written; `name`, `namePrefix`, `org` |
| `Download` | — | Open body plus content type and length |
| `model.SnapshotSummary` | generated | Used directly for list and latest |
| `model.SnapshotTreeResponse` | generated | Flat entries with `kind` and optional `size` |
| `model.SnapshotDownloadResponse` | generated | Chunk manifest |
| `model.DeletionOperation` | generated | Async deletion state |
| `model.SavedSnapshot` | generated | Saved snapshot state |
| `model.NamedSnapshotShare` | generated | Share metadata and code prefix |

`SavedListRequest.Org` is the spec's `org` query parameter: an organization id, an organization
name, or `personal`. It wins over the `X-Boat-Org` header that `core.WithOrg` sets.

`Download.ContentLength` is `-1` when the server did not send a `Content-Length` header. A single
file answers `application/octet-stream`; a folder or the whole snapshot answers `application/x-tar`.
`Download` does not buffer the body.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `List` | `GET` | `/snapshots` | `limit`, `cursor`, `sort` |
| `ListForSandbox` | `GET` | `/sandboxes/{sandboxId}/snapshots` | Same query |
| `Latest` | `GET` | `/sandboxes/{sandboxId}/snapshots/latest` | `snapshot` may be `null` |
| `Tree` | `GET` | `/snapshots/{snapshotId}/tree` | |
| `File` | `GET` | `/snapshots/{snapshotId}/files` | `path` query; omit for everything |
| `Download` | `GET` | `/snapshots/{snapshotId}/files` | Shorthand for an empty `path` |
| `DownloadManifest` | `GET` | `/snapshots/{snapshotId}/download` | JSON; signed URLs |
| `Delete` | `DELETE` | `/snapshots/{snapshotId}` | `X-Ascii-Confirm-Delete` = snapshot id |
| `DeleteForSandbox` | `DELETE` | `/sandboxes/{sandboxId}/snapshots` | `X-Ascii-Confirm-Delete` = sandbox id |
| `ListSaved` | `GET` | `/saved-snapshots` | `name`, `namePrefix`, `org` |
| `Save` | `POST` | `/saved-snapshots` | `202`; async |
| `GetSaved` | `GET` | `/saved-snapshots/{name}` | |
| `DeleteSaved` | `DELETE` | `/saved-snapshots/{name}` | |
| `CreateShare` | `POST` | `/saved-snapshots/{name}/shares` | `201`; full code shown once |
| `ListShares` | `GET` | `/saved-snapshots/{name}/shares` | |
| `RevokeShare` | `DELETE` | `/saved-snapshots/{name}/shares/{shareId}` | |

Paths are escaped with `url.PathEscape`.

## 6. Error behavior

All failures are `*core.APIError` with `Status`, `Code`, `Message`, `RequestID` and, when the API
supplies one, `RetryAfter`. Use `core.IsCode(err, code)` and `errors.As`; never match messages.

- `400 invalid_query`: an unknown query parameter; the message names it.
- `401 unauthorized`: missing or invalid API key.
- `402 credit_required`: `Save` when the wallet is empty and the plan's included saved snapshots
  are used. Add money or delete a saved snapshot.
- `403 forbidden`: the snapshot or sandbox is not owned by the caller.
- `404 not_found`: the sandbox, snapshot, saved snapshot or share does not exist.
- `409 save_in_progress`: a save under that name is still running. Wait for `GetSaved` to settle
  before retrying or deleting.
- `409 legacy_snapshot`, `snapshot_not_indexed`, `base_image_file`, `is_symlink`: `File` cannot
  return those bytes. Take a new snapshot, use `DownloadManifest`, or request the symlink target.
- `409 snapshot_compacted`: `DownloadManifest` cannot reconstruct the chain because older layers
  were cleaned up. Read files with `File` instead.
- `409` on either delete: `X-Ascii-Confirm-Delete` is missing or wrong. The SDK always sends the
  exact id, so this only happens when an interceptor alters the header.

The transport retries GETs on `429`, `500`, `502`, `503` with jitter and `Retry-After`. Deletes and
`Save` are not retried: a lost `202` may already have been accepted. Reusing a saved-snapshot name
is safe by design, but it replaces the artifact, so a retry is the caller's decision.

## 7. Streaming and long-running behavior

`File` and `Download` use `DoRaw` and return as soon as the response headers arrive. The body is
streamed, never buffered:

```go
dl, err := client.Snapshots.File(ctx, snapshotID, "var/log/app.log")
if err != nil {
    return err
}
defer dl.Body.Close()
_, err = io.Copy(w, dl.Body)
```

Rules:

- The caller owns `Download.Body` and must close it. The SDK does not close it on success.
- On a non-2xx response the SDK maps the error with `core.ErrorFromResponse`, which reads a bounded
  body and closes it; no body leaks.
- Cancelling `ctx` aborts the request and the read. The caller still closes the body.
- `core.WithHeader("Range", "bytes=0-1023")` resumes a large file; the server answers `206` with
  `Content-Range`.
- Folder and whole-snapshot responses are tar archives. The SDK does not extract them.

`Save` is asynchronous. It returns the accepted saved snapshot with status `saving`; a running
source sandbox takes a fresh capture first, which can run minutes. Poll `GetSaved` until the status
is `ready` or `failed`; a failed save carries `Error` and can be retried by saving again.
`Delete` and `DeleteForSandbox` are asynchronous too: they return `pending` operations, and the
stored data is purged in the background. `DeleteForSandbox` returns one operation per snapshot and
does not delete the sandbox or its saved snapshots.

`DownloadManifest` returns time-limited signed URLs. They are secret-bearing; never log them.

## 8. Examples

```go
saved, err := client.Snapshots.Save(ctx, sandboxID, "nightly")
if err != nil {
    return err
}

for saved.Status == model.SavedSnapshotStatusSaving {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(5 * time.Second):
    }
    saved, err = client.Snapshots.GetSaved(ctx, "nightly")
    if err != nil {
        return err
    }
}
if saved.Status == model.SavedSnapshotStatusFailed {
    return fmt.Errorf("save failed: %s", *saved.Error)
}
```

Downloading one file:

```go
dl, err := client.Snapshots.File(ctx, snapshotID, "workspace/result.json")
if err != nil {
    return err
}
defer dl.Body.Close()

if _, err := io.Copy(dst, dl.Body); err != nil {
    return err
}
```

Inline fragments are illustrative.

## 9. Testing notes

- `snapshots/list_test.go`: query shape, page cursor, empty page, null latest, error mapping.
- `snapshots/inspect_test.go`: tree entries, legacy snapshot with `tree_available: false`.
- `snapshots/download_test.go`: caller-owned body and exact close count, `Accept: */*`, `Range`
  forwarding, whole-snapshot tar, manifest chunks, `snapshot_compacted`.
- `snapshots/delete_test.go`: both confirmation headers, operation list, `409`, `403` and `404`
  mapping.
- `snapshots/saved_test.go`: save body and `202`, get, a `saving` → `ready` poll loop, delete,
  `save_in_progress`, and a table of `400`/`401`/`402`/`404`/`409` save failures.
- `snapshots/shares_test.go`: created code, list, revoke path, not-found mapping.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`. None call the live API.
- No live tests yet; a short save/get/delete flow would cost storage and belongs behind the `live`
  tag.

## 10. Open questions / known gaps

- `named-snapshots` is intentionally not exposed: the spec has it, the main API reference does not.
  It may merge with saved snapshots later.
- `DeleteForSandbox` returns `[]model.DeletionOperation` because the API accepts one operation per
  snapshot. A single-operation return would lose data.
- `Download` maps to whole-snapshot bytes under `/files`, matching its `*Download` return type; the
  spec's JSON `/download` endpoint is exposed as `DownloadManifest`.
- Response headers on `/files` (`X-Snapshot-Entry-Kind`, `X-Snapshot-File-Count`,
  `X-Snapshot-Total-Size-Bytes`, `X-Snapshot-Skipped-Base-Image-Files`) are not surfaced yet.
- `ListSaved` exposes a single page because the endpoint has no cursor, even though its return type
  is `core.Page` for symmetry with the other list endpoints.
- No `WaitSaved` helper. Callers poll `GetSaved`; a shared polling utility is not in the SDK yet.
