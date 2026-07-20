package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gope/decompressor/internal/decompress"
	"gope/decompressor/internal/runner"
)

// Embedded compressed payload written by gope at pack time.
//
//go:embed compressed
var embeddedExecutable []byte

func main() {
	exitCode, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

type runMode int

const (
	modeMask runMode = iota
	modeReplace
)

// Build-time mode injected by gope at pack time (mask or replace).
var buildMode = "mask"

func run() (int, error) {
	// Pass through all arguments to the wrapped program.
	args := os.Args[1:]

	// Resolve run mode from build-time configuration.
	mode, err := parseRunMode(buildMode)
	if err != nil {
		return 0, err
	}
	if mode == modeReplace && runtime.GOOS == "windows" {
		return 0, fmt.Errorf("replace mode is not supported on windows")
	}

	exePath, exeDir, err := executablePath()
	if err != nil {
		return 0, err
	}
	workDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("error getting working directory: %w", err)
	}

	binaryPath, cleanup, err := prepareExecutable(exePath, exeDir, mode)
	if err != nil {
		return 0, err
	}
	defer cleanup()

	// Run executable
	code, err := runner.Run(binaryPath, args, workDir)
	if err != nil {
		return 0, fmt.Errorf("error running binary: %w", err)
	}

	return code, nil
}

func parseRunMode(value string) (runMode, error) {
	switch value {
	case "mask":
		return modeMask, nil
	case "replace":
		return modeReplace, nil
	default:
		return modeMask, fmt.Errorf("invalid build mode: %s", value)
	}
}

func executablePath() (string, string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("error getting executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", "", fmt.Errorf("error resolving executable path: %w", err)
	}
	return resolved, filepath.Dir(resolved), nil
}

// prepareExecutable materializes the real program using the chosen mode and
// returns a cleanup function that removes any file created for mask mode.
func prepareExecutable(exePath string, exeDir string, mode runMode) (string, func(), error) {
	if mode == modeReplace {
		binaryPath, err := decompressToPath(exeDir, exePath)
		return binaryPath, func() {}, err
	}

	// Mask mode: write the real program next to the wrapper so that programs
	// which locate themselves through os.Executable resolve to the packed
	// file's directory rather than a private temp directory.
	binaryPath, err := decompressBeside(exeDir)
	if err != nil {
		return "", func() {}, err
	}
	return binaryPath, func() { _ = os.Remove(binaryPath) }, nil
}

// decompressBeside writes the real executable into dir (the wrapper's own
// directory) so the running program shares the wrapper's location.
func decompressBeside(dir string) (binaryPath string, retErr error) {
	outFile, tempBinaryPath, err := createOutputExecutable(dir)
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			_ = os.Remove(tempBinaryPath)
		}
	}()
	if err = decompressExecutable(outFile); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err = os.Chmod(tempBinaryPath, 0755); err != nil {
			return "", fmt.Errorf("setting executable permission failed: %w", err)
		}
	}
	return tempBinaryPath, nil
}

// decompressToPath replaces the wrapper with the decompressed payload.
func decompressToPath(outputDir string, targetPath string) (binaryPath string, retErr error) {
	outFile, tempBinaryPath, err := createOutputExecutable(outputDir)
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			_ = os.Remove(tempBinaryPath)
		}
	}()
	if err = decompressExecutable(outFile); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err = os.Chmod(tempBinaryPath, 0755); err != nil {
			return "", fmt.Errorf("setting executable permission failed: %w", err)
		}
	}
	if err = os.Rename(tempBinaryPath, targetPath); err != nil {
		return "", fmt.Errorf("error replacing executable: %w", err)
	}
	return targetPath, nil
}

// decompressExecutable expands the embedded payload into the given file.
func decompressExecutable(outFile *os.File) error {
	_, decompressErr := decompress.Decompress(bytes.NewReader(embeddedExecutable), outFile)
	closeErr := outFile.Close()
	if err := errors.Join(decompressErr, closeErr); err != nil {
		return fmt.Errorf("error decompressing executable: %w", err)
	}
	return nil
}

// createOutputExecutable allocates the temp executable path.
func createOutputExecutable(outputDir string) (*os.File, string, error) {
	pattern := ".gope-exec-*"
	if runtime.GOOS == "windows" {
		pattern += ".exe"
	}
	outFile, err := os.CreateTemp(outputDir, pattern)
	if err != nil {
		return nil, "", fmt.Errorf("error creating executable file: %w", err)
	}
	return outFile, outFile.Name(), nil
}
