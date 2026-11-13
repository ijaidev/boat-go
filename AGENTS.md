# AGENTS.md — Boat Go SDK

This repository is an **unofficial, idiomatic Go SDK for the Boat public API v1**.

Read this file before changing code. It is the contract for humans and agents working here.

**Precedence on conflict:** `AGENTS.md` (rules and workflow) > `docs/architecture.md` (design) >
`docs/services/*.md` (service behavior) > godoc (individual symbols). The OpenAPI spec is the source
of truth for request/response schemas.

| Task | Read first |
|---|---|
| Add a service | §4, §5, `docs/services/_template.md` |
| Transport, retries, pagination | `docs/architecture.md` §6–9 |
| Write tests | §6, `docs/testing.md` |
| Write docs | §9, `docs/README.md` |
| Commit | §2.1 |

The external API reference is <https://docs.boat.dev/api/v1>. Do not copy it here; document our
design, contracts and decisions.

---

## 1. Goals

- A clean, maintainable and readable Go SDK.
- Small hand-written files, domain-scoped packages, explicit seams.
- Go idioms, not a line-by-line port of the TypeScript SDK.
- Compile-time safety, predictable errors, context cancellation everywhere.
- Unit tests that do not require a Boat account or network access.
- Documentation that stays true as the SDK grows.

**Non-goals:** a CLI, a server, a desktop client, a replacement for the REST API, or an agent
runtime. This is a client library.

---

## 2. Non-negotiable rules

1. **Test first.** For every new function, method or service behavior where a unit test is
   feasible, write the failing test first, then implement the code. See §6.
2. **No untested behavior.** Do not add production behavior without a test. Exceptions are generated
   code, pure type aliases, and wiring covered by a compile or integration test. State the reason.
3. **`go test ./...` must pass** before a change is done.
4. **Format and vet.** Run `gofmt`/`goimports` and `go vet ./...`.
5. **Never log secrets.** API keys, bearer tokens, desktop URLs, signed file URLs and webhook
   secrets must be redacted.
6. **Never hand-edit generated code.** Generated files carry a `Code generated … DO NOT EDIT.`
   header.
7. **Keep the package graph acyclic.** Dependencies point inward only; see §4.2.
8. **Context first.** Every public I/O method takes `context.Context` as its first argument.
9. **Errors, not panics.** Library code returns errors. Panics are reserved for unrecoverable
   programmer errors in initialization.
10. **Small files.** Hand-written files should stay under ~250 lines. Split by operation cluster,
    not by type. Generated files and `_test.go` files are exempt.
11. **No god interfaces.** Accept small interfaces, return concrete structs. See §5.
12. **No new dependency without a reason.** Prefer the standard library. Record non-trivial
    dependency decisions in `docs/architecture.md`.
13. **No live API calls in normal tests.** Live tests are opt-in and tagged; see §6.
14. **Public API changes need docs.** Update the relevant `docs/services/*.md` in the same change.
15. **No empty packages "for later".** Add a package when its first real implementation lands.

### 2.1 Pre-commit checks (mandatory)

Run both checks before every commit. **Assume this repository is public and every commit is
permanent and world-readable.** A secret removed in a later commit is still in history.

#### Check 1 — No personal, private or secret information

Do not commit:

- API keys, bearer tokens, passwords, webhook secrets, SSH keys, certificates or `.env` files.
- Real sandbox IDs, account IDs, organization names, customer data, email addresses or phone numbers.
- Personal names, home-directory paths, internal hostnames, private URLs or signed URLs.
- Private repository names or links that are not already public.
- Screenshots, logs, fixtures or test output containing any of the above.
- Absolute local paths such as `/Users/<name>/...` or `/home/<name>/...`.

Use obviously fake values in tests and examples: `boat_test_...`, `example.com`, `bx_example`,
`whsec_test_...`. Never use a real key, even in a "temporary" test.

Inspect the staged diff before committing:

```bash
git status --short
git diff --cached --name-only
git diff --cached
gitleaks git --staged --redact   # pinned scanner; if unavailable, inspect manually
```

If a secret was committed accidentally, **rotate it first**, then remove it from history. Deleting
it in a new commit is not enough.

#### Check 2 — No artifacts, generated junk or out-of-scope files

Keep the repository a clean SDK source tree. Do not commit:

- Build outputs: `bin/`, `dist/`, `build/`, `*.exe`, `*.test`, `*.out`, `coverage.*`.
- Caches and dependency trees: `.cache/`, `node_modules/`, `.venv/`, `__pycache__/`, `.gocache/`.
- Editor and OS files: `.DS_Store`, `Thumbs.db`, `.idea/`, `*.swp`, local `.vscode/` settings.
- Logs, temporary files, downloaded archives, screenshots, recordings or local databases.
- Local environment files: `.env`, `.env.*`, `*.local`.
- Large binaries or datasets unrelated to the SDK.
- Any file that is not clearly part of the SDK, its tests, its docs or its pinned spec.

Allowed, when intentional and reviewed:

- `openapi/boat-v1.yaml` — the pinned OpenAPI spec.
- Generated models or clients produced by `go generate`, carrying a `DO NOT EDIT` header.
- Small fixtures under `testdata/`.
- Docs, examples and minimal repository metadata.

Create `.gitignore` if absent. It must cover at least `/bin/`, `/dist/`, `*.test`, `*.out`,
`coverage.*`, `.gocache/`, `.DS_Store`, `Thumbs.db`, `.idea/`, `.vscode/`, `.env`, `.env.*`, and
`!.env.example`.

Inspect what is staged and what is untracked:

```bash
git status --short
git diff --cached --name-only
git diff --cached --stat
git ls-files | sort
```

Do not use `git add -A` or `git add .` blindly. Add files explicitly.

#### Pre-commit checklist

- [ ] `git diff --cached` reviewed line by line.
- [ ] No personal, private or secret information.
- [ ] No build artifacts, caches or out-of-scope files.
- [ ] Tests and docs are updated for the change.

---

## 3. What this SDK is

The SDK wraps the Boat v1 API:

- Base URL: `https://boat.dev/api/v1`
- Auth: `Authorization: Bearer boat_...` (legacy `box_...` keys also work)
- Optional organization scope: `X-Boat-Org: <org>`
- Idempotency: `Idempotency-Key` on create and fork
- Envelope: `{ "ok": true, "type": "...", ... }`
- Errors: `{ "ok": false, "type": "sandbox.error", "status", "code", "message", "requestId", "error" }`
- Pagination: cursor + `pageInfo.nextCursor` / `hasMore`
- Streaming: `application/x-ndjson` for command output
- Binary: `application/octet-stream` and `application/x-tar` for files and artifacts
- Webhooks: lifecycle events with HMAC-SHA256 signatures

The OpenAPI spec is the source of truth for schemas. The official spec is
`https://docs.boat.dev/openapi/boat-v1.yaml`. Pin a copy in `openapi/` when code generation starts.

---

## 4. Architecture

### 4.1 Target layout

```text
.
├── core/                   # SDK kernel; stdlib only
├── model/                  # public generated schema; semver-covered
├── internal/
│   ├── transport/          # implements core.Doer
│   ├── ndjson/
│   └── testutil/
├── sandboxes/
├── commands/
├── files/
├── agents/
├── snapshots/
├── environments/
├── credentials/
├── account/
├── webhooks/
├── telemetry/
├── ci/
├── jobs/                   # cross-service orchestration
├── boat.go                 # facade: Config, Client, New
├── services.go
├── options.go
├── errors.go
├── pagination.go
├── docs/
└── examples/
```

This is a target, not a mandate to create every package now. Start with `core`, `model`,
`internal/transport`, `sandboxes`, `commands` and the root facade. Add domains as they are
implemented. The detailed tree is in `docs/architecture.md`.

### 4.2 Dependency rules (canonical)

Arrows point toward dependencies: `A → B` means **A imports B**.

```text
          model              core
             ▲                 ▲
             │                 │
             ├──── services ───┤
             │   (sandboxes, commands, files, agents,
             │    snapshots, environments, credentials,
             │    account, webhooks, telemetry, ci)
             │                 ▲
             │              jobs
             │                 ▲
             └────────────── boat (root facade)

internal/transport ──imports──► core
```

| Package | May import | Must not import |
|---|---|---|
| `model` | standard library only | anything else in this module |
| `core` | standard library only | services, transport, root |
| `internal/transport` | `core` | services, root |
| each service | `core`, `model` | another service, `jobs`, root |
| `jobs` | services, `core`, `model` | root |
| `boat` (root) | everything above | — |
| `webhooks/verify.go` | standard library, `model` | transport |

Services never call services. Cross-domain workflows belong in `jobs/`. The root facade only wires
and re-exports; it contains no workflow logic. `internal/transport` implements `core.Doer`.

### 4.3 Root facade

```go
package boat

type Client struct {
    Sandboxes    *sandboxes.Service
    Commands     *commands.Service
    Files        *files.Service
    Agents       *agents.Service
    Snapshots    *snapshots.Service
    Environments *environments.Service
    Credentials  *credentials.Service
    Account      *account.Service
    Webhooks     *webhooks.Service
    Telemetry    *telemetry.Service
    CI           *ci.Service
    Jobs         *jobs.Service
}

func New(cfg Config) (*Client, error)
```

Fields, not getters. Services are concrete. `New` wires one transport and injects it into each
service.

---

## 5. Interfaces and API design

### 5.1 Interfaces at seams only

```go
package core

type Doer interface {
    Do(ctx context.Context, req *Request, out any, opts ...RequestOption) error
    DoRaw(ctx context.Context, req *Request, opts ...RequestOption) (*http.Response, error)
}

type Request struct {
    Method string
    Path   string
    Query  url.Values
    Header http.Header
    JSON   any       // marshalled once, reused on retries
    Body   io.Reader // one-shot; retries require a replayable body
}

type HTTPClient interface {
    Do(*http.Request) (*http.Response, error)
}

type TokenSource interface {
    Token(context.Context) (string, error)
}

type Stream[T any] interface {
    Next(ctx context.Context) (T, error) // io.EOF when finished
    Close() error
}
```

`*http.Client` satisfies `HTTPClient`. `transport.Client` implements `core.Doer`.

### 5.2 Service pattern

```go
package sandboxes

type Service struct {
    d core.Doer
}

func New(d core.Doer) *Service {
    return &Service{d: d}
}

func (s *Service) Create(
    ctx context.Context,
    req CreateRequest,
    opts ...core.RequestOption,
) (*model.Sandbox, error) {
    var out model.CreateSandboxResponse
    if err := s.d.Do(ctx, &core.Request{
        Method: http.MethodPost,
        Path:   "/sandboxes",
        JSON:   req,
    }, &out, opts...); err != nil {
        return nil, err
    }
    return &out.Sandbox, nil
}
```

- Constructors return `*Service`, not an interface.
- Do not define `SandboxService` / `ISandboxService` god interfaces.
- Consumers may define their own narrow interfaces for mocking.
- Do not add an interface until there is a second implementation or a real seam.

### 5.3 Naming and fields

- Packages: lowercase, no underscores, domain-scoped (`sandboxes`, not `sandbox_utils`).
- Files: lowercase; `snake_case.go` is acceptable; tests end in `_test.go`.
- Initialisms: `ID`, `URL`, `HTTP`, `API`, `JSON`, `TTY`.
- Request types: `CreateRequest`, `ResumeRequest`, `ForkRequest`.
- Response types: reuse `model.*`; alias only when it improves discoverability.
- Error strings: lowercase, no trailing punctuation.
- Avoid stutter: `sandboxes.Service`, not `sandboxes.SandboxesService`.
- No package named `utils`, `common`, `helpers`, `types`, or `interfaces`.
- Use pointers for optional scalar fields unless the generated model provides a dedicated optional
  type. Document the semantics of optional slices/maps.

---

## 6. Test-driven development (mandatory)

For every new function, method or service behavior where a unit test is feasible:

1. **Write the test first** in the same package or `<package>_test`.
2. **Run it and see it fail.** A compile failure is acceptable for a brand-new API; add the minimal
   signature/stub, rerun, and confirm the failure is now an assertion failure.
3. **Write the minimal implementation** that makes the test pass.
4. **Run the package tests**, then the full suite.
5. **Refactor** with tests green.
6. **Add edge cases** before considering the work done.

If a unit test is not feasible, say why and cover the behavior with the nearest integration,
compile or example test. "Not feasible" must not mean "inconvenient".

Rules:

- Tests live next to the code: `sandboxes/create.go` → `sandboxes/create_test.go`.
- Use table-driven tests.
- Name tests `TestService_Method_Behavior`.
- Use `t.Helper()`, `t.Cleanup()` and `t.Parallel()` where safe.
- Use `httptest` + `internal/testutil`; assert method, path, query, headers and body.
- Never use a live sandbox id, IP or wall-clock timing.
- Test 400, 401, 402, 403, 404, 409, 413, 429, 500, 502 and 503 paths.
- Fake `core.Doer` or an `httptest` server; do not create a mock per service method.
- Streaming: test every frame type, `io.EOF`, cancellation, `Close()` and reconnect behavior.
- Webhooks: test the exact raw body, invalid signatures, stale timestamps and duplicate deliveries.
- Run `go test -race ./...` when touching streaming, pagination, retries or shared state.
- Live tests use `//go:build live`, require `BOAT_API_KEY`, use a short TTL and always clean up.

Full patterns: `docs/testing.md`.

---

## 7. Errors, retries and streaming

### 7.1 Errors

- Map API failures to `*core.APIError` with `Status`, `Code`, `Message`, `RequestID`, `Details`,
  `RetryAfter`.
- Preserve the request id; it is required for support.
- Use `errors.As` / `errors.Is`; never match on error strings.
- Wrap transport errors with `%w`.
- Do not expose raw response bodies unless they are safe and useful. Redact secrets and signed URLs.

### 7.2 Retries

- Retry idempotent GETs on `429`, `500`, `502`, `503` with jitter and `Retry-After`.
- Retry create/fork **only** when an `Idempotency-Key` is present.
- For create/fork, generate one UUID key per logical call, store it on the request, and reuse it
  verbatim on every attempt. Default-on; callers may override it.
- **Never auto-retry a command.** A 502 may mean the command is already running. The only safe
  command retry is the documented `sandbox_restoring` case, where the command did not run.
- Only retry when the body is replayable (`JSON` or a body factory). Never retry a one-shot
  `io.Reader`.
- Respect `RateLimit` and `RateLimit-Policy` headers.
- A refused start does not count as a start; still surface the error.

### 7.3 Streaming

- Never buffer a whole NDJSON or tar response in memory.
- Use the generic decoder in `internal/ndjson`.
- `Next` returns `io.EOF` at the end; other errors are terminal unless documented.
- Always close the response body.
- Honor context cancellation promptly.
- Do not spin on empty frames.

### 7.4 Webhooks

- Verify `HMAC_SHA256(secret, deliveryID + "." + timestamp + "." + rawBody)`.
- Compare in constant time.
- Reject timestamps outside a configurable tolerance (default 5 minutes).
- Parse JSON only after verification.
- Deduplication is caller-owned; document that the library does not keep state.

---

## 8. Code generation and `model`

- The OpenAPI spec is the source of truth for schemas.
- Pin the spec in `openapi/boat-v1.yaml`; record the source date or commit in
  `docs/architecture.md`.
- Generate models into `model/` and/or the raw client into `internal/gen/` via `go generate`.
- Generated files are never edited by hand.
- Add a CI check that regeneration produces no diff.
- Do not fetch the latest spec during CI without review; review schema changes as a pull request.

`model` is the **public, semver-covered schema package**. The rule "do not expose generated code
directly" refers to the generated raw HTTP client and internal generated code, not to `model`
types. The hand-written facade remains the public behavior surface; `model` is its public data
contract.

---

## 9. Documentation

Two layers:

1. **Godoc** (`doc.go`, exported comments) — what a developer sees in `go doc` and pkg.go.dev.
2. **Repository docs** (`/docs`) — architecture, testing strategy, service contracts and codegen.

Both are required. Godoc explains the symbol; `/docs` explains the design and behavior.

```text
docs/
├── README.md                 # index and documentation rules
├── architecture.md           # package graph, interfaces, request lifecycle, decisions
├── testing.md                # TDD, test types, fixtures, live tests
└── services/
    ├── README.md             # status table and doc contract
    ├── _template.md          # copy this for a new service
    └── <service>.md          # one file per implemented service
```

Rules:

- **One service, one file.** `docs/services/commands.md`, not a single giant services page.
- Create the service doc in the same change as the first service code.
- A service is not done until its doc exists and is linked from `docs/services/README.md`.
- Update the doc when behavior, endpoints, error handling or streaming semantics change.
- Do not duplicate the official API reference. Link to it and document our mapping and decisions.
- Use Mermaid for diagrams. Keep them small.
- Complete examples (package clause + imports) live in `examples/` and must compile. Inline
  fragments in docs are illustrative only.
- No stale "planned" language once a service ships.
- Required service-doc sections are defined in `docs/services/README.md` and `_template.md`; use the
  template and do not omit sections.

Root README: short description, status, install, a 10-line example, links to `/docs` and this file.
Do not turn it into the full documentation.

---

## 10. Definition of done

- [ ] The failing test was written first (or the exception is documented).
- [ ] New behavior has a unit test for success and error paths.
- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `gofmt -l .` is empty.
- [ ] `goimports -l .` is empty (if `goimports` is used).
- [ ] `go test ./...` passes.
- [ ] `go test -race ./...` passes when concurrency or streaming is touched.
- [ ] `go generate ./...` produces no diff.
- [ ] Public API comments are present.
- [ ] The relevant `docs/services/*.md` is updated.
- [ ] The service status table in `docs/services/README.md` is current.
- [ ] Pre-commit check 1: no personal, private or secret information.
- [ ] Pre-commit check 2: no artifacts, caches or out-of-scope files.
- [ ] No secret is logged or committed.
- [ ] The package graph remains acyclic and follows §4.2.
- [ ] `go build ./examples/...` and `go vet ./examples/...` pass, if examples changed.

---

## 11. Commands

Once `go.mod` exists:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go generate ./...
go build ./examples/...
go mod tidy
```

Optional, once configured:

```bash
staticcheck ./...
golangci-lint run
```

Live tests:

```bash
BOAT_API_KEY=boat_... go test -tags=live ./...
```

---

## 12. What not to do

- Do not add a CLI or a server to this module.
- Do not expose the generated raw HTTP client or internal generated code as the public API.
- Do not create `SandboxService`, `ICommandService` or other god interfaces.
- Do not add a `utils`, `common`, `helpers` or `types` package.
- Do not auto-retry commands or other non-idempotent operations.
- Do not log tokens, desktop URLs, signed URLs or webhook secrets.
- Do not use `panic` for API or network failures.
- Do not create empty packages in advance.
- Do not fetch the latest OpenAPI spec silently.
- Do not write a test that depends on wall-clock timing without tolerance.
- Do not use `time.Sleep` in tests when a channel, poll or fake clock will do.
- Do not leave a service undocumented because "the code is self-explanatory".

---

## 13. Where to read more

- [Architecture](docs/architecture.md)
- [Testing and TDD](docs/testing.md)
- [Service docs index](docs/services/README.md)
- [Service doc template](docs/services/_template.md)
- [Boat API reference](https://docs.boat.dev/api/v1)
- [Boat OpenAPI spec](https://docs.boat.dev/openapi/boat-v1.yaml)
