package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ijaidev/boat-go/core"
)

type envelope struct {
	OK                *bool           `json:"ok"`
	Type              string          `json:"type"`
	Status            json.RawMessage `json:"status"`
	Code              string          `json:"code"`
	Message           string          `json:"message"`
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

func parseAPIError(resp *http.Response, body []byte) *core.APIError {
	status := resp.StatusCode
	apiErr := &core.APIError{
		Status:    status,
		RequestID: resp.Header.Get("X-Request-Id"),
	}

	var env envelope
	if len(body) > 0 && json.Unmarshal(body, &env) == nil && (env.Code != "" || env.Error != nil || env.Message != "") {
		apiErr.Code = env.Code
		apiErr.Message = env.Message
		apiErr.RequestID = env.RequestID
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
		apiErr.RetryAfter = parseRetryAfter(resp.Header)
	}
	return apiErr
}

func parseRetryAfter(header http.Header) time.Duration {
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

func shouldRetryStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}
