package files

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// WriteRequest is the body of PUT /sandboxes/{sandboxId}/files.
//
// Path is required. A relative path resolves against /home/user; the canonical path must stay under
// /home/user or /tmp. Missing parent folders are created. An empty Encoding means EncodingUTF8.
type WriteRequest struct {
	Path     string   `json:"path"`
	Content  string   `json:"content"`
	Encoding Encoding `json:"encoding,omitempty"`
}

// Write writes one file and returns its canonical path and size.
//
// Content is limited to 5 MiB after decoding. A larger file fails with 413 file_too_large; that
// error is terminal, and UploadURL is the alternative. A 200 means the file is on the sandbox disk.
func (s *Service) Write(ctx context.Context, sandboxID string, req WriteRequest, opts ...core.RequestOption) (*model.FileWriteResponse, error) {
	if err := requireID(sandboxID); err != nil {
		return nil, err
	}
	if err := requirePath(req.Path); err != nil {
		return nil, err
	}
	enc, err := normalizeEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}
	req.Encoding = enc

	var out model.FileWriteResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPut,
		Path:   sandboxPath(sandboxID, "files"),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
