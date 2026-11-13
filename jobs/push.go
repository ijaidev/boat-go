package jobs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ijaidev/boat-go/commands"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
)

const extractTimeoutSeconds = 600

// Push uploads the git work tree in sourceDir and extracts it into remoteDir inside the sandbox.
//
// It lists tracked and untracked-not-ignored files with git ls-files, skips deleted paths and
// submodule directories, and archives regular files and symlinks without following links. The
// archive is built locally, uploaded through a signed file URL, and extracted with tar. A remoteDir
// of "" means ".".
//
// Push requires git on the local machine and tar on the sandbox. Request options apply to the
// upload-URL and extract calls. The signed upload URL is never included in an error.
func (s *Service) Push(ctx context.Context, sandboxID, sourceDir, remoteDir string, opts ...core.RequestOption) error {
	if err := s.requirePush(); err != nil {
		return err
	}
	if sandboxID == "" {
		return errors.New("jobs: sandbox id is required")
	}
	if sourceDir == "" {
		return errors.New("jobs: source directory is required")
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("jobs: source directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("jobs: source directory %q is not a directory", sourceDir)
	}

	paths, err := gitPaths(ctx, sourceDir)
	if err != nil {
		return err
	}
	archiveName, err := newArchiveName()
	if err != nil {
		return err
	}

	archive, err := os.CreateTemp("", "boat-push-*.tgz")
	if err != nil {
		return fmt.Errorf("jobs: create archive: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)

	if err := writeArchive(archive, sourceDir, paths); err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("jobs: close archive: %w", err)
	}

	signed, err := s.files.UploadURL(ctx, sandboxID, files.URLRequest{Path: archiveName}, opts...)
	if err != nil {
		return fmt.Errorf("jobs: get upload URL: %w", err)
	}
	if err := s.uploadArchive(ctx, signed.URL, archivePath); err != nil {
		return err
	}
	return s.extractArchive(ctx, sandboxID, archiveName, remoteDir, opts...)
}

func (s *Service) uploadArchive(ctx context.Context, signedURL, archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("jobs: open archive: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, file)
	if err != nil {
		return errors.New("jobs: upload archive: invalid signed upload URL")
	}
	req.Header.Set("Content-Type", "application/gzip")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return requestFailed("upload archive", err)
	}
	drainAndClose(resp)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("jobs: upload archive: http %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) extractArchive(ctx context.Context, sandboxID, archiveName, remoteDir string, opts ...core.RequestOption) error {
	remote := remoteDir
	if remote == "" {
		remote = "."
	}
	command := fmt.Sprintf(
		"mkdir -p %s && tar xzf %s -C %s && rm %s",
		shellQuote(remote),
		shellQuote(archiveName),
		shellQuote(remote),
		shellQuote(archiveName),
	)
	out, err := s.commands.Run(ctx, sandboxID, commands.RunRequest{
		Command:        command,
		TimeoutSeconds: core.Ptr(extractTimeoutSeconds),
	}, opts...)
	if err != nil {
		return fmt.Errorf("jobs: extract archive: %w", err)
	}
	if out.ExitCode != nil && *out.ExitCode != 0 {
		return fmt.Errorf("jobs: extract archive: exit code %d", *out.ExitCode)
	}
	if !out.Success {
		return errors.New("jobs: extract archive failed")
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
