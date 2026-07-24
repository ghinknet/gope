package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveOutputPathAbsolute(t *testing.T) {
	oldOutput, oldSystem := output, system
	t.Cleanup(func() { output, system = oldOutput, oldSystem })
	output = filepath.Join(string(filepath.Separator), "tmp", "gope-output")
	system = runtime.GOOS
	if got := resolveOutputPath(t.TempDir()); got != filepath.Clean(output) {
		t.Fatalf("output path: got %q, want %q", got, filepath.Clean(output))
	}
}

func TestResolveOutputPathAddsWindowsSuffixCaseInsensitively(t *testing.T) {
	oldOutput, oldSystem := output, system
	t.Cleanup(func() { output, system = oldOutput, oldSystem })
	system = "windows"
	output = "program.EXE"
	if got := resolveOutputPath("work"); got != filepath.Join("work", output) {
		t.Fatalf("output path: %q", got)
	}
	output = "program"
	if got := resolveOutputPath("work"); got != filepath.Join("work", "program.exe") {
		t.Fatalf("output path: %q", got)
	}
}

func TestMoveOutputCreatesParentDirectory(t *testing.T) {
	oldOutput, oldSystem, oldQuiet := output, system, quiet
	t.Cleanup(func() { output, system, quiet = oldOutput, oldSystem, oldQuiet })
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "output"), []byte("binary"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	output = filepath.Join(t.TempDir(), "nested", "program")
	system = runtime.GOOS
	quiet = true
	path, err := moveOutput(tempDir, t.TempDir())
	if err != nil {
		t.Fatalf("move output: %v", err)
	}
	if path != output {
		t.Fatalf("output path: got %q, want %q", path, output)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "binary" {
		t.Fatalf("output content: %q, %v", data, err)
	}
}

func TestRunCheckedRejectsNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	err := runChecked(nil, "/bin/sh", []string{"-c", "exit 7"}, t.TempDir(), true)
	if err == nil || !strings.Contains(err.Error(), "code 7") {
		t.Fatalf("runChecked error: %v", err)
	}
}

func TestReleaseEmbeddedCreatesStandaloneModule(t *testing.T) {
	dir := t.TempDir()
	if err := releaseEmbedded(dir); err != nil {
		t.Fatalf("release embedded: %v", err)
	}
	for _, name := range []string{"main.go", "go.mod", "go.sum", filepath.Join("internal", "runner", "run.go")} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod.template")); !os.IsNotExist(err) {
		t.Fatalf("go.mod.template should not be released: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "compressed")); !os.IsNotExist(err) {
		t.Fatalf("payload placeholder should not be released: %v", err)
	}
}

func TestReleasedDecompressorBuilds(t *testing.T) {
	for _, method := range []string{"zstd", "gzip"} {
		t.Run(method, func(t *testing.T) {
			dir := t.TempDir()
			if err := releaseEmbedded(dir); err != nil {
				t.Fatalf("release embedded: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "compressed"), nil, 0644); err != nil {
				t.Fatalf("write payload: %v", err)
			}
			cmd := exec.Command("go", "build", "-tags", method, "-o", filepath.Join(dir, "output"), ".")
			cmd.Dir = dir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("build released decompressor: %v\n%s", err, output)
			}
		})
	}
}

func TestValidateFlagsRejectsWindowsReplace(t *testing.T) {
	oldInput, oldMethod := input, method
	oldSystem, oldRunMode := system, runMode
	oldLevel, oldUpxLevel := level, upxLevel
	t.Cleanup(func() {
		input, method = oldInput, oldMethod
		system, runMode = oldSystem, oldRunMode
		level, upxLevel = oldLevel, oldUpxLevel
	})
	input, method = "input", "zstd"
	system, runMode = "windows", "replace"
	level, upxLevel = 3, 9
	if err := validateFlags(); err == nil {
		t.Fatal("expected windows replace validation error")
	}
}
