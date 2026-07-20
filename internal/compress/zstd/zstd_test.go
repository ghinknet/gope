package zstd

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestCompressRoundTrip(t *testing.T) {
	input := []byte("hello zstd")
	var buf bytes.Buffer

	compressed, err := Compress(bytes.NewReader(input), &buf, 3)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if compressed != int64(buf.Len()) {
		t.Fatalf("compressed size: got %d, want %d", compressed, buf.Len())
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

func TestCompressReturnsWriterError(t *testing.T) {
	if _, err := Compress(bytes.NewReader([]byte("payload")), errorWriter{}, 3); err == nil {
		t.Fatal("expected writer error")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
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

func TestBatchDecompressRejectsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "archive.zst")
	if err := writeTarZstHeader(archivePath, &tar.Header{
		Name:     "link",
		Typeflag: tar.TypeSymlink,
		Linkname: "../outside",
		Mode:     0777,
	}); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	outDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := BatchDecompress(archivePath, outDir); err == nil {
		t.Fatal("expected symbolic link error")
	}
}

func TestBatchDecompressRejectsExistingSymlinkPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated privileges")
	}
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "archive.zst")
	if err := writeTarZst(archivePath, "link/escape.txt", []byte("no")); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	outDir := filepath.Join(tmpDir, "out")
	outsideDir := filepath.Join(tmpDir, "outside")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(outDir, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := BatchDecompress(archivePath, outDir); err == nil {
		t.Fatal("expected existing symbolic link error")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("archive escaped destination: %v", err)
	}
}

func writeTarZst(path string, name string, data []byte) error {
	return writeTarZstHeader(path, &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}, data...)
}

func writeTarZstHeader(path string, hdr *tar.Header, data ...byte) error {
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

	if err = tarWriter.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tarWriter.Write(data)
	return err
}
