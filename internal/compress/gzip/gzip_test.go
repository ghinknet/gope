package gzip

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
)

func TestCompressRoundTrip(t *testing.T) {
	input := []byte("hello gzip")
	var buf bytes.Buffer

	compressed, err := Compress(bytes.NewReader(input), &buf, 5)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if compressed != int64(buf.Len()) {
		t.Fatalf("compressed size: got %d, want %d", compressed, buf.Len())
	}

	reader, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}
	defer reader.Close()

	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !bytes.Equal(input, out) {
		t.Fatalf("round trip mismatch: got %q", string(out))
	}
}

func TestCompressReturnsWriterError(t *testing.T) {
	if _, err := Compress(bytes.NewReader([]byte("payload")), errorWriter{}, 5); err == nil {
		t.Fatal("expected writer error")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
