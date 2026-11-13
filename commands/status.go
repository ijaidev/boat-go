package commands

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// StatusRequest selects how much output Status returns.
//
// With no offsets, the answer holds the end of each log. With StdoutOffset or StderrOffset, it
// holds only the output written since those byte offsets. TailBytes caps each log when no offset is
// used; the server default is 8 MiB.
type StatusRequest struct {
	TailBytes    *int
	StdoutOffset *int
	StderrOffset *int
}

// Status returns the status and log tails of a detached command.
func (s *Service) Status(ctx context.Context, sandboxID string, processID int, req StatusRequest, opts ...core.RequestOption) (*model.CommandStatusResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireProcessID(processID); err != nil {
		return nil, err
	}
	if err := validateOptionalNonNegative("tail bytes", req.TailBytes); err != nil {
		return nil, err
	}
	if err := validateOptionalNonNegative("stdout offset", req.StdoutOffset); err != nil {
		return nil, err
	}
	if err := validateOptionalNonNegative("stderr offset", req.StderrOffset); err != nil {
		return nil, err
	}

	query := make(url.Values)
	setOptionalInt(query, "tailBytes", req.TailBytes)
	setOptionalInt(query, "stdoutOffset", req.StdoutOffset)
	setOptionalInt(query, "stderrOffset", req.StderrOffset)

	var out model.CommandStatusResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   commandPath(sandboxID, processID),
		Query:  query,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Stop sends SIGTERM, then SIGKILL after five seconds, to a detached command and returns its final
// record.
func (s *Service) Stop(ctx context.Context, sandboxID string, processID int, opts ...core.RequestOption) (*model.CommandStopResponse, error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireProcessID(processID); err != nil {
		return nil, err
	}
	var out model.CommandStopResponse
	if err := s.do(ctx, &core.Request{
		Method:  http.MethodDelete,
		Path:    commandPath(sandboxID, processID),
		NoRetry: true,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func validateOptionalNonNegative(name string, value *int) error {
	if value != nil && *value < 0 {
		return errors.New("commands: " + name + " must not be negative")
	}
	return nil
}

func setOptionalInt(query url.Values, key string, value *int) {
	if value != nil {
		query.Set(key, strconv.Itoa(*value))
	}
}
