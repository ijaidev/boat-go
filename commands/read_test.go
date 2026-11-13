package commands_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Status_SendsTailAndOffsets(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/commands/77",
		Body:   `{"ok":true,"type":"command.status","success":true,"processId":77,"status":"running","running":true,"exitCode":null,"stdout":"hi","stderr":"","stdoutNextOffset":2,"stderrNextOffset":0}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "tailBytes", "1024")
			requireQuery(t, r, "stdoutOffset", "0")
			requireQuery(t, r, "stderrOffset", "3")
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	status, err := svc.Status(context.Background(), "bx_example", 77, commands.StatusRequest{
		TailBytes:    core.Ptr(1024),
		StdoutOffset: core.Ptr(0),
		StderrOffset: core.Ptr(3),
	})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.ProcessID != 77 || status.Status != model.CommandStatusResponseStatusRunning || !status.Running {
		t.Fatalf("status = %#v", status)
	}
	if status.Stdout != "hi" {
		t.Fatalf("stdout = %q", status.Stdout)
	}
	if status.StdoutNextOffset == nil || *status.StdoutNextOffset != 2 {
		t.Fatalf("stdout next offset = %v", status.StdoutNextOffset)
	}
}

func TestService_Status_OmitsDefaultQuery(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/commands/77",
		Body:   `{"ok":true,"type":"command.status","success":true,"processId":77,"status":"exited","running":false,"exitCode":0,"stdout":"","stderr":""}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.RawQuery; got != "" {
				t.Fatalf("query = %q, want empty", got)
			}
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Status(context.Background(), "bx_example", 77, commands.StatusRequest{}); err != nil {
		t.Fatalf("Status: %v", err)
	}
}

func TestService_Status_RejectsInvalidArguments(t *testing.T) {
	t.Parallel()

	svc := commands.New(&testutil.Doer{})
	tests := []struct {
		name      string
		sandboxID string
		processID int
		req       commands.StatusRequest
	}{
		{name: "missing sandbox", processID: 1},
		{name: "missing process", sandboxID: "bx_example"},
		{name: "negative tail", sandboxID: "bx_example", processID: 1, req: commands.StatusRequest{TailBytes: core.Ptr(-1)}},
		{name: "negative stdout offset", sandboxID: "bx_example", processID: 1, req: commands.StatusRequest{StdoutOffset: core.Ptr(-1)}},
		{name: "negative stderr offset", sandboxID: "bx_example", processID: 1, req: commands.StatusRequest{StderrOffset: core.Ptr(-1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := svc.Status(context.Background(), tt.sandboxID, tt.processID, tt.req); err == nil {
				t.Fatal("Status = nil, want error")
			}
		})
	}
}

func TestService_Stop_ReturnsFinalRecord(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/sandboxes/bx_example/commands/77",
		Body:   `{"ok":true,"type":"command.stopped","processId":77,"running":false,"signaled":true,"command":{"commandId":"cmd_1","sandboxId":"bx_example","command":"sleep 100","mode":"detached","status":"killed","stdoutBytes":0,"stderrBytes":0,"outputTruncated":false,"startedAt":"2026-01-02T03:04:05Z","processId":77}}`,
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	stop, err := svc.Stop(context.Background(), "bx_example", 77)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stop.ProcessID != 77 || stop.Running || !stop.Signaled {
		t.Fatalf("stop = %#v", stop)
	}
	if stop.Command == nil || stop.Command.CommandID != "cmd_1" {
		t.Fatalf("command = %#v", stop.Command)
	}
}

func TestService_History_Paginates(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/command-history",
		Body: `{"ok":true,"type":"command.list","commands":[
			{"commandId":"cmd_2","sandboxId":"bx_example","command":"npm test","mode":"detached","status":"running","stdoutBytes":0,"stderrBytes":0,"outputTruncated":false,"startedAt":"2026-01-02T03:05:05Z"}
		],"nextCursor":"cur_2"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "mode", "detached,process")
			requireQuery(t, r, "q", "npm run")
			requireQuery(t, r, "limit", "5")
			requireQuery(t, r, "cursor", "cur_1")
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.History(context.Background(), "bx_example", commands.HistoryRequest{
		Mode:   "detached,process",
		Query:  "npm run",
		Limit:  5,
		Cursor: "cur_1",
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].CommandID != "cmd_2" {
		t.Fatalf("items = %#v", page.Items)
	}
	if !page.HasMore || page.NextCursor != "cur_2" {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_History_LastPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/command-history",
		Body:   `{"ok":true,"type":"command.list","commands":[],"nextCursor":null}`,
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.History(context.Background(), "bx_example", commands.HistoryRequest{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(page.Items) != 0 || page.HasMore || page.NextCursor != "" {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_GetHistory_ReturnsRecord(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/command-history/cmd_1",
		Body:   `{"ok":true,"type":"command.info","command":{"commandId":"cmd_1","sandboxId":"bx_example","command":"echo hi","mode":"sync","status":"exited","exitCode":0,"stdoutBytes":3,"stderrBytes":0,"outputTruncated":false,"startedAt":"2026-01-02T03:04:05Z"}}`,
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	record, err := svc.GetHistory(context.Background(), "bx_example", "cmd_1")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if record.CommandID != "cmd_1" || record.Mode != model.Sync || record.Status != model.CommandRecordStatusExited {
		t.Fatalf("record = %#v", record)
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("exit code = %v", record.ExitCode)
	}
}

func TestService_Logs_SendsReadOptions(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/command-history/cmd_1/logs",
		Body:   `{"ok":true,"type":"command.logs","commandId":"cmd_1","stream":"stderr","offset":10,"nextOffset":12,"size":12,"encoding":"base64","data":"aGk=","done":false}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "stream", "stderr")
			requireQuery(t, r, "offset", "10")
			requireQuery(t, r, "limit", "1024")
			requireQuery(t, r, "encoding", "base64")
		},
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	logs, err := svc.Logs(context.Background(), "bx_example", "cmd_1", commands.LogsRequest{
		Stream:   model.CommandLogsParamsStreamStderr,
		Offset:   10,
		Limit:    1024,
		Encoding: model.CommandLogsParamsEncodingBase64,
	})
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if logs.CommandID != "cmd_1" || logs.Stream != model.CommandLogsResponseStreamStderr {
		t.Fatalf("logs = %#v", logs)
	}
	if logs.NextOffset != 12 || logs.Data != "aGk=" || logs.Done {
		t.Fatalf("logs = %#v", logs)
	}
}

func TestService_Processes_ReturnsList(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/processes",
		Body:   `{"ok":true,"type":"process.list","live":true,"sampledAt":"2026-01-02T03:04:05Z","total":2,"processes":[{"pid":1,"command":"zsh","cpuPercent":0.5,"memoryMB":10}]}`,
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	processes, err := svc.Processes(context.Background(), "bx_example")
	if err != nil {
		t.Fatalf("Processes: %v", err)
	}
	if !processes.Live || processes.Total != 2 || len(processes.Processes) != 1 {
		t.Fatalf("processes = %#v", processes)
	}
	if processes.Processes[0].Pid != 1 || processes.Processes[0].Command != "zsh" {
		t.Fatalf("process = %#v", processes.Processes[0])
	}
	if processes.SampledAt == nil {
		t.Fatal("sampledAt = nil")
	}
}

func TestService_GetHistory_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/command-history/cmd_missing",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"sandbox.error","status":404,"code":"not_found","message":"no such command","requestId":"req_1"}`,
	})
	svc := commands.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.GetHistory(context.Background(), "bx_example", "cmd_missing")
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}

func TestService_ReadRequiresIDs(t *testing.T) {
	t.Parallel()

	svc := commands.New(&testutil.Doer{})
	if _, err := svc.GetHistory(context.Background(), "", "cmd_1"); err == nil {
		t.Fatal("GetHistory missing sandbox = nil, want error")
	}
	if _, err := svc.GetHistory(context.Background(), "bx_example", ""); err == nil {
		t.Fatal("GetHistory missing command = nil, want error")
	}
	if _, err := svc.History(context.Background(), "", commands.HistoryRequest{}); err == nil {
		t.Fatal("History missing sandbox = nil, want error")
	}
	if _, err := svc.Logs(context.Background(), "", "cmd_1", commands.LogsRequest{}); err == nil {
		t.Fatal("Logs missing sandbox = nil, want error")
	}
	if _, err := svc.Logs(context.Background(), "bx_example", "", commands.LogsRequest{}); err == nil {
		t.Fatal("Logs missing command = nil, want error")
	}
	if _, err := svc.Processes(context.Background(), ""); err == nil {
		t.Fatal("Processes missing sandbox = nil, want error")
	}
	if _, err := svc.Stop(context.Background(), "", 1); err == nil {
		t.Fatal("Stop missing sandbox = nil, want error")
	}
	if _, err := svc.Stop(context.Background(), "bx_example", 0); err == nil {
		t.Fatal("Stop missing process = nil, want error")
	}
}
