package core

import "context"

// Stream is a forward-only stream of frames.
//
// Next returns io.EOF after the natural end of the stream. Any other error is terminal unless the
// implementation documents otherwise. Close releases the underlying response body and is safe to
// call more than once.
type Stream[T any] interface {
	Next(ctx context.Context) (T, error)
	Close() error
}
