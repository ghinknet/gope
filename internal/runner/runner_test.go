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
	exitCode, err := Run(nil, "/bin/sh", []string{"-c", "exit 7"}, t.TempDir(), true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if exitCode != 7 {
		t.Fatalf("exit code: got %d", exitCode)
	}
}

func TestRunWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	tmpDir := t.TempDir()
	marker := filepath.Join(tmpDir, "marker")
	_, err := Run(nil, "/bin/sh", []string{"-c", "pwd > marker"}, tmpDir, true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err = os.Stat(marker); err != nil {
		t.Fatalf("missing marker: %v", err)
	}
}

