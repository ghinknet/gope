package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseRunMode(t *testing.T) {
	if mode, err := parseRunMode("mask"); err != nil || mode != modeMask {
		t.Fatalf("mask mode: %v, %v", mode, err)
	}
	if mode, err := parseRunMode("replace"); err != nil || mode != modeReplace {
		t.Fatalf("replace mode: %v, %v", mode, err)
	}
	if _, err := parseRunMode("unknown"); err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestPrepareExecutableMaskWritesBesideWrapper(t *testing.T) {
	oldPayload := embeddedExecutable
	embeddedExecutable = compressTestPayload(t, []byte("payload"))
	t.Cleanup(func() { embeddedExecutable = oldPayload })

	exeDir := t.TempDir()
	wrapper := filepath.Join(exeDir, "wrapper")

	binaryPath, cleanup, err := prepareExecutable(wrapper, exeDir, modeMask)
	if err != nil {
		t.Fatalf("prepare executable: %v", err)
	}
	t.Cleanup(cleanup)
	if filepath.Dir(binaryPath) != exeDir {
		t.Fatalf("binary path %q is not beside the wrapper directory %q", binaryPath, exeDir)
	}
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if !bytes.Equal(data, []byte("payload")) {
		t.Fatalf("payload: got %q", data)
	}
}

func TestDecompressFailureRemovesPartialFile(t *testing.T) {
	oldPayload := embeddedExecutable
	embeddedExecutable = []byte("not-zstd")
	t.Cleanup(func() { embeddedExecutable = oldPayload })

	tempDir := t.TempDir()
	if _, err := decompressBeside(tempDir); err == nil {
		t.Fatal("expected decompression error")
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial files remain: %v", entries)
	}
}

func TestRunPreservesCallerWorkingDirectory(t *testing.T) {
	if os.Getenv("GOPE_TEST_HELPER") == "1" {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(os.Getenv("GOPE_TEST_MARKER"), []byte(cwd), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	executableBytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}

	oldPayload, oldBuildMode := embeddedExecutable, buildMode
	oldArgs := os.Args
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		embeddedExecutable, buildMode = oldPayload, oldBuildMode
		os.Args = oldArgs
		_ = os.Chdir(oldDir)
	})

	embeddedExecutable = compressTestPayload(t, executableBytes)
	buildMode = "mask"
	callerDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "cwd")
	if err = os.Chdir(callerDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Setenv("GOPE_TEST_HELPER", "1")
	t.Setenv("GOPE_TEST_MARKER", marker)
	os.Args = []string{executable, "-test.run=TestRunPreservesCallerWorkingDirectory"}

	code, err := run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	resolvedCallerDir, err := filepath.EvalSymlinks(callerDir)
	if err != nil {
		t.Fatalf("resolve caller dir: %v", err)
	}
	if string(data) != resolvedCallerDir {
		t.Fatalf("child cwd: got %q, want %q", data, resolvedCallerDir)
	}
}
