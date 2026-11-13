// Package telemetry wraps the Boat OpenTelemetry export configuration.
//
// Boat sends account and sandbox metrics to <endpoint>/v1/metrics every 60 seconds and sandbox
// lifecycle events to <endpoint>/v1/logs. It does not send traces, and it never sends agent token
// data. This package manages the export destination; the SDK itself emits no telemetry.
//
// Export headers may carry vendor credentials such as an API key. The API returns header names
// only, never values, and this package never logs header values or includes them in error text.
package telemetry

import (
	"context"
	"errors"

	"github.com/ijaidev/boat-go/core"
)

// Service is the Boat telemetry API.
type Service struct {
	d core.Doer
}

// New returns a telemetry service backed by d.
func New(d core.Doer) *Service {
	return &Service{d: d}
}

func (s *Service) do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	if s == nil || s.d == nil {
		return errors.New("telemetry: service has no transport")
	}
	return s.d.Do(ctx, req, out, opts...)
}
