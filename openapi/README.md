# Pinned OpenAPI spec

`boat-v1.yaml` is a byte-for-byte copy of the public Boat API v1 spec.

- Source: <https://docs.boat.dev/openapi/boat-v1.yaml>
- Fetched: 2026-11-13
- Upstream last modified: 2026-11-11
- SHA-256: see `sha256sum boat-v1.yaml`

The spec is the source of truth for request and response schemas. Update it only in a reviewed pull
request, then regenerate `model/` with `go generate ./...` and commit the result.

The upstream spec contains public placeholder examples. One is a Stripe-style `whsec_...` value in
the webhook response example; it is not a credential. `.github/secret_scanning.yml` excludes this
third-party artifact from secret scanning so it does not create recurring false positives.
