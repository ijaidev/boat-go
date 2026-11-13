package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ijaidev/boat-go/core"
)

// Do executes req and decodes the success envelope payload into out.
func (c *Client) Do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	resp, err := c.DoRaw(ctx, req, opts...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONBody+1))
	if err != nil {
		return fmt.Errorf("boat: read response: %w", err)
	}
	if len(body) > maxJSONBody {
		return errors.New("boat: response body is too large")
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return core.ParseAPIError(resp.StatusCode, resp.Header, body)
	}

	var env struct {
		OK *bool `json:"ok"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &env); err != nil {
			return fmt.Errorf("boat: decode response envelope: %w", err)
		}
	}
	if env.OK != nil && !*env.OK {
		return core.ParseAPIError(resp.StatusCode, resp.Header, body)
	}
	if out == nil {
		return nil
	}
	if len(body) == 0 {
		return errors.New("boat: empty JSON response")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("boat: decode response payload: %w", err)
	}
	return nil
}

// DoRaw executes req and returns the response. The caller owns and must close the body.
//
// Non-2xx responses are returned as-is; use core.APIError mapping via the service helpers or
// call Do for envelope requests.
func (c *Client) DoRaw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("boat: nil request")
	}
	if req.Method == "" {
		return nil, errors.New("boat: request method is empty")
	}
	if req.Path == "" {
		return nil, errors.New("boat: request path is empty")
	}
	if req.JSON != nil && req.Body != nil {
		return nil, errors.New("boat: request cannot set JSON and Body")
	}
	core.Apply(req, opts...)

	var jsonBody []byte
	if req.JSON != nil {
		var err error
		jsonBody, err = json.Marshal(req.JSON)
		if err != nil {
			return nil, fmt.Errorf("boat: encode request: %w", err)
		}
	}

	replayable := req.Body == nil
	retryable := replayable && !req.NoRetry && (isSafeMethod(req.Method) || req.Header.Get("Idempotency-Key") != "")
	attempts := c.maxRetries + 1

	for attempt := range attempts {
		resp, err := c.execute(ctx, req, jsonBody)
		if err != nil {
			if retryable && attempt+1 < attempts {
				if err := sleepContext(ctx, c.retryDelay(attempt, 0)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, fmt.Errorf("boat: %s %s: %w", req.Method, req.Path, err)
		}

		if retryable && attempt+1 < attempts && shouldRetryStatus(resp.StatusCode) {
			delay := c.retryDelay(attempt, core.RetryAfterFromHeader(resp.Header))
			if delay > maxRetryDelay {
				return resp, nil
			}
			drainAndClose(resp.Body)
			if err := sleepContext(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}
		return resp, nil
	}
	return nil, errors.New("boat: retry loop exhausted")
}

func (c *Client) execute(ctx context.Context, req *core.Request, jsonBody []byte) (*http.Response, error) {
	var body io.Reader
	if req.Body != nil {
		body = req.Body
	} else if req.JSON != nil {
		body = bytes.NewReader(jsonBody)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, c.requestURL(req), body)
	if err != nil {
		return nil, err
	}

	header := make(http.Header, len(req.Header)+4)
	for key, values := range req.Header {
		header[key] = append([]string(nil), values...)
	}

	if header.Get("Authorization") == "" {
		token, err := c.token(ctx)
		if err != nil {
			return nil, err
		}
		if token != "" {
			header.Set("Authorization", "Bearer "+token)
		}
	}
	if header.Get("User-Agent") == "" {
		header.Set("User-Agent", c.userAgent)
	}
	if header.Get("Accept") == "" {
		header.Set("Accept", "application/json")
	}
	if req.JSON != nil && header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}

	if values, ok := header["X-Boat-Org"]; ok {
		if len(values) == 0 || values[0] == "" {
			header.Del("X-Boat-Org")
		}
	} else if c.org != "" {
		header.Set("X-Boat-Org", c.org)
	}

	httpReq.Header = header
	return c.httpClient.Do(httpReq)
}
