package ci

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Job returns one GitHub Actions job that Boat ran or refused.
//
// The job carries its state, machine class, sandbox, wait and run time, and cost. It also works after
// the sandbox is gone. JobID is the GitHub workflow job id. The API key needs account.read.
func (s *Service) Job(ctx context.Context, jobID string, opts ...core.RequestOption) (*model.CiJob, error) {
	if err := requireJobID(jobID); err != nil {
		return nil, err
	}
	var out model.CiJobResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   jobPath(jobID),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Job, nil
}

// JobLog returns a job's log from GitHub.
//
// The spec defines a JSON response with the log in string form, one timestamped line each, capped at
// the last 1 MB. It is empty before the job starts: poll JobLog while the job runs.
// The log may contain repository output; do not log it wholesale. The API key needs account.read.
func (s *Service) JobLog(ctx context.Context, jobID string, opts ...core.RequestOption) (*model.CiJobLogResponse, error) {
	if err := requireJobID(jobID); err != nil {
		return nil, err
	}
	var out model.CiJobLogResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   jobPath(jobID, "logs"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
