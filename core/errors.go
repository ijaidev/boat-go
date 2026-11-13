package core

import (
	"errors"
	"fmt"
	"time"
)

// APIError is a structured error returned by the Boat API.
//
// Status and Code are the primary machine-readable signals. RequestID is preserved because Boat
// support needs it. RetryAfter is populated from Retry-After, retryAfterSeconds or retryAfter when
// the API supplies one.
type APIError struct {
	Status     int
	Code       string
	Message    string
	RequestID  string
	Details    map[string]any
	RetryAfter time.Duration
}

// Error implements error. It never includes request URLs, headers or response bodies.
func (e *APIError) Error() string {
	if e == nil {
		return "boat: api error"
	}
	msg := "boat: "
	if e.Status != 0 {
		msg += fmt.Sprintf("http %d", e.Status)
		if e.Code != "" {
			msg += " " + e.Code
		}
	} else if e.Code != "" {
		msg += e.Code
	}
	if e.Message != "" {
		if msg != "boat: " {
			msg += ": "
		}
		msg += e.Message
	}
	if e.RequestID != "" {
		msg += " (request " + e.RequestID + ")"
	}
	if msg == "boat: " {
		msg += "api error"
	}
	return msg
}

// IsCode reports whether err is an *APIError with the given code.
func IsCode(err error, code string) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == code
}

// RetryAfter returns the server-provided retry delay carried by err, or zero.
func RetryAfter(err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

// Retryable reports whether the API error carries a server hint that the failed machine operation
// provably did not run and may be retried. It does not decide the retry policy.
func Retryable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	v, ok := apiErr.Details["retryable"].(bool)
	return ok && v
}
