// Package ndjson decodes newline-delimited JSON streams without buffering the whole body.
package ndjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const maxFrameSize = 4 << 20 // 4 MiB

// Decoder reads one JSON object per line.
type Decoder[T any] struct {
	scanner *bufio.Scanner
	line    int
}

// NewDecoder returns a decoder reading from r.
func NewDecoder[T any](r io.Reader) *Decoder[T] {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), maxFrameSize)
	return &Decoder[T]{scanner: scanner}
}

// Next returns the next frame. It skips empty lines and returns io.EOF at the end.
func (d *Decoder[T]) Next() (T, error) {
	var zero T
	for d.scanner.Scan() {
		d.line++
		line := d.scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var frame T
		if err := json.Unmarshal(line, &frame); err != nil {
			return zero, fmt.Errorf("ndjson: decode frame %d: %w", d.line, err)
		}
		return frame, nil
	}
	if err := d.scanner.Err(); err != nil {
		return zero, fmt.Errorf("ndjson: read: %w", err)
	}
	return zero, io.EOF
}
