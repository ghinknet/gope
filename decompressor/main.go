package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.gh.ink/gope/decompressor/internal/decompress"
	"go.gh.ink/gope/decompressor/internal/runner"
	"go.gh.ink/gope/decompressor/internal/temp"
)

//go:embed compressed
var embeddedExecutable []byte

// ReleaseEmbedded releases the embedded compressed executable
func releaseEmbedded(temp string) error {
	return os.WriteFile(
		filepath.Join(temp, "compressed"), embeddedExecutable, 0644,
	)
}

func main() {
	exitCode, err := run()
	if err != nil {
		fmt.Println(err)
		return
	}
	os.Exit(exitCode)
}

type runMode int

const (
	modeMask runMode = iota
	modeReplace
)

var buildMode = "mask"

func run() (int, error) {
	args := os.Args[1:]

	mode := modeMask
	if buildMode == "replace" {
		mode = modeReplace
	}
	if mode == modeReplace && runtime.GOOS == "windows" {
		mode = modeMask
	}

	exePath, workDir, err := executablePath()
	if err != nil {
		return 0, err
	}

	// Make temp dir
	mkTemp, err := temp.MkTemp()
	if err != nil {
		return 0, fmt.Errorf("error making temp dir: %w", err)
	}
	defer func(mkTemp temp.Dir) {
		if e := mkTemp.Release(); e != nil {
			panic(e)
		}
	}(mkTemp)

	binaryPath, cleanup, err := prepareExecutable(mkTemp.Path(), exePath, workDir, mode)
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

func prepareExecutable(tempPath string, exePath string, outputDir string, mode runMode) (string, func(), error) {
	if mode == modeReplace {
		binaryPath, err := decompressToPath(tempPath, outputDir, exePath)
		if err != nil {
			return "", func() {}, err
		}
		return binaryPath, func() {}, nil
	}

	binaryPath, err := decompressToTemp(tempPath, outputDir)
	if err != nil {
		return "", func() {}, err
	}
	return binaryPath, func() {
		_ = os.Remove(binaryPath)
	}, nil
}

func decompressToTemp(tempPath string, outputDir string) (string, error) {
	outFile, binaryPath, err := createOutputExecutable(outputDir)
	if err != nil {
		return "", err
	}
	if err = decompressExecutable(tempPath, outFile); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err = os.Chmod(binaryPath, 0755); err != nil {
			return "", fmt.Errorf("setting executable permission failed: %w", err)
		}
	}
	return binaryPath, nil
}

func decompressToPath(tempPath string, outputDir string, targetPath string) (string, error) {
	outFile, tempBinaryPath, err := createOutputExecutable(outputDir)
	if err != nil {
		return "", err
	}
	if err = decompressExecutable(tempPath, outFile); err != nil {
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

func decompressExecutable(tempPath string, outFile *os.File) error {
	if err := releaseEmbedded(tempPath); err != nil {
		return fmt.Errorf("error releasing executable: %w", err)
	}

	compressedBinaryPath := filepath.Join(tempPath, "compressed")
	inFile, err := os.Open(compressedBinaryPath)
	if err != nil {
		return fmt.Errorf("error opening compressed file: %w", err)
	}
	defer func(inFile *os.File) {
		if e := inFile.Close(); e != nil {
			panic(e)
		}
	}(inFile)

	defer func(outFile *os.File) {
		if e := outFile.Close(); e != nil {
			panic(e)
		}
	}(outFile)

	if _, err = decompress.Decompress(inFile, outFile); err != nil {
		return fmt.Errorf("error decompressing executable: %w", err)
	}

	return nil
}

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
