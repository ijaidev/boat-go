package transport

import (
	"context"
	"io"
	"math/rand/v2"
	"time"
)

const maxRetryDelay = 30 * time.Second

// retryDelay returns the wait before retry attempt (0-based). A server-provided delay wins.
func (c *Client) retryDelay(attempt int, server time.Duration) time.Duration {
	if server > 0 {
		if server > maxRetryDelay {
			return maxRetryDelay + 1
		}
		return server
	}

	delay := c.retryMin
	for range attempt {
		if delay >= c.retryMax/2 {
			delay = c.retryMax
			break
		}
		delay *= 2
	}
	if delay > c.retryMax {
		delay = c.retryMax
	}
	if delay <= 0 {
		return 0
	}
	// Jitter in [delay/2, delay].
	half := delay / 2
	return half + time.Duration(rand.Int64N(int64(half)+1))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 32<<10))
	_ = body.Close()
}
