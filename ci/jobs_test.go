package ci_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Job_ReadsJob(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci/jobs/" + jobID,
		Body: `{"ok":true,"type":"ci.job","job":{"id":"122395651084","repo":"acme/api",
			"runId":"8001","jobName":"test","workflowName":"ci","headBranch":"main","headSha":"abc123",
			"htmlUrl":"https://github.com/acme/api/actions/runs/8001","label":"boat/debug",
			"machineClass":"default","state":"success","sandboxId":"bx_example","sandboxState":"kept",
			"queuedAt":"2026-11-13T09:00:00Z","runStartedAt":"2026-11-13T09:00:10Z",
			"finishedAt":"2026-11-13T09:02:00Z","waitSeconds":10,"runSeconds":110,"costDollars":0.5}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Accept"); got != "application/json" {
				t.Fatalf("accept = %q", got)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Job(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Job: %v", err)
	}
	if got.ID != jobID || got.State != model.CiJobStateSuccess {
		t.Fatalf("job = %#v", got)
	}
	if got.WorkflowName == nil || *got.WorkflowName != "ci" {
		t.Fatalf("workflow = %v", got.WorkflowName)
	}
	if got.SandboxState == nil || *got.SandboxState != model.CiJobSandboxStateKept {
		t.Fatalf("sandbox state = %v", got.SandboxState)
	}
	if got.FinishedAt == nil {
		t.Fatal("finished at = nil, want a timestamp")
	}
	if got.RunSeconds == nil || *got.RunSeconds != 110 {
		t.Fatalf("run seconds = %v", got.RunSeconds)
	}
}

func TestService_JobLog_ReadsLog(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci/jobs/" + jobID + "/logs",
		Body:   `{"ok":true,"type":"ci.job.logs","log":"2026-11-13T09:00:00.0000000Z line one\n2026-11-13T09:00:01.0000000Z line two\n"}`,
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.JobLog(context.Background(), jobID)
	if err != nil {
		t.Fatalf("JobLog: %v", err)
	}
	if !got.Ok || got.Type != model.CiJobLogs {
		t.Fatalf("envelope = %#v", got)
	}
	want := "2026-11-13T09:00:00.0000000Z line one\n2026-11-13T09:00:01.0000000Z line two\n"
	if got.Log != want {
		t.Fatalf("log = %q, want %q", got.Log, want)
	}
}

func TestService_JobLog_EmptyBeforeStart(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci/jobs/" + jobID + "/logs",
		Body:   `{"ok":true,"type":"ci.job.logs","log":""}`,
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.JobLog(context.Background(), jobID)
	if err != nil {
		t.Fatalf("JobLog: %v", err)
	}
	if got.Log != "" {
		t.Fatalf("log = %q, want empty", got.Log)
	}
}

func TestService_JobLog_HandlesOneMiBLog(t *testing.T) {
	t.Parallel()

	log := strings.Repeat("x", 1<<20)
	payload, err := json.Marshal(map[string]any{
		"ok":   true,
		"type": "ci.job.logs",
		"log":  log,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci/jobs/" + jobID + "/logs",
		Body:   string(payload),
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.JobLog(context.Background(), jobID)
	if err != nil {
		t.Fatalf("JobLog: %v", err)
	}
	if len(got.Log) != 1<<20 {
		t.Fatalf("log length = %d, want %d", len(got.Log), 1<<20)
	}
}

func TestService_JobAndJobLog_EscapeJobID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		call func(context.Context, *ci.Service) error
	}{
		{
			name: "job",
			path: "/ci/jobs/12%2F34",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Job(ctx, "12/34")
				return err
			},
		},
		{
			name: "job log",
			path: "/ci/jobs/12%2F34/logs",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.JobLog(ctx, "12/34")
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{DoFunc: func(_ context.Context, req *core.Request, _ any, _ ...core.RequestOption) error {
				if req.Path != tt.path {
					t.Fatalf("path = %q, want %q", req.Path, tt.path)
				}
				return nil
			}}
			if err := tt.call(context.Background(), ci.New(d)); err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(d.Calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(d.Calls))
			}
		})
	}
}

func TestService_JobAndJobLog_RequireJobID(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	svc := ci.New(d)
	ctx := context.Background()
	if _, err := svc.Job(ctx, ""); err == nil {
		t.Fatal("Job = nil, want job id error")
	}
	if _, err := svc.JobLog(ctx, ""); err == nil {
		t.Fatal("JobLog = nil, want job id error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(d.Calls))
	}
}
