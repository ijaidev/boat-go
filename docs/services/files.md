# Files

Status: done

Owner: SDK maintainers

Package: `files`

## 1. Purpose and scope

Wraps the sandbox filesystem and sandbox access endpoints:

- inline file read and write (`utf8` and `base64`)
- delete, list, info, search, find, folder creation, move, permissions and text replacement
- signed upload and download URLs for files over the inline cap
- binary artifact downloads
- hosted ports, the desktop stream URL and SSH key setup

Paths are relative to the sandbox work directory `/home/user` unless they start with `/`. An
absolute path must resolve under `/home/user` or `/tmp`; anything else returns 400 `invalid_path`.
Missing folders are created by writes, folders and moves; the service does not add recursive flags
the API does not have.

## 2. Non-goals

- Command execution. That lives in `commands/`.
- Repository push/pull, archive extraction and other multi-step flows. Those belong in `jobs/`.
- Fetching signed upload or download URLs and desktop streams. The service returns the URL; the
  caller sends the PUT/GET or embeds the stream with its own client.
- Directory or tarball artifact downloads. v1 returns one file per artifact request.
- Cursor pagination. List, search and find have fixed caps and a `truncated` flag.
- SSH sessions or tunnels. `SSHKey` only authorizes a public key.

## 3. Public API

| Method | Signature | Notes |
|---|---|---|
| `Read` | `Read(ctx, sandboxID, path string, encoding Encoding, ...core.RequestOption) (*ReadResult, error)` | 5 MiB inline cap |
| `Write` | `Write(ctx, sandboxID string, req WriteRequest, ...core.RequestOption) (*model.FileWriteResponse, error)` | 5 MiB after decoding |
| `Delete` | `Delete(ctx, sandboxID, path string, options DeleteOptions, ...core.RequestOption) (*model.FileDeleteResponse, error)` | `Recursive` for directories |
| `List` | `List(ctx, sandboxID string, req ListRequest, ...core.RequestOption) (*model.FileListResponse, error)` | One level, sorted by name |
| `Info` | `Info(ctx, sandboxID, path string, ...core.RequestOption) (*model.FileInfoResponse, error)` | Size, mode, type, mtime |
| `Search` | `Search(ctx, sandboxID string, req SearchRequest, ...core.RequestOption) (*model.FileSearchResponse, error)` | Glob on names |
| `Find` | `Find(ctx, sandboxID string, req FindRequest, ...core.RequestOption) (*model.FileFindResponse, error)` | Text or regex lines |
| `Mkdir` | `Mkdir(ctx, sandboxID string, req MkdirRequest, ...core.RequestOption) (*model.FileInfoResponse, error)` | Creates missing parents |
| `Move` | `Move(ctx, sandboxID string, req MoveRequest, ...core.RequestOption) (*model.FileMoveResponse, error)` | Rename or move |
| `Chmod` | `Chmod(ctx, sandboxID string, req ChmodRequest, ...core.RequestOption) (*model.FileInfoResponse, error)` | Octal mode |
| `Replace` | `Replace(ctx, sandboxID string, req ReplaceRequest, ...core.RequestOption) (*model.FileReplaceResponse, error)` | Per-file results |
| `UploadURL` | `UploadURL(ctx, sandboxID string, req URLRequest, ...core.RequestOption) (*model.FileURLResponse, error)` | Signed secret URL |
| `DownloadURL` | `DownloadURL(ctx, sandboxID string, req URLRequest, ...core.RequestOption) (*model.FileURLResponse, error)` | Signed secret URL |
| `Artifact` | `Artifact(ctx, sandboxID, path string, ...core.RequestOption) (*Download, error)` | Raw binary stream |
| `Host` | `Host(ctx, sandboxID string, req HostRequest, ...core.RequestOption) (*model.HostPortResponse, error)` | Token-gated URL unless public |
| `Desktop` | `Desktop(ctx, sandboxID string, req DesktopRequest, ...core.RequestOption) (*model.DesktopResponse, error)` | Secret-bearing desktop URL |
| `SSHKey` | `SSHKey(ctx, sandboxID string, req SSHKeyRequest, ...core.RequestOption) (*model.SSHKeyResponse, error)` | Authorizes a public key |

Every method takes `context.Context` first and accepts per-call `core.RequestOption`s such as
`core.WithOrg`. Services are safe for concurrent use; no mutable package state exists.

## 4. Types and aliases

| Public type | Underlying model | Notes |
|---|---|---|
| `Encoding` | — | `EncodingUTF8` and `EncodingBase64`; `""` means UTF-8 |
| `ReadResult` | embeds `model.FileReadResponse` | `Bytes()` decodes base64 |
| `WriteRequest` | — | Hand-written body; `Encoding` is optional |
| `DeleteOptions` | — | Query option `Recursive` |
| `ListRequest` | — | `Path` is optional and defaults to `/home/user` |
| `SearchRequest` | — | `Path`, `Pattern` |
| `FindRequest` | — | `Path`, `Pattern`, `Regex` |
| `MkdirRequest` | — | Hand-written body; optional `Mode` |
| `MoveRequest` | — | Hand-written body; optional `Overwrite` |
| `ChmodRequest` | — | Hand-written body |
| `ReplaceRequest` | — | Hand-written body |
| `URLRequest` | — | Hand-written body; `TTLSeconds *int` |
| `Download` | — | `Body io.ReadCloser`, `ContentType`, `ContentLength` |
| `HostRequest` | — | Hand-written body; `Port` is required |
| `DesktopTheme` | — | `DesktopThemeLight`, `DesktopThemeDark` |
| `DesktopRequest` | — | `VNC`, `Theme`, `PublicAccess` |
| `SSHKeyRequest` | — | Hand-written body |

`ReadResult.Content` is the wire form. Use `Bytes()` to decode it; base64 decoding errors are
wrapped, never silently ignored.

## 5. Endpoints and mapping

| SDK method | HTTP | Path | Notes |
|---|---|---|---|
| `Read` | `GET` | `/sandboxes/{sandboxId}/files` | `path`, `encoding` |
| `Write` | `PUT` | `/sandboxes/{sandboxId}/files` | |
| `Delete` | `DELETE` | `/sandboxes/{sandboxId}/files` | `path`, `recursive` |
| `List` | `GET` | `/sandboxes/{sandboxId}/files/list` | `path` |
| `Info` | `GET` | `/sandboxes/{sandboxId}/files/info` | `path` |
| `Search` | `GET` | `/sandboxes/{sandboxId}/files/search` | `path`, `pattern` |
| `Find` | `GET` | `/sandboxes/{sandboxId}/files/find` | `path`, `pattern`, `regex` |
| `Mkdir` | `POST` | `/sandboxes/{sandboxId}/files/folder` | |
| `Move` | `POST` | `/sandboxes/{sandboxId}/files/move` | `overwrite` in the body |
| `Chmod` | `POST` | `/sandboxes/{sandboxId}/files/permissions` | |
| `Replace` | `POST` | `/sandboxes/{sandboxId}/files/replace` | |
| `DownloadURL` | `POST` | `/sandboxes/{sandboxId}/files/download-url` | `path`, `ttlSeconds` |
| `UploadURL` | `POST` | `/sandboxes/{sandboxId}/files/upload-url` | `path`, `ttlSeconds` |
| `Artifact` | `GET` | `/sandboxes/{sandboxId}/artifacts` | `path`; `Accept: application/octet-stream` |
| `Host` | `POST` | `/sandboxes/{sandboxId}/host` | |
| `Desktop` | `POST` | `/sandboxes/{sandboxId}/desktop` | `vnc=1`, `theme` |
| `SSHKey` | `POST` | `/sandboxes/{sandboxId}/sshkey` | |

`List`, `Search` and `Find` omit `path` when the request path is empty, so the server default
`/home/user` applies. `Read` and `Write` always send an explicit encoding; an empty `Encoding`
becomes `utf8`.

## 6. Error behavior

- `400 invalid_path`: the path escapes `/home/user` and `/tmp`, or is otherwise invalid. Fix the
  path; retrying cannot help.
- `400` invalid permission bits, a bad glob or an invalid regular expression (`Find` with
  `Regex: true`).
- `401 unauthorized`: invalid or missing key.
- `402 billing_required` / plan errors on `Host`, `Desktop` and `SSHKey`.
- `403`: the key lacks the `file.read` or `file.write` scope.
- `404 not_found`: the sandbox or file does not exist, including signed URL and desktop requests.
- `409 destination_exists` on `Move` without `Overwrite`. Use `overwrite: true` or pick another
  destination.
- `413 file_too_large`: the inline read or write cap (5 MiB) was exceeded.
  `error.details.maxBytes` and `error.details.sizeBytes` carry the numbers. This error is terminal
  and is never retried; use `DownloadURL` or `UploadURL`.
- `429 rate_limited`: retried by the transport for GETs, respecting `Retry-After`.
- `500`, `502`, `503`: retried for GETs. Mutations (`PUT`, `POST`, `DELETE`) are never retried
  automatically, because the SDK sends no idempotency key for them.

`Read`, `Info`, `List`, `Search`, `Find` and `Artifact` are GETs and therefore retried by the
transport on `429`, `500`, `502` and `503`. Writes, deletes, moves and other mutations surface the
first error. A `413` is never retried.

Every API error is a `*core.APIError` with `Status`, `Code`, `Message`, `RequestID`,
`Details` and `RetryAfter`. Use `errors.As`, `core.IsCode` and `core.RetryAfter`; error text never
contains signed URLs, desktop URLs or request bodies.

## 7. Streaming and long-running behavior

- `Artifact` returns the response body unread in `Download.Body`. Nothing is buffered by the SDK;
  read it incrementally and close it. `ContentLength` is `-1` when the server omits it. A non-2xx
  response is mapped with `core.ErrorFromResponse`, which closes the body.
- A read or artifact download stops when the context is cancelled; the underlying `http.Client`
  aborts the request.
- `Desktop` with `VNC` may answer `provisioning: true`. Poll `Desktop` again until `DesktopURL` is
  present; the SDK does not poll for you.
- `Host` is idempotent for a sandbox and port: repeating it returns the same URL and token.
- `UploadURL` and `DownloadURL` return an expiry; the caller decides how to send the signed request
  and must not retain or log the URL past its useful life.

## 8. Examples

```go
res, err := svc.Read(ctx, sandboxID, "notes.txt", files.EncodingUTF8)
if err != nil {
    return err
}
content, err := res.Bytes()
if err != nil {
    return err
}

_, err = svc.Write(ctx, sandboxID, files.WriteRequest{
    Path:    "src/main.go",
    Content: "package main\n",
})

dl, err := svc.Artifact(ctx, sandboxID, "dist/app.tar")
if err != nil {
    return err
}
defer dl.Body.Close()
_, err = io.Copy(w, dl.Body)
```

For a large file, call `DownloadURL` or `UploadURL` and send the GET/PUT with your own HTTP client.
Treat the returned URL as a credential.

## 9. Testing notes

- Tests live next to the code: `files/read_test.go`, `files/write_test.go`, `files/manage_test.go`,
  `files/edit_test.go`, `files/urls_test.go`, `files/artifact_test.go` and `files/access_test.go`.
- All tests use `httptest` through `internal/testutil` or a fake `core.Doer`; none call the live API.
- Request shape is asserted (method, path, query names, headers, JSON body), including base64
  encoding, `recursive`, `regex`, `overwrite`, `vnc`, `theme` and `ttlSeconds`.
- Error coverage maps 400, 401, 402, 403, 404, 409, 413, 429, 500, 502 and 503 to
  `*core.APIError`, checks the request id, and asserts that a `413` produces exactly one request.
- `Artifact` tests assert the body is not read before the caller reads it and is closed on both the
  success and error paths.
- No live-test coverage; the service only needs a sandbox and a key, but tests must stay offline.

## 10. Open questions / known gaps

- The SDK does not PUT or GET signed URLs for the caller. That keeps the HTTP client under caller
  control, but a helper may belong in `jobs/` later.
- `Artifact` supports one file per call, up to 50 MiB; v1 has no directory or tar artifact endpoint.
- `List`, `Search` and `Find` are capped by the API (10,000 entries and 1,000 matches) and expose
  `truncated`; pagination may need a server-side change.
- `ReadResult.Bytes` decodes base64 and returns every other encoding as raw bytes; a future API
  encoding would need a new case.
