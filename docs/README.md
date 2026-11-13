# Documentation

This folder is the source of truth for the SDK's **design and service contracts**. `AGENTS.md` is
the source of truth for **rules and workflow**. Godoc in the
code explains individual symbols; this folder explains how the pieces fit together and why.

Start with [AGENTS.md](../AGENTS.md) for the repository rules.

## Index

| Document | What it covers |
|---|---|
| [Architecture](architecture.md) | Package graph, core interfaces, request lifecycle, streaming, codegen, key decisions |
| [Testing and TDD](testing.md) | The failing-test-first workflow, test types, fixtures, fakes, live tests |
| [Services](services/README.md) | Status of each service and the per-service documentation contract |
| [Service template](services/_template.md) | Copy this when adding a service |

## Documentation rules

1. **One service, one file.** Put service contracts in `docs/services/<service>.md`.
2. **Create the doc with the code.** The first implementation change for a service also adds its
   doc and updates the status table.
3. **A service is not done until it is documented.** "Done" includes the doc, tests and examples.
4. **Do not duplicate the official API reference.** Link to
   <https://docs.boat.dev/> and document our mapping, decisions and gaps.
5. **Keep examples compiling.** If a doc example is more than a fragment, put it under `examples/`
   and link it.
6. **Update docs in the same change as behavior.** A behavior change with stale docs is an
   incomplete change.
7. **No speculative service docs.** Planned services get a row in the status table, not an empty
   page.

## Godoc vs repository docs

| Layer | Location | Audience | Content |
|---|---|---|---|
| Godoc | `doc.go`, exported comments | Go developers in an editor / pkg.go.dev | What a symbol does, parameters, errors, concurrency safety |
| Repository docs | `docs/` | Maintainers and integrators | Architecture, service contracts, decisions, workflows, examples |

Both are required. Neither replaces the other.

## Diagrams

Use Mermaid fenced blocks. Keep diagrams small and focused on one idea. Do not embed images that
cannot be updated in a pull request.
