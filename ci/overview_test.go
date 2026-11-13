package ci_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Overview_SendsFiltersAndReadsResponse(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci",
		Body: `{"ok":true,"type":"ci.overview",
			"setup":{"complete":false,"next":"workflow",
				"steps":[{"id":"install","title":"Install the app","done":true,"detail":"1 account"}],
				"problems":[{"jobId":"9001","repo":"acme/api","message":"public repository",
					"at":"2026-11-12T10:00:00Z","count":2}]},
			"installations":[{"id":"123456","accountLogin":"acme","accountType":"Organization",
				"allowPublicRepos":true,"publicRepoUsers":["alice"],
				"manageUrl":"https://github.com/organizations/acme/settings/installations/123456",
				"repositories":[{"fullName":"acme/api","private":true,
					"htmlUrl":"https://github.com/acme/api","status":"ok","jobs7d":12,
					"lastState":"success","lastAt":"2026-11-13T09:00:00Z","lastError":null}]}],
			"filter":{"account":"acme","repo":"acme/api","state":"active"},
			"counts":{"all":12,"active":2,"failed":1},
			"stats":{"jobs":12,"running":2,"successRate":0.92,"medianWaitSeconds":8,
				"medianRunSeconds":75,"costDollars":1.25,
				"perDay":[{"day":"2026-11-12","success":2,"failure":1,"other":0}]},
			"jobs":[{"id":"122395651084","repo":"acme/api","runId":"8001","jobName":"test",
				"workflowName":"ci","headBranch":"main","headSha":"abc123",
				"htmlUrl":"https://github.com/acme/api/actions/runs/8001",
				"label":"boat-large/debug","machineClass":"large","state":"running",
				"error":null,"sandboxId":"bx_example","sandboxState":"live",
				"queuedAt":"2026-11-13T09:00:00Z","runStartedAt":"2026-11-13T09:00:10Z",
				"finishedAt":null,"waitSeconds":10,"runSeconds":null,"costDollars":null}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			q := r.URL.Query()
			if got := q.Get("account"); got != "acme" {
				t.Fatalf("account = %q", got)
			}
			if got := q.Get("repo"); got != "acme/api" {
				t.Fatalf("repo = %q", got)
			}
			if got := q.Get("state"); got != "active" {
				t.Fatalf("state = %q", got)
			}
			if got := q.Get("limit"); got != "50" {
				t.Fatalf("limit = %q", got)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer boat_test_key" {
				t.Fatalf("authorization = %q", got)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Overview(context.Background(), ci.OverviewRequest{
		Account: "acme",
		Repo:    "acme/api",
		State:   ci.StateActive,
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if !got.Ok || got.Type != model.CiOverview {
		t.Fatalf("envelope = %#v", got)
	}
	if got.Setup.Next == nil || *got.Setup.Next != model.CiOverviewResponseSetupNextWorkflow {
		t.Fatalf("setup next = %v", got.Setup.Next)
	}
	if got.Filter == nil || got.Filter.State == nil || *got.Filter.State != model.CiOverviewResponseFilterStateActive {
		t.Fatalf("filter = %#v", got.Filter)
	}
	if len(got.Installations) != 1 || got.Installations[0].AccountLogin != "acme" {
		t.Fatalf("installations = %#v", got.Installations)
	}
	if got.Installations[0].Repositories[0].Status != model.CiRepositoryStatusOk {
		t.Fatalf("repository status = %q", got.Installations[0].Repositories[0].Status)
	}
	if got.Counts.Active == nil || *got.Counts.Active != 2 {
		t.Fatalf("active count = %v", got.Counts.Active)
	}
	if len(got.Jobs) != 1 {
		t.Fatalf("jobs = %#v", got.Jobs)
	}
	job := got.Jobs[0]
	if job.ID != jobID || job.State != model.CiJobStateRunning {
		t.Fatalf("job = %#v", job)
	}
	if job.MachineClass != model.CiJobMachineClassLarge {
		t.Fatalf("machine class = %q", job.MachineClass)
	}
	if job.SandboxState == nil || *job.SandboxState != model.CiJobSandboxStateLive {
		t.Fatalf("sandbox state = %v", job.SandboxState)
	}
	if job.WaitSeconds == nil || *job.WaitSeconds != 10 {
		t.Fatalf("wait seconds = %v", job.WaitSeconds)
	}
}

func TestService_Overview_OmitsUnsetFilters(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci",
		Body:   `{"ok":true,"type":"ci.overview","setup":{},"installations":[],"counts":{},"stats":{},"jobs":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if q := r.URL.Query(); len(q) != 0 {
				t.Fatalf("query = %v, want none", q)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Overview(context.Background(), ci.OverviewRequest{}); err != nil {
		t.Fatalf("Overview: %v", err)
	}
}

func TestService_Overview_SendsExplicitAllState(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ci",
		Body:   `{"ok":true,"type":"ci.overview","setup":{},"installations":[],"counts":{},"stats":{},"jobs":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("state"); got != "all" {
				t.Fatalf("state = %q, want all", got)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Overview(context.Background(), ci.OverviewRequest{State: ci.StateAll}); err != nil {
		t.Fatalf("Overview: %v", err)
	}
}

func TestService_Overview_RejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  ci.OverviewRequest
	}{
		{name: "unknown state", req: ci.OverviewRequest{State: ci.State("queued")}},
		{name: "negative limit", req: ci.OverviewRequest{Limit: -1}},
		{name: "limit above cap", req: ci.OverviewRequest{Limit: 501}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if _, err := ci.New(d).Overview(context.Background(), tt.req); err == nil {
				t.Fatal("Overview = nil, want filter error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("calls = %d, want 0", len(d.Calls))
			}
		})
	}
}
