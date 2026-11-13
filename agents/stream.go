package agents

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

const defaultPollInterval = time.Second

// StreamEvents returns a forward-only stream of sandbox events.
//
// There is no SSE or WebSocket endpoint: the stream long-polls Events with sort=asc, the last
// opaque cursor and the last ETag, about once per second. An unchanged page (304) or an empty page
// backs off with the context instead of spinning. The first request is made before StreamEvents
// returns, so authentication and not-found errors surface immediately.
//
// The stream ends with io.EOF only after Close; otherwise it keeps polling until the context is
// canceled, which returns the context error. Close is idempotent, holds no response body and may be
// called from another goroutine to stop a waiting Next or an in-flight poll. A stream is not safe
// for concurrent Next calls.
func (s *Service) StreamEvents(ctx context.Context, sandboxID string, req EventsRequest, opts ...core.RequestOption) (core.Stream[model.SandboxEvent], error) {
	if err := require(sandboxID, "sandbox id"); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	interval := req.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	maxBackoff := 30 * interval
	if maxBackoff > 30*time.Second {
		maxBackoff = 30 * time.Second
	}
	if maxBackoff < interval {
		maxBackoff = interval
	}

	req.Sort = "asc"
	streamCtx, cancel := context.WithCancel(ctx)
	st := &eventStream{
		svc:        s,
		sandboxID:  sandboxID,
		req:        req,
		opts:       opts,
		cursor:     req.Cursor,
		etag:       req.ETag,
		interval:   interval,
		maxBackoff: maxBackoff,
		ctx:        streamCtx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
	if err := st.poll(streamCtx); err != nil {
		cancel()
		return nil, err
	}
	return st, nil
}

type eventStream struct {
	svc       *Service
	sandboxID string
	req       EventsRequest
	opts      []core.RequestOption

	pending []model.SandboxEvent
	cursor  string
	etag    string
	hasMore bool

	empty      int
	interval   time.Duration
	maxBackoff time.Duration

	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
}

// Next returns the next event, waiting for it when the sandbox has produced everything so far.
func (st *eventStream) Next(ctx context.Context) (model.SandboxEvent, error) {
	var zero model.SandboxEvent
	ctx, cancel := st.mergeContext(ctx)
	defer cancel()

	for {
		if st.closed() {
			return zero, io.EOF
		}

		if len(st.pending) > 0 {
			event := st.pending[0]
			st.pending = st.pending[1:]
			return event, nil
		}
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		if !st.hasMore {
			if err := st.wait(ctx); err != nil {
				if st.closed() {
					return zero, io.EOF
				}
				return zero, err
			}
		}
		if err := st.poll(ctx); err != nil {
			if st.closed() {
				return zero, io.EOF
			}
			return zero, err
		}
	}
}

// Close stops the stream. It is safe to call more than once and from another goroutine.
func (st *eventStream) Close() error {
	st.closeOnce.Do(func() {
		close(st.done)
		st.cancel()
	})
	return nil
}

func (st *eventStream) closed() bool {
	select {
	case <-st.done:
		return true
	default:
		return false
	}
}

// mergeContext returns a context canceled when either the caller's context or the stream context
// ends. It lets Close abort an in-flight poll.
func (st *eventStream) mergeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	merged, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(st.ctx, cancel)
	return merged, func() {
		stop()
		cancel()
	}
}

func (st *eventStream) poll(ctx context.Context) error {
	req := st.req
	req.Cursor = st.cursor
	req.ETag = st.etag
	req.Sort = "asc"

	page, err := st.svc.events(ctx, st.sandboxID, req, st.opts...)
	if err != nil {
		return err
	}
	if page.HasMore && page.NextCursor == "" {
		return errors.New("agents: events page has more items but no next cursor")
	}

	st.etag = page.ETag
	if page.NextCursor != "" {
		st.cursor = page.NextCursor
	}
	// Empty pages must wait before the next poll, even if the server claims there is more,
	// otherwise a malformed response could spin the stream.
	st.hasMore = page.HasMore && len(page.Items) > 0
	st.pending = append(st.pending[:0], page.Items...)
	if len(page.Items) == 0 {
		st.empty++
	} else {
		st.empty = 0
	}
	return nil
}

func (st *eventStream) wait(ctx context.Context) error {
	delay := st.interval
	for i := 1; i < st.empty && delay < st.maxBackoff; i++ {
		delay *= 2
		if delay > st.maxBackoff {
			delay = st.maxBackoff
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-st.done:
		return io.EOF
	case <-timer.C:
		return nil
	}
}
