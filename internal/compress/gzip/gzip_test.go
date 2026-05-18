package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestCompressRoundTrip(t *testing.T) {
	input := []byte("hello gzip")
	var buf bytes.Buffer

	if _, err := Compress(bytes.NewReader(input), &buf, 5); err != nil {
		t.Fatalf("compress: %v", err)
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

