# AGENTS.md — Boat Go SDK

Unofficial, idiomatic Go SDK for the Boat public API v1. Module `github.com/ijaidev/boat-go`,
Go 1.26. This file is the working contract for humans and coding agents.

**Precedence:** `AGENTS.md` (workflow, boundaries) > `docs/architecture.md` (design) >
`docs/services/*.md` (service behavior) > godoc. The pinned OpenAPI spec is the source of truth for
request and response schemas.

## Start here

| Task | Read first |
|---|---|
| Understand the design | [docs/architecture.md](docs/architecture.md) |
| Add or change a service | [architecture §5](docs/architecture.md#5-core-kernel), [service template](docs/services/_template.md), a nearby service |
| Write tests | [docs/testing.md](docs/testing.md) |
| Write docs | [docs/README.md](docs/README.md), [service docs index](docs/services/README.md) |
| Commit or release | [pre-commit checks](#pre-commit-checks-mandatory) |
| Boat API reference | <https://docs.boat.dev/api/v1> (link, never copy) |

**What lives where.** Do not duplicate these sources; link them.

| File | Owns |
|---|---|
| `AGENTS.md` | workflow, hard boundaries, commands, pre-commit checks |
| `docs/architecture.md` | package graph, interfaces, request lifecycle, errors, retries, streaming, codegen |
| `docs/testing.md` | test types, patterns, fixtures, fakes, live tests |
| `docs/services/*.md` | one service's behavior, endpoints, errors and gaps |

## Non-negotiable rules

1. **Test first.** Write the failing test first, then the minimal implementation. If a unit test is
   genuinely impossible, say why and cover it with an integration, compile or example test.
2. **No untested behavior.** Generated code, pure type aliases and wiring covered by a compile or
   integration test are the only exceptions; state the reason.
3. **Format, vet, test.** `gofmt -l .` must be empty, `go vet ./...` must pass, and `go test ./...`
   must pass before a change is done.
4. **Never commit secrets or personal data.** API keys, tokens, desktop URLs, signed URLs, webhook
   secrets, real account or sandbox IDs, emails and local home paths stay out of the repository.
5. **Never hand-edit generated code.** `model/` comes from the pinned spec; run `go generate ./...`.
6. **Keep the package graph acyclic.** Services never import services; cross-domain work lives in
   `jobs/`. The canonical table is in [architecture §4](docs/architecture.md#4-package-graph).
7. **Context first, errors not panics.** Every public I/O method takes `context.Context` first.
   Library code returns errors; panic only for unrecoverable initialization errors.
8. **Concrete services, small interfaces.** Return `*Service`; define interfaces only at real seams.
   No `SandboxService`, `ICommandService` or similar god interfaces.
9. **Small hand-written files.** Stay under about 250 lines; split by operation cluster. Generated
   files and `_test.go` files are exempt.
10. **No new dependency without a reason.** Prefer the standard library; record non-trivial choices in
    `docs/architecture.md`.
11. **No live API calls in normal tests.** Live tests use `//go:build live`, require `BOAT_API_KEY`,
    use a short TTL and always clean up.
12. **Public API changes need docs.** Update the service doc and its status row in the same change.

## Workflow

1. Read the routing table and the canonical doc for the area you will touch.
2. Write the failing test.
3. Implement the minimum that makes it pass; then add edge cases.
4. Run the package tests, then the full suite.
5. Update the service doc and `docs/services/README.md` status.
6. Run the pre-commit checks below.
7. Commit with a conventional message: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`.

## Commands

```bash
go build ./...
gofmt -w .
go vet ./...
go test ./...
go test ./... -run <TestName>   # focused, while iterating
go test -race ./...             # streaming, pagination, retries or shared state
go generate ./...               # then check for a dirty tree
go build ./examples/...
go vet ./examples/...
go mod tidy
```

Live tests (opt-in, costs money):

```bash
BOAT_API_KEY=boat_... go test -tags=live ./...
```

## Pre-commit checks (mandatory)

Assume this repository is public and every commit is permanent. A secret removed later is still in
history.

- No API keys, tokens, passwords, webhook secrets, SSH keys, certificates or `.env` files.
- No real sandbox IDs, account IDs, organization names, customer data, emails or phone numbers.
- No personal names, home-directory paths, internal hostnames, private URLs or signed URLs.
- No screenshots, logs, fixtures or test output containing any of the above.
- Use fake values: `boat_test_...`, `example.com`, `bx_example`, `whsec_test_...`.
- No build outputs, caches, editor files, logs, local databases or unrelated binaries. `.gitignore`
  must cover at least `/bin/`, `/dist/`, `*.test`, `*.out`, `coverage.*`, `.gocache/`, `.DS_Store`,
  `Thumbs.db`, `.idea/`, `.vscode/`, `.env`, `.env.*`, and `!.env.example`.
- Add files explicitly; never `git add -A` or `git add .` blindly.

Inspect the staged change before committing:

```bash
git status --short
git diff --cached --name-only
git diff --cached
gitleaks git --staged --redact   # if available; otherwise inspect manually
```

If a secret was committed, **rotate it first**, then remove it from history. Deleting it in a new
commit is not enough.

## Definition of done

- [ ] The failing test came first, or the exception is documented.
- [ ] New behavior has success and error tests.
- [ ] `go build ./...` passes; `gofmt -l .` is empty; `go vet ./...` and `go test ./...` pass.
- [ ] `go test -race ./...` passes when concurrency or streaming is touched.
- [ ] `go generate ./...` produces no diff.
- [ ] Public API comments are present; the service doc and status table are current.
- [ ] Examples build and vet if they changed.
- [ ] No secret, artifact or out-of-scope file is staged.
- [ ] The package graph still follows [architecture §4](docs/architecture.md#4-package-graph).

## Boundaries

- No CLI, server, desktop client or agent runtime in this module.
- Do not expose the generated raw HTTP client or internal generated code. `model` is public.
- Do not add empty packages, or `utils`, `common`, `helpers`, `types` or `interfaces` packages.
- Do not auto-retry commands or other non-idempotent operations.
- Do not log tokens, desktop URLs, signed URLs or webhook secrets.
- Do not use `panic` for API or network failures.
- Do not fetch the latest OpenAPI spec silently; pin and review it.
- Do not write tests that depend on wall-clock timing without tolerance.
- Do not use `time.Sleep` when a channel, poll or fake clock works.
- Do not leave a service undocumented because "the code is self-explanatory".
