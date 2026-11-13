package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

const maxErrorBody = 64 << 10

// ParseAPIError maps a response status, headers and JSON body to *APIError.
//
// It never includes the raw body in the message. A non-JSON body produces an http_<status> code.
func ParseAPIError(status int, header http.Header, body []byte) *APIError {
	apiErr := &APIError{Status: status}
	if header != nil {
		apiErr.RequestID = header.Get("X-Request-Id")
	}

	var env errorEnvelope
	if len(body) > 0 && json.Unmarshal(body, &env) == nil && (env.Code != "" || env.Error != nil || env.Message != "") {
		apiErr.Code = env.Code
		apiErr.Message = env.Message
		if env.RequestID != "" {
			apiErr.RequestID = env.RequestID
		}
		var numericStatus int
		if json.Unmarshal(env.Status, &numericStatus) == nil && numericStatus != 0 {
			apiErr.Status = numericStatus
		}
		if env.Error != nil {
			if apiErr.Code == "" {
				apiErr.Code = env.Error.Code
			}
			if apiErr.Message == "" {
				apiErr.Message = env.Error.Message
			}
			if env.Error.Details != nil {
				apiErr.Details = env.Error.Details
			}
			if env.Error.Retryable != nil {
				if apiErr.Details == nil {
					apiErr.Details = make(map[string]any)
				}
				apiErr.Details["retryable"] = *env.Error.Retryable
			}
		}
		if apiErr.Details == nil {
			apiErr.Details = env.Details
		}
		if env.RetryAfterSeconds != nil {
			apiErr.RetryAfter = time.Duration(*env.RetryAfterSeconds) * time.Second
		} else if env.RetryAfter != nil {
			apiErr.RetryAfter = time.Duration(*env.RetryAfter) * time.Second
		}
	} else if apiErr.Code == "" {
		apiErr.Code = fmt.Sprintf("http_%d", status)
	}

	if apiErr.RetryAfter == 0 {
		apiErr.RetryAfter = RetryAfterFromHeader(header)
	}
	return apiErr
}

// ErrorFromResponse reads a bounded error body, closes it, and maps it to *APIError.
//
// Use it after DoRaw returns a non-2xx response. The caller must not read or close the body again.
func ErrorFromResponse(resp *http.Response) error {
	if resp == nil {
		return &APIError{Message: "nil response"}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody+1))
	_ = resp.Body.Close()
	return ParseAPIError(resp.StatusCode, resp.Header, body)
}

// RetryAfterFromHeader parses a Retry-After header in seconds or HTTP-date form.
func RetryAfterFromHeader(header http.Header) time.Duration {
	if header == nil {
		return 0
	}
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if d := time.Until(when); d > 0 {
			return d
		}
	}
	return 0
}

type errorEnvelope struct {
	Code              string          `json:"code"`
	Message           string          `json:"message"`
	Status            json.RawMessage `json:"status"`
	RequestID         string          `json:"requestId"`
	Details           map[string]any  `json:"details"`
	Error             *errorDetail    `json:"error"`
	RetryAfterSeconds *int            `json:"retryAfterSeconds"`
	RetryAfter        *int            `json:"retryAfter"`
}

type errorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Status    int            `json:"status"`
	Details   map[string]any `json:"details"`
	Retryable *bool          `json:"retryable"`
}
