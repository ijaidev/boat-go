package core

import "net/http"

// RequestOption edits a request just before it is sent. Options are applied in order and may
// override values set by a service, including the default Idempotency-Key of create and fork.
type RequestOption func(*Request)

// Apply applies opts to req in order.
func Apply(req *Request, opts ...RequestOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}
}

// WithIdempotencyKey sets the Idempotency-Key header. Create and fork set a generated key by
// default; this option overrides it.
func WithIdempotencyKey(key string) RequestOption {
	return WithHeader("Idempotency-Key", key)
}

// WithOrg scopes the request to an organization through X-Boat-Org. An empty value clears the
// client-level organization for this request.
func WithOrg(org string) RequestOption {
	return WithHeader("X-Boat-Org", org)
}

// WithHeader sets a request header.
func WithHeader(key, value string) RequestOption {
	return func(req *Request) {
		if req.Header == nil {
			req.Header = make(http.Header)
		}
		req.Header.Set(key, value)
	}
}

// WithQuery adds a query parameter without replacing existing values.
func WithQuery(key, value string) RequestOption {
	return func(req *Request) {
		req.AddQuery(key, value)
	}
}
