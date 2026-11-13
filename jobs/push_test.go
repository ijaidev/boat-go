package jobs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
)

const commandResultBody = `{"ok":true,"type":"command.result","command":"tar","exitCode":0,"success":true,"stdout":"","stderr":""}`

type tarEntry struct {
	mode     int64
	typeflag byte
	link     string
	body     string
}

func TestService_Push_ArchivesGitWorkTree(t *testing.T) {
	t.Parallel()

	sourceDir := newPushRepo(t)

	var (
		uploaded      []byte
		uploadPath    string
		extractCmd    string
		extractCwd    *string
		extractTimout *int
	)
	uploadSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/upload",
		Status: http.StatusOK,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = body
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("Authorization = %q, want none on a signed upload", got)
			}
		},
	})
	signedURL := uploadSrv.URL + "/upload?sig=secret-token"

	apiSrv := testutil.NewServer(t,
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes/bx_1/files/upload-url",
			Status: http.StatusOK,
			Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"PUT","path":".boat-push-test.tgz","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL),
			Assert: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req files.URLRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode upload-url request: %v", err)
				}
				uploadPath = req.Path
				if req.TTLSeconds != nil {
					t.Fatalf("ttlSeconds = %d, want nil", *req.TTLSeconds)
				}
				if !strings.HasPrefix(req.Path, ".boat-push-") || !strings.HasSuffix(req.Path, ".tgz") {
					t.Fatalf("archive path = %q, want .boat-push-*.tgz", req.Path)
				}
				if strings.Contains(req.Path, "/") {
					t.Fatalf("archive path = %q, want a path under /home/user", req.Path)
				}
			},
		},
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes/bx_1/commands",
			Status: http.StatusOK,
			Body:   commandResultBody,
			Assert: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req commands.RunRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode extract request: %v", err)
				}
				extractCmd = req.Command
				extractCwd = req.Cwd
				extractTimout = req.TimeoutSeconds
			},
		},
	)

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{Commands: commands.New(tr), Files: files.New(tr)})

	if err := svc.Push(context.Background(), "bx_1", sourceDir, "work/it's here"); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if len(uploaded) == 0 {
		t.Fatal("upload body is empty")
	}
	if uploadPath == "" {
		t.Fatal("upload-url request did not carry a path")
	}
	if extractCwd != nil {
		t.Fatalf("cwd = %q, want nil", *extractCwd)
	}
	if extractTimout == nil || *extractTimout != 600 {
		t.Fatalf("timeoutSeconds = %v, want 600", extractTimout)
	}
	wantPrefix := "mkdir -p 'work/it'\\''s here'"
	if !strings.HasPrefix(extractCmd, wantPrefix) {
		t.Fatalf("extract command = %q, want prefix %q", extractCmd, wantPrefix)
	}
	if !strings.Contains(extractCmd, "tar xzf ") || !strings.Contains(extractCmd, "-C 'work/it'\\''s here'") {
		t.Fatalf("extract command = %q, want tar extraction into the quoted remote dir", extractCmd)
	}
	if !strings.Contains(extractCmd, "&& rm ") {
		t.Fatalf("extract command = %q, want archive removal", extractCmd)
	}

	entries := readTarArchive(t, uploaded)
	want := map[string]tarEntry{
		".gitignore":     {body: "ignored.log\n"},
		"tracked.txt":    {body: "tracked\n"},
		"src/nested.txt": {body: "nested\n"},
		"run.sh":         {body: "#!/bin/sh\n", mode: 0o755},
		"link":           {typeflag: tar.TypeSymlink, link: "tracked.txt"},
		"untracked.txt":  {body: "untracked\n"},
	}
	for name, wantEntry := range want {
		got, ok := entries[name]
		if !ok {
			t.Errorf("archive is missing %q", name)
			continue
		}
		if got.body != wantEntry.body {
			t.Errorf("%s body = %q, want %q", name, got.body, wantEntry.body)
		}
		if wantEntry.mode != 0 && got.mode&0o777 != wantEntry.mode {
			t.Errorf("%s mode = %#o, want %#o", name, got.mode&0o777, wantEntry.mode)
		}
		if got.typeflag != wantEntry.typeflag {
			wantType := wantEntry.typeflag
			if wantType == 0 {
				wantType = tar.TypeReg
			}
			if got.typeflag != wantType {
				t.Errorf("%s typeflag = %d, want %d", name, got.typeflag, wantType)
			}
		}
		if got.link != wantEntry.link {
			t.Errorf("%s link = %q, want %q", name, got.link, wantEntry.link)
		}
	}
	if len(entries) != len(want) {
		t.Fatalf("archive has %d entries, want %d: %v", len(entries), len(want), entryNames(entries))
	}
	for _, name := range []string{"ignored.log", "deleted.txt", "submodule"} {
		if _, ok := entries[name]; ok {
			t.Errorf("archive contains %q, want it excluded", name)
		}
	}
}

func TestService_Push_ExtractFailure(t *testing.T) {
	t.Parallel()

	sourceDir := newPushRepo(t)
	uploadSrv := testutil.NewServer(t, testutil.Route{Method: http.MethodPut, Path: "/upload", Status: http.StatusOK})
	signedURL := uploadSrv.URL + "/upload"

	apiSrv := testutil.NewServer(t,
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes/bx_1/files/upload-url",
			Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"PUT","path":".boat-push-test.tgz","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL),
		},
		testutil.Route{
			Method: http.MethodPost,
			Path:   "/sandboxes/bx_1/commands",
			Body:   `{"ok":true,"type":"command.result","command":"tar","exitCode":2,"success":false,"stdout":"","stderr":"tar: bad archive"}`,
		},
	)

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{Commands: commands.New(tr), Files: files.New(tr)})

	err := svc.Push(context.Background(), "bx_1", sourceDir, "")
	if err == nil {
		t.Fatal("Push returned nil error, want extract failure")
	}
	if !strings.Contains(err.Error(), "exit code 2") {
		t.Fatalf("error = %q, want exit code 2", err)
	}
}

func TestService_Push_UploadFailureRemovesArchive(t *testing.T) {
	sourceDir := newPushRepo(t)

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	secret := "sig=super-secret"
	uploadSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/upload",
		Status: http.StatusInternalServerError,
	})
	signedURL := uploadSrv.URL + "/upload?" + secret

	apiSrv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/sandboxes/bx_1/files/upload-url",
		Body:   fmt.Sprintf(`{"ok":true,"type":"file.url","url":%q,"method":"PUT","path":".boat-push-test.tgz","expiresAt":"2026-11-13T00:00:00Z"}`, signedURL),
	})

	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{Commands: commands.New(tr), Files: files.New(tr)})

	err := svc.Push(context.Background(), "bx_1", sourceDir, "")
	if err == nil {
		t.Fatal("Push returned nil error, want upload failure")
	}
	if strings.Contains(err.Error(), signedURL) || strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaks the signed URL: %q", err)
	}
	entries, readErr := os.ReadDir(tmp)
	if readErr != nil {
		t.Fatalf("read temp dir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dir still contains %v, want the archive removed", entryNamesDir(entries))
	}
}

func TestService_Push_GitFailure(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	apiSrv := testutil.NewServer(t)
	tr := testutil.Transport(t, apiSrv.URL, "boat_test")
	svc := New(Config{Commands: commands.New(tr), Files: files.New(tr)})

	err := svc.Push(context.Background(), "bx_1", t.TempDir(), "")
	if err == nil {
		t.Fatal("Push returned nil error, want git failure")
	}
	if !strings.Contains(err.Error(), "git ls-files") {
		t.Fatalf("error = %q, want git ls-files context", err)
	}
}

func TestService_Push_RequiresServices(t *testing.T) {
	t.Parallel()

	svc := New(Config{})
	if err := svc.Push(context.Background(), "bx_1", t.TempDir(), ""); err == nil {
		t.Fatal("Push returned nil error, want missing service error")
	}
}

func TestShellQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "simple", want: "'simple'"},
		{in: "with space", want: "'with space'"},
		{in: "it's", want: `'it'\''s'`},
		{in: "a\nb", want: "'a\nb'"},
		{in: "$(rm -rf /)", want: "'$(rm -rf /)'"},
	}
	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func newPushRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	gitCommand(t, dir, "init", "-q")
	gitCommand(t, dir, "config", "user.email", "test@example.com")
	gitCommand(t, dir, "config", "user.name", "Test User")

	writeTestFile(t, filepath.Join(dir, ".gitignore"), "ignored.log\n", 0o644)
	writeTestFile(t, filepath.Join(dir, "tracked.txt"), "tracked\n", 0o644)
	writeTestFile(t, filepath.Join(dir, "src", "nested.txt"), "nested\n", 0o644)
	writeTestFile(t, filepath.Join(dir, "run.sh"), "#!/bin/sh\n", 0o755)
	writeTestFile(t, filepath.Join(dir, "deleted.txt"), "deleted\n", 0o644)
	writeTestFile(t, filepath.Join(dir, "ignored.log"), "ignored\n", 0o644)
	if err := os.Symlink("tracked.txt", filepath.Join(dir, "link")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	gitCommand(t, dir, "add", ".gitignore", "tracked.txt", "src/nested.txt", "run.sh", "deleted.txt", "link")
	gitCommand(t, dir, "commit", "-q", "-m", "initial")
	if err := os.Remove(filepath.Join(dir, "deleted.txt")); err != nil {
		t.Fatalf("remove tracked file: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "untracked.txt"), "untracked\n", 0o644)

	if err := os.Mkdir(filepath.Join(dir, "submodule"), 0o755); err != nil {
		t.Fatalf("create submodule dir: %v", err)
	}
	gitCommand(t, dir, "update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,submodule")
	return dir
}

func gitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readTarArchive(t *testing.T, data []byte) map[string]tarEntry {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer gz.Close()

	entries := make(map[string]tarEntry)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar entry %q: %v", hdr.Name, err)
		}
		entries[hdr.Name] = tarEntry{
			mode:     hdr.Mode,
			typeflag: hdr.Typeflag,
			link:     hdr.Linkname,
			body:     string(body),
		}
	}
	return entries
}

func entryNames(entries map[string]tarEntry) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}

func entryNamesDir(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
