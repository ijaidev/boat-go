package files

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ReadResult is one inline file read.
//
// Content is in the wire encoding reported by Encoding. Use Bytes to decode it.
type ReadResult struct {
	model.FileReadResponse
}

// Bytes returns Content decoded from its wire encoding.
//
// Base64 content is decoded; UTF-8 and an empty encoding return the raw bytes.
func (r *ReadResult) Bytes() ([]byte, error) {
	if r == nil {
		return nil, errors.New("files: nil read result")
	}
	if Encoding(r.Encoding) == EncodingBase64 {
		content, err := base64.StdEncoding.DecodeString(r.Content)
		if err != nil {
			return nil, fmt.Errorf("files: decode base64 content for %s: %w", r.Path, err)
		}
		return content, nil
	}
	return []byte(r.Content), nil
}

// Read returns one file's content.
//
// A relative path resolves against /home/user; the canonical path must stay under /home/user or
// /tmp, and anything else fails with 400 invalid_path. An empty encoding means EncodingUTF8.
//
// Files up to 5 MiB can be read inline. A larger file fails with 413 file_too_large; that error is
// terminal and is never retried. Use DownloadURL for large files and stream the signed URL yourself.
func (s *Service) Read(ctx context.Context, sandboxID, path string, encoding Encoding, opts ...core.RequestOption) (*ReadResult, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(path); err != nil {
		return nil, err
	}
	enc, err := normalizeEncoding(encoding)
	if err != nil {
		return nil, err
	}

	var out model.FileReadResponse
	err = s.do(ctx, &core.Request{
		Method: http.MethodGet,
		Path:   sandboxPath(sandboxID, "files"),
		Query: url.Values{
			"path":     {path},
			"encoding": {string(enc)},
		},
	}, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &ReadResult{FileReadResponse: out}, nil
}
