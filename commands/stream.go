package commands

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/ndjson"
)

const ndjsonContentType = "application/x-ndjson"

// openStream opens req and returns a stream of NDJSON frames.
func openStream[T any](s *Service, ctx context.Context, req *core.Request, opts ...core.RequestOption) (core.Stream[T], error) {
	req.SetHeader("Accept", ndjsonContentType)
	resp, err := s.raw(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, core.ErrorFromResponse(resp)
	}
	if resp.Body == nil {
		return nil, errors.New("commands: response has no body")
	}
	return newNDJSONStream[T](resp.Body), nil
}

// ndjsonStream decodes one frame per Next call and closes the response body exactly once.
type ndjsonStream[T any] struct {
	dec       *ndjson.Decoder[T]
	body      io.Closer
	closeOnce sync.Once
	closeErr  error
	stateMu   sync.Mutex
	done      bool
	err       error
}

func newNDJSONStream[T any](body io.ReadCloser) core.Stream[T] {
	return &ndjsonStream[T]{dec: ndjson.NewDecoder[T](body), body: body}
}

// Next returns the next frame or io.EOF at the natural end. A malformed frame or read error is
// terminal and is returned again by later calls.
func (s *ndjsonStream[T]) Next(ctx context.Context) (T, error) {
	var zero T
	if ctx == nil {
		ctx = context.Background()
	}

	s.stateMu.Lock()
	if s.done {
		err := s.err
		s.stateMu.Unlock()
		if err == nil {
			return zero, io.EOF
		}
		return zero, err
	}
	s.stateMu.Unlock()

	if err := ctx.Err(); err != nil {
		_ = s.Close()
		s.finish(err)
		return zero, err
	}

	if ctx.Done() == nil {
		frame, err := s.dec.Next()
		return s.frame(frame, err)
	}
	return s.nextCancelable(ctx)
}

func (s *ndjsonStream[T]) nextCancelable(ctx context.Context) (T, error) {
	type result struct {
		frame T
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		frame, err := s.dec.Next()
		ch <- result{frame: frame, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = s.Close()
		s.finish(ctx.Err())
		var zero T
		return zero, ctx.Err()
	case r := <-ch:
		return s.frame(r.frame, r.err)
	}
}

func (s *ndjsonStream[T]) frame(frame T, err error) (T, error) {
	var zero T
	if err == nil {
		return frame, nil
	}
	_ = s.Close()
	if errors.Is(err, io.EOF) {
		s.finish(nil)
		return zero, io.EOF
	}
	s.finish(err)
	return zero, err
}

func (s *ndjsonStream[T]) finish(err error) {
	s.stateMu.Lock()
	s.done = true
	s.err = err
	s.stateMu.Unlock()
}

// Close closes the response body. It is safe to call more than once.
func (s *ndjsonStream[T]) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.body.Close()
	})
	return s.closeErr
}
