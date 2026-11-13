package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// requestError keeps the underlying error for errors.Is/errors.As while hiding its text, because
// net/http errors can contain a signed URL.
type requestError struct {
	operation string
	err       error
}

func (e *requestError) Error() string {
	return "jobs: " + e.operation + ": request failed"
}

func (e *requestError) Unwrap() error {
	return e.err
}

func requestFailed(operation string, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("jobs: %s: %w", operation, context.Canceled)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("jobs: %s: %w", operation, context.DeadlineExceeded)
	default:
		return &requestError{operation: operation, err: err}
	}
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	_ = resp.Body.Close()
}
