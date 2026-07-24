package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	exitCode, err := Run("/bin/sh", []string{"-c", "exit 9"}, t.TempDir())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if exitCode != 9 {
		t.Fatalf("exit code: got %d", exitCode)
	}
}

func TestRunWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	tmpDir := t.TempDir()
	marker := filepath.Join(tmpDir, "marker")
	_, err := Run("/bin/sh", []string{"-c", "pwd > marker"}, tmpDir)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err = os.Stat(marker); err != nil {
		t.Fatalf("missing marker: %v", err)
	}
}

func TestRunSignalExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh and Unix signals")
	}
	exitCode, err := Run("/bin/sh", []string{"-c", "kill -TERM $$"}, t.TempDir())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if exitCode != 143 {
		t.Fatalf("exit code: got %d, want 143", exitCode)
	}
}
