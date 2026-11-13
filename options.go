package boat

import (
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// APIError is the structured error returned by the Boat API.
type APIError = core.APIError

// RequestOption edits a request before it is sent.
type RequestOption = core.RequestOption

// Page is one page of a cursor-paginated list.
type Page[T any] = core.Page[T]

// Stream is a forward-only stream of frames.
type Stream[T any] = core.Stream[T]

// SandboxPage is a page of sandboxes.
type SandboxPage = core.Page[model.Sandbox]

// SnapshotPage is a page of snapshots.
type SnapshotPage = core.Page[model.SnapshotSummary]

// WithIdempotencyKey sets the Idempotency-Key header.
func WithIdempotencyKey(key string) RequestOption { return core.WithIdempotencyKey(key) }

// WithOrg scopes a request to an organization through X-Boat-Org.
func WithOrg(org string) RequestOption { return core.WithOrg(org) }

// WithHeader sets a request header.
func WithHeader(key, value string) RequestOption { return core.WithHeader(key, value) }

// WithQuery adds a query parameter.
func WithQuery(key, value string) RequestOption { return core.WithQuery(key, value) }
