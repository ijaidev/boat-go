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

// AttachRequest selects where the attached output starts. Zero means the beginning of each log.
type AttachRequest struct {
	StdoutOffset int
	StderrOffset int
}

// Attach streams the output of a detached command.
//
// It returns attached, stdout, stderr, heartbeat, reconnect and exit frames. A reconnect frame is
// the last line after ten minutes; open the stream again with its offsets to continue without a gap
// or duplicate output. Next returns io.EOF at the end. A malformed frame is terminal.
func (s *Service) Attach(ctx context.Context, sandboxID string, processID int, req AttachRequest, opts ...core.RequestOption) (core.Stream[model.CommandAttachFrame], error) {
	if err := requireSandboxID(sandboxID); err != nil {
		return nil, err
	}
	if err := requireProcessID(processID); err != nil {
		return nil, err
	}
	if req.StdoutOffset < 0 {
		return nil, errors.New("commands: stdout offset must not be negative")
	}
	if req.StderrOffset < 0 {
		return nil, errors.New("commands: stderr offset must not be negative")
	}

	query := make(url.Values)
	if req.StdoutOffset > 0 {
		query.Set("stdoutOffset", strconv.Itoa(req.StdoutOffset))
	}
	if req.StderrOffset > 0 {
		query.Set("stderrOffset", strconv.Itoa(req.StderrOffset))
	}
	return openStream[model.CommandAttachFrame](s, ctx, &core.Request{
		Method: http.MethodGet,
		Path:   commandPath(sandboxID, processID, "stream"),
		Query:  query,
	}, opts...)
}
