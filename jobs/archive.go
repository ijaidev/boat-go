package jobs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func gitPaths(ctx context.Context, sourceDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = sourceDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return nil, fmt.Errorf("jobs: git ls-files: %w: %s", err, detail)
		}
		return nil, fmt.Errorf("jobs: git ls-files: %w", err)
	}

	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, raw := range bytes.Split(stdout.Bytes(), []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		path := filepath.ToSlash(string(raw))
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func writeArchive(dst io.Writer, root string, paths []string) error {
	gz := gzip.NewWriter(dst)
	tw := tar.NewWriter(gz)

	var err error
	for _, rel := range paths {
		if err = addArchiveEntry(tw, root, rel); err != nil {
			break
		}
	}
	if closeErr := tw.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("jobs: finish archive: %w", closeErr)
	}
	if closeErr := gz.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("jobs: finish archive: %w", closeErr)
	}
	return err
}

func addArchiveEntry(tw *tar.Writer, root, rel string) error {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("jobs: invalid git path %q", rel)
	}

	full := filepath.Join(root, clean)
	info, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("jobs: stat %q: %w", rel, err)
	}
	if info.IsDir() {
		// A gitlink (submodule) is listed by ls-files but is a directory in the work tree.
		return nil
	}

	var link string
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		link, err = os.Readlink(full)
		if err != nil {
			return fmt.Errorf("jobs: read link %q: %w", rel, err)
		}
	case info.Mode().IsRegular():
	default:
		return nil
	}

	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return fmt.Errorf("jobs: tar header %q: %w", rel, err)
	}
	header.Name = filepath.ToSlash(rel)
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("jobs: write archive header %q: %w", rel, err)
	}
	if !info.Mode().IsRegular() {
		return nil
	}

	file, err := os.Open(full)
	if err != nil {
		return fmt.Errorf("jobs: open %q: %w", rel, err)
	}
	_, copyErr := io.Copy(tw, file)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("jobs: archive %q: %w", rel, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("jobs: close %q: %w", rel, closeErr)
	}
	return nil
}

func newArchiveName() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("jobs: generate archive name: %w", err)
	}
	return ".boat-push-" + hex.EncodeToString(random[:]) + ".tgz", nil
}
