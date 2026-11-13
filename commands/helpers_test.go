package commands_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

// closeCounter wraps a reader and counts Close calls.
type closeCounter struct {
	r      io.Reader
	mu     sync.Mutex
	closes int
}

func newCloseCounter(r io.Reader) *closeCounter {
	return &closeCounter{r: r}
}

func (c *closeCounter) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *closeCounter) Close() error {
	c.mu.Lock()
	c.closes++
	c.mu.Unlock()
	if closer, ok := c.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *closeCounter) CloseCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closes
}

// signalingReader closes started on the first Read.
type signalingReader struct {
	r       io.Reader
	once    sync.Once
	started chan struct{}
}

func newSignalingReader(r io.Reader) *signalingReader {
	return &signalingReader{r: r, started: make(chan struct{})}
}

func (r *signalingReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.r.Read(p)
}

func (r *signalingReader) Close() error {
	if closer, ok := r.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// streamDoer returns a fake Doer whose raw responses carry body.
func streamDoer(status int, body io.Reader) (*testutil.Doer, *closeCounter) {
	counter := newCloseCounter(body)
	d := &testutil.Doer{
		RawFunc: func(context.Context, *core.Request, ...core.RequestOption) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       counter,
			}, nil
		},
	}
	return d, counter
}

func readJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := decodeJSON(r.Body, &body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func decodeJSON(r io.Reader, out any) error {
	return json.NewDecoder(r).Decode(out)
}

func requireQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
