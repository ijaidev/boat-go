package commands

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// EventType is the kind of a RunCommand event.
type EventType string

const (
	// EventStarted reports the detached process id and command history id.
	EventStarted EventType = "started"
	// EventStdout carries one stdout chunk in Event.Data.
	EventStdout EventType = "stdout"
	// EventStderr carries one stderr chunk in Event.Data.
	EventStderr EventType = "stderr"
	// EventExit reports the final status, exit code and signal.
	EventExit EventType = "exit"
	// EventError reports an attach failure. The command may have run; inspect Event.Error.
	EventError EventType = "error"
)

// Event is one event from RunCommand.
type Event struct {
	Type      EventType
	ProcessID int
	CommandID string
	Data      string
	ExitCode  *int
	Signal    string
	Status    model.CommandAttachFrameStatus
	Error     string
	Message   string
}

// RunCommand starts a command detached, attaches to its output and follows it until it exits.
//
// It reconnects when the attach stream ends with a reconnect frame, using the offsets in that frame,
// so output is neither lost nor duplicated. Heartbeat frames are hidden. An attach error frame is
// returned as EventError and ends the stream. Close releases the current attach stream but does not
// stop the command; use Stop for that.
//
// If Start fails, RunCommand returns the start error. If the first Attach fails, it returns the
// attach error and leaves the detached command running.
func (s *Service) RunCommand(ctx context.Context, sandboxID string, req RunRequest, opts ...core.RequestOption) (core.Stream[Event], error) {
	started, err := s.Start(ctx, sandboxID, req, opts...)
	if err != nil {
		return nil, err
	}
	inner, err := s.Attach(ctx, sandboxID, started.ProcessID, AttachRequest{}, opts...)
	if err != nil {
		return nil, err
	}
	return &followStream{
		svc:       s,
		sandboxID: sandboxID,
		processID: started.ProcessID,
		started:   started,
		inner:     inner,
		opts:      append([]core.RequestOption(nil), opts...),
	}, nil
}

type followStream struct {
	svc       *Service
	sandboxID string
	processID int
	started   *model.CommandStartedResponse
	opts      []core.RequestOption

	stdoutOffset int
	stderrOffset int
	inner        core.Stream[model.CommandAttachFrame]

	mu          sync.Mutex
	startedSent bool
	done        bool
	err         error
}

func (f *followStream) Next(ctx context.Context) (Event, error) {
	var zero Event

	f.mu.Lock()
	if f.done {
		err := f.err
		f.mu.Unlock()
		if err != nil {
			return zero, err
		}
		return zero, io.EOF
	}
	if !f.startedSent {
		f.startedSent = true
		event := f.startedEvent()
		f.mu.Unlock()
		return event, nil
	}
	inner := f.inner
	f.mu.Unlock()

	for {
		frame, err := inner.Next(ctx)
		if err != nil {
			f.mu.Lock()
			closed := f.done
			if closed {
				f.mu.Unlock()
				_ = inner.Close()
				return zero, io.EOF
			}
			f.done = true
			f.err = err
			f.inner = nil
			f.mu.Unlock()
			_ = inner.Close()
			return zero, err
		}

		switch frame.Type {
		case model.CommandAttachFrameTypeAttached:
			f.applyOffsets(frame)
		case model.CommandAttachFrameTypeStdout:
			f.applyOffsets(frame)
			return Event{Type: EventStdout, ProcessID: f.processID, Data: stringValue(frame.Data)}, nil
		case model.CommandAttachFrameTypeStderr:
			f.applyOffsets(frame)
			return Event{Type: EventStderr, ProcessID: f.processID, Data: stringValue(frame.Data)}, nil
		case model.CommandAttachFrameTypeHeartbeat:
			continue
		case model.CommandAttachFrameTypeReconnect:
			f.applyOffsets(frame)
			_ = inner.Close()
			next, attachErr := f.svc.Attach(ctx, f.sandboxID, f.processID, AttachRequest{
				StdoutOffset: f.stdoutOffset,
				StderrOffset: f.stderrOffset,
			}, f.opts...)
			if attachErr != nil {
				f.mu.Lock()
				f.done = true
				f.err = attachErr
				f.inner = nil
				f.mu.Unlock()
				return zero, attachErr
			}
			f.mu.Lock()
			if f.done {
				f.mu.Unlock()
				_ = next.Close()
				return zero, io.EOF
			}
			f.inner = next
			f.mu.Unlock()
			inner = next
		case model.CommandAttachFrameTypeExit:
			f.applyOffsets(frame)
			_ = inner.Close()
			f.mu.Lock()
			f.done = true
			f.inner = nil
			f.mu.Unlock()
			return Event{
				Type:      EventExit,
				ProcessID: f.processID,
				ExitCode:  frame.ExitCode,
				Signal:    stringValue(frame.Signal),
				Status:    statusValue(frame.Status),
			}, nil
		case model.CommandAttachFrameTypeError:
			f.applyOffsets(frame)
			_ = inner.Close()
			f.mu.Lock()
			f.done = true
			f.inner = nil
			f.mu.Unlock()
			return Event{
				Type:      EventError,
				ProcessID: f.processID,
				Error:     stringValue(frame.Error),
				Message:   stringValue(frame.Message),
			}, nil
		default:
			_ = inner.Close()
			f.mu.Lock()
			f.done = true
			f.err = fmt.Errorf("commands: unknown attach frame type %q", frame.Type)
			f.inner = nil
			err := f.err
			f.mu.Unlock()
			return zero, err
		}
	}
}

// Close releases the current attach stream. It does not stop the detached command.
//
// Close does not wait for an in-flight Next: closing the response body unblocks it, and Next then
// returns io.EOF.
func (f *followStream) Close() error {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return nil
	}
	f.done = true
	inner := f.inner
	f.inner = nil
	f.mu.Unlock()
	if inner == nil {
		return nil
	}
	return inner.Close()
}

func (f *followStream) startedEvent() Event {
	commandID := ""
	if f.started != nil && f.started.CommandID != nil {
		commandID = *f.started.CommandID
	}
	return Event{Type: EventStarted, ProcessID: f.processID, CommandID: commandID}
}

func (f *followStream) applyOffsets(frame model.CommandAttachFrame) {
	switch frame.Type {
	case model.CommandAttachFrameTypeStdout:
		if frame.NextOffset != nil {
			f.stdoutOffset = *frame.NextOffset
		}
	case model.CommandAttachFrameTypeStderr:
		if frame.NextOffset != nil {
			f.stderrOffset = *frame.NextOffset
		}
	}
	if frame.StdoutOffset != nil {
		f.stdoutOffset = *frame.StdoutOffset
	}
	if frame.StderrOffset != nil {
		f.stderrOffset = *frame.StderrOffset
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func statusValue(status *model.CommandAttachFrameStatus) model.CommandAttachFrameStatus {
	if status == nil {
		return ""
	}
	return *status
}
