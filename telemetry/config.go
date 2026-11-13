package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

const (
	maxHeaders          = 20
	maxHeaderValueChars = 8192
)

// Get returns the account's OpenTelemetry export configuration.
//
// It returns nil, nil when no export is configured. The configuration carries header names only;
// the API never returns their values. Get needs an API key with account.admin.
func (s *Service) Get(ctx context.Context, opts ...core.RequestOption) (*model.TelemetryConfig, error) {
	var out model.TelemetryResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   "/telemetry",
	}, &out, opts...); err != nil {
		return nil, err
	}
	return out.Telemetry, nil
}

// Set replaces the account's OpenTelemetry export configuration and returns the saved
// configuration with the result of the first test export.
//
// The request headers are sent to Boat, which encrypts them at rest. Their values are not returned
// by the API and are never included in the response or in error text.
//
// Set is idempotent but is not retried by default, because PUT is not a safe method. Pass
// core.WithIdempotencyKey to let the transport retry 429, 500, 502 and 503 responses. Set needs an
// API key with account.admin.
func (s *Service) Set(ctx context.Context, req model.TelemetrySetRequest, opts ...core.RequestOption) (*model.TelemetrySetResponse, error) {
	if err := validateSetRequest(req); err != nil {
		return nil, err
	}
	var out model.TelemetrySetResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPut,
		Path:   "/telemetry",
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes the account's OpenTelemetry export and its queued deliveries.
//
// Delete is never retried, even when the caller supplies an idempotency key: a retry after a lost
// success answers 404. Delete needs an API key with account.admin.
func (s *Service) Delete(ctx context.Context, opts ...core.RequestOption) error {
	return s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    "/telemetry",
		NoRetry: true,
	}, nil, opts...)
}

// validateSetRequest mirrors the spec limits before sending. Error text names a header but never
// includes its value, because header values may be credentials.
func validateSetRequest(req model.TelemetrySetRequest) error {
	if strings.TrimSpace(req.Endpoint) == "" {
		return errors.New("telemetry: endpoint is required")
	}
	if req.Headers == nil {
		return nil
	}
	if len(*req.Headers) > maxHeaders {
		return fmt.Errorf("telemetry: at most %d headers are allowed", maxHeaders)
	}

	names := make([]string, 0, len(*req.Headers))
	for name := range *req.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return errors.New("telemetry: header name is required")
		}
		if utf8.RuneCountInString((*req.Headers)[name]) > maxHeaderValueChars {
			return fmt.Errorf("telemetry: header %q value is too long", name)
		}
	}
	return nil
}
