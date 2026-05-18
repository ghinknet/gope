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

