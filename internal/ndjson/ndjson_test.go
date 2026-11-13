package ndjson_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/internal/ndjson"
)

type frame struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func TestDecoder_Next_DecodesFrames(t *testing.T) {
	t.Parallel()

	dec := ndjson.NewDecoder[frame](strings.NewReader("{\"type\":\"stdout\",\"data\":\"a\"}\n\n{\"type\":\"stderr\",\"data\":\"b\"}\n{\"type\":\"exit\"}\n"))
	want := []frame{
		{Type: "stdout", Data: "a"},
		{Type: "stderr", Data: "b"},
		{Type: "exit"},
	}
	for i, wantFrame := range want {
		got, err := dec.Next()
		if err != nil {
			t.Fatalf("Next %d: %v", i, err)
		}
		if got != wantFrame {
			t.Fatalf("Next %d = %#v, want %#v", i, got, wantFrame)
		}
	}
	if _, err := dec.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after end = %v, want io.EOF", err)
	}
}

func TestDecoder_Next_EmptyStream(t *testing.T) {
	t.Parallel()

	dec := ndjson.NewDecoder[frame](strings.NewReader(""))
	if _, err := dec.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next = %v, want io.EOF", err)
	}
}

func TestDecoder_Next_MalformedFrame(t *testing.T) {
	t.Parallel()

	dec := ndjson.NewDecoder[frame](strings.NewReader("{\"type\":\"stdout\"}\nnot json\n"))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("first Next: %v", err)
	}
	_, err := dec.Next()
	if err == nil {
		t.Fatal("Next = nil, want decode error")
	}
	if !strings.Contains(err.Error(), "frame 2") {
		t.Fatalf("error = %v, want frame number", err)
	}
}

func TestDecoder_Next_TruncatedFrame(t *testing.T) {
	t.Parallel()

	dec := ndjson.NewDecoder[frame](strings.NewReader("{\"type\":\"stdout\"}\n{\"type\":"))
	if _, err := dec.Next(); err != nil {
		t.Fatalf("first Next: %v", err)
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("Next = nil, want decode error")
	}
}

type badReader struct{}

func (badReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestDecoder_Next_ReadError(t *testing.T) {
	t.Parallel()

	dec := ndjson.NewDecoder[frame](badReader{})
	if _, err := dec.Next(); err == nil {
		t.Fatal("Next = nil, want read error")
	}
}
