package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// Platform describes a GOOS/GOARCH build target.
type Platform struct {
	GOOS   string
	GOARCH string
}

func main() {
	// Flags and basic setup.
	var namePrefix string
	var parallel int
	flag.StringVar(&namePrefix, "n", "GoPE", "Prefix of file")
	flag.IntVar(&parallel, "p", runtime.NumCPU()-1, "Number of parallel tasks (default: CPU count)")
	flag.Parse()

	if err := resetWorkspace(); err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	if err := runPacker(); err != nil {
		fmt.Printf("[error] packer failed: %v\n", err)
		return
	}

	if err := os.MkdirAll("dists", 0755); err != nil {
		fmt.Printf("[error] create dists: %v\n", err)
		return
	}

	if err := os.Setenv("CGO_ENABLED", "0"); err != nil {
		fmt.Printf("[error] set env: %v\n", err)
		return
	}

	platforms, err := getPlatforms()
	if err != nil {
		fmt.Printf("[error] list platforms: %v\n", err)
		return
	}

	fmt.Printf("[info] build prefix: %s\n", namePrefix)
	fmt.Printf("[info] platforms: %d\n", len(platforms))
	fmt.Printf("[info] parallel: %d\n", parallel)

	// Resolve build targets and run the build in parallel.
	compileAll(platforms, namePrefix, parallel)

	fmt.Printf("[info] build finished\n")

	if err = os.RemoveAll("decomp_src"); err != nil {
		fmt.Printf("[warn] cleanup decomp_src: %v\n", err)
	}
}

// resetWorkspace removes previous outputs and embedded artefacts.
func resetWorkspace() error {
	if err := os.RemoveAll("dists"); err != nil {
		return fmt.Errorf("failed to remove dists: %w", err)
	}
	if err := os.RemoveAll("decomp_src"); err != nil {
		return fmt.Errorf("failed to remove decomp_src: %w", err)
	}
	return nil
}

// runPacker generates the embedded decompressor archive.
func runPacker() error {
	cmd := exec.Command("go", "run", "./packer/main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, string(output))
	}
	return nil
}

// getPlatforms loads supported GOOS/GOARCH pairs from the Go toolchain.
func getPlatforms() ([]Platform, error) {
	cmd := exec.Command("go", "tool", "dist", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var platforms []Platform
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		parts := strings.Split(line, "/")
		if len(parts) != 2 {
			continue
		}

		goos, goarch := parts[0], parts[1]

		if goos == "ios" || goos == "android" || goos == "js" || goos == "wasip1" || goos == "plan9" {
			continue
		}

		platforms = append(platforms, Platform{GOOS: goos, GOARCH: goarch})
	}

	return platforms, nil
}

// compileAll builds all target platforms with a concurrency limit.
func compileAll(platforms []Platform, namePrefix string, parallel int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)

	for _, platform := range platforms {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(p Platform) {
			defer wg.Done()
			defer func() { <-semaphore }()

			compilePlatform(p, namePrefix)
		}(platform)
	}

	wg.Wait()
}

// compilePlatform builds a single target and writes to dists/.
func compilePlatform(platform Platform, namePrefix string) {
	suffix := ""
	if platform.GOOS == "windows" {
		suffix = ".exe"
	}

	outputFile := fmt.Sprintf("dists/%s-%s-%s%s", namePrefix, platform.GOOS, platform.GOARCH, suffix)

	fmt.Printf("[info] build %s/%s\n", platform.GOOS, platform.GOARCH)

	cmd := exec.Command("go", "build",
		"-ldflags=-s -w",
		"-trimpath",
		"-o", outputFile,
		".")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", platform.GOOS),
		fmt.Sprintf("GOARCH=%s", platform.GOARCH),
		"CGO_ENABLED=0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[error] build %s/%s: %v\n%s", platform.GOOS, platform.GOARCH, err, string(output))
		return
	}
	fmt.Printf("[ok] build %s/%s -> %s\n", platform.GOOS, platform.GOARCH, outputFile)
}
