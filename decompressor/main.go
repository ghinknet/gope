package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ghinknet/gope/decompressor/internal/decompress"
	"github.com/ghinknet/gope/decompressor/internal/runner"
	"github.com/ghinknet/gope/decompressor/internal/temp"
	"github.com/ghinknet/toolbox/expr"
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
	// Get all flags
	args := os.Args[1:]

	// Get work dir
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting work dir:", err)
		return
	}

	// Make temp dir
	mkTemp, err := temp.MkTemp()
	if err != nil {
		fmt.Println("error making temp dir:", err)
		return
	}
	defer func(mkTemp temp.Dir) {
		if e := mkTemp.Release(); e != nil {
			panic(e)
		}
	}(mkTemp)

	// Construct executable path
	binaryPath := filepath.Join(
		mkTemp.Path(),
		expr.Ternary(runtime.GOOS == "windows", "compressed.exe", "compressed"),
	)

	func() {
		// Release compressed executable
		if err = releaseEmbedded(mkTemp.Path()); err != nil {
			fmt.Println("error releasing executable:", err)
			return
		}

		// Read compressed executable
		compressedBinaryPath := filepath.Join(mkTemp.Path(), "compressed")
		inFile, err := os.Open(compressedBinaryPath)
		if err != nil {
			fmt.Println("error opening compressed file:", err)
			return
		}
		defer func(inFile *os.File) {
			if e := inFile.Close(); e != nil {
				panic(e)
			}
		}(inFile)

		// Create executable
		outFile, err := os.Create(binaryPath)
		if err != nil {
			fmt.Println("error creating executable file:", err)
			return
		}
		defer func(outFile *os.File) {
			if e := outFile.Close(); e != nil {
				panic(e)
			}
		}(outFile)

		// Decompress executable
		_, err = decompress.Decompress(inFile, outFile)
		if err != nil {
			fmt.Println("error decompressing executable:", err)
			return
		}
	}()

	// Set executable permission
	if runtime.GOOS != "windows" {
		if err = os.Chmod(binaryPath, 0755); err != nil {
			fmt.Println("setting executable permission failed:", err)
			return
		}
	}

	// Run executable
	code, err := runner.Run(binaryPath, args, workDir)
	if err != nil {
		fmt.Println("error running binary:", err)
		return
	}

	os.Exit(code)
}
