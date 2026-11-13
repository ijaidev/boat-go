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
	f.mu.Lock()
	defer f.mu.Unlock()

	var zero Event
	if f.done {
		if f.err != nil {
			return zero, f.err
		}
		return zero, io.EOF
	}
	if !f.startedSent {
		f.startedSent = true
		return f.startedEvent(), nil
	}

	for {
		frame, err := f.inner.Next(ctx)
		if err != nil {
			_ = f.inner.Close()
			f.done = true
			f.err = err
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
			_ = f.inner.Close()
			inner, err := f.svc.Attach(ctx, f.sandboxID, f.processID, AttachRequest{
				StdoutOffset: f.stdoutOffset,
				StderrOffset: f.stderrOffset,
			}, f.opts...)
			if err != nil {
				f.done = true
				f.err = err
				return zero, err
			}
			f.inner = inner
		case model.CommandAttachFrameTypeExit:
			f.applyOffsets(frame)
			_ = f.inner.Close()
			f.done = true
			return Event{
				Type:      EventExit,
				ProcessID: f.processID,
				ExitCode:  frame.ExitCode,
				Signal:    stringValue(frame.Signal),
				Status:    statusValue(frame.Status),
			}, nil
		case model.CommandAttachFrameTypeError:
			f.applyOffsets(frame)
			_ = f.inner.Close()
			f.done = true
			return Event{
				Type:      EventError,
				ProcessID: f.processID,
				Error:     stringValue(frame.Error),
				Message:   stringValue(frame.Message),
			}, nil
		default:
			_ = f.inner.Close()
			f.done = true
			f.err = fmt.Errorf("commands: unknown attach frame type %q", frame.Type)
			return zero, f.err
		}
	}
}

// Close releases the current attach stream. It does not stop the detached command.
func (f *followStream) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.done && f.inner == nil {
		return nil
	}
	f.done = true
	if f.inner == nil {
		return nil
	}
	err := f.inner.Close()
	f.inner = nil
	return err
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
