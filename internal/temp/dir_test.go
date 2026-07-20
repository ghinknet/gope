package temp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMvFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := MvFile(src, dst); err != nil {
		t.Fatalf("mv: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected src removed")
	}

	if data, err := os.ReadFile(dst); err != nil || string(data) != "data" {
		t.Fatalf("dst content mismatch: %v", err)
	}
}

func TestMvFileReplacesExistingDestination(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")
	if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}
	if err := MvFile(src, dst); err != nil {
		t.Fatalf("move: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("dst content: got %q", data)
	}
}

func TestMvFilePreservesDestinationOnCopyFailure(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source-directory")
	dst := filepath.Join(tmpDir, "dst")
	if err := os.Mkdir(src, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("original"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}
	if err := MvFile(src, dst); err == nil {
		t.Fatal("expected move error")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("destination was modified: %q", data)
	}
}
