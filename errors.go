package boat

import (
	"time"

	"github.com/ijaidev/boat-go/core"
)

// IsCode reports whether err is an *APIError with the given code.
func IsCode(err error, code string) bool { return core.IsCode(err, code) }

// RetryAfter returns the server-provided retry delay carried by err, or zero.
func RetryAfter(err error) time.Duration { return core.RetryAfter(err) }

// Retryable reports whether the API marked the failed machine operation as safe to retry.
func Retryable(err error) bool { return core.Retryable(err) }
