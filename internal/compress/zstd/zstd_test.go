package zstd

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestCompressRoundTrip(t *testing.T) {
	input := []byte("hello zstd")
	var buf bytes.Buffer

	if _, err := Compress(bytes.NewReader(input), &buf, 3); err != nil {
		t.Fatalf("compress: %v", err)
	}

	decoder, err := zstd.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}
	defer decoder.Close()

	out, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !bytes.Equal(input, out) {
		t.Fatalf("round trip mismatch: got %q", string(out))
	}
}

func TestBatchDecompress(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "archive.zst")
	payloadPath := "go/file.txt"
	payload := []byte("payload")

	if err := writeTarZst(archivePath, payloadPath, payload); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	outDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := BatchDecompress(archivePath, outDir); err != nil {
		t.Fatalf("batch decompress: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, payloadPath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("payload mismatch: got %q", string(data))
	}
}

func TestBatchDecompressRejectsTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "archive.zst")

	if err := writeTarZst(archivePath, "../evil.txt", []byte("no")); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	outDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := BatchDecompress(archivePath, outDir); err == nil {
		t.Fatalf("expected error for traversal")
	}
}

func writeTarZst(path string, name string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder, err := zstd.NewWriter(file)
	if err != nil {
		return err
	}
	defer encoder.Close()

	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err = tarWriter.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tarWriter.Write(data)
	return err
}

