package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"gope/internal/meta"
)

// Platform describes a GOOS/GOARCH build target.
type Platform struct {
	GOOS   string
	GOARCH string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[error] %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Flags and basic setup.
	var namePrefix string
	var version string
	var parallel int
	flag.StringVar(&namePrefix, "n", "GoPE", "Prefix of file")
	flag.StringVar(&version, "v", "", "Semantic version to inject and embed in artifact names (e.g. v0.0.1)")
	flag.IntVar(&parallel, "p", defaultParallelism(), "Number of parallel tasks")
	flag.Parse()
	if parallel < 1 {
		return fmt.Errorf("parallel task count must be at least 1")
	}

	platforms, err := getPlatforms()
	if err != nil {
		return fmt.Errorf("list platforms: %w", err)
	}

	if err = resetWorkspace(); err != nil {
		return err
	}

	if err = os.MkdirAll("dists", 0755); err != nil {
		return fmt.Errorf("create dists: %w", err)
	}

	fmt.Printf("[info] build prefix: %s\n", namePrefix)
	if version != "" {
		fmt.Printf("[info] version: %s\n", version)
	}
	fmt.Printf("[info] platforms: %d\n", len(platforms))
	fmt.Printf("[info] parallel: %d\n", parallel)

	// Resolve build targets and run the build in parallel.
	if err = compileAll(platforms, namePrefix, version, parallel); err != nil {
		return err
	}

	fmt.Printf("[info] build finished\n")
	return nil
}

func defaultParallelism() int {
	if runtime.NumCPU() <= 1 {
		return 1
	}
	return runtime.NumCPU() - 1
}

// resetWorkspace removes previous outputs.
func resetWorkspace() error {
	if err := os.RemoveAll("dists"); err != nil {
		return fmt.Errorf("failed to remove dists: %w", err)
	}
	return nil
}

// getPlatforms loads supported GOOS/GOARCH pairs from the Go toolchain.
func getPlatforms() ([]Platform, error) {
	cmd := exec.Command("go", "tool", "dist", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	var platforms []Platform
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		parts := strings.Split(line, "/")
		if len(parts) != 2 {
			continue
		}

		goos, goarch := parts[0], parts[1]

		if !meta.IsSupportedSystem(goos) {
			continue
		}

		platforms = append(platforms, Platform{GOOS: goos, GOARCH: goarch})
	}

	return platforms, nil
}

// compileAll builds all target platforms with a concurrency limit.
func compileAll(platforms []Platform, namePrefix string, version string, parallel int) error {
	return compileAllWith(platforms, namePrefix, parallel, func(p Platform, prefix string) error {
		return compilePlatform(p, prefix, version)
	})
}

func compileAllWith(platforms []Platform, namePrefix string, parallel int, compiler func(Platform, string) error) error {
	if parallel < 1 {
		return fmt.Errorf("parallel task count must be at least 1")
	}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)
	errorsChan := make(chan error, len(platforms))

	for _, platform := range platforms {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(p Platform) {
			defer wg.Done()
			defer func() { <-semaphore }()

			if err := compiler(p, namePrefix); err != nil {
				errorsChan <- err
			}
		}(platform)
	}

	wg.Wait()
	close(errorsChan)
	var buildErrors []error
	for err := range errorsChan {
		buildErrors = append(buildErrors, err)
	}
	return errors.Join(buildErrors...)
}

// compilePlatform builds a single target and writes to dists/.
func compilePlatform(platform Platform, namePrefix string, version string) error {
	suffix := ""
	if platform.GOOS == "windows" {
		suffix = ".exe"
	}

	// Artifact names embed the version when provided: GoPE-v0.0.1-linux-amd64.
	name := namePrefix
	if version != "" {
		name = fmt.Sprintf("%s-%s", namePrefix, version)
	}
	outputFile := fmt.Sprintf("dists/%s-%s-%s%s", name, platform.GOOS, platform.GOARCH, suffix)

	fmt.Printf("[info] build %s/%s\n", platform.GOOS, platform.GOARCH)

	ldflags := "-s -w"
	if version != "" {
		ldflags += " -X gope/internal/meta.Version=" + version
	}

	cmd := exec.Command("go", "build",
		"-ldflags="+ldflags,
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
		return fmt.Errorf("build %s/%s: %w\n%s", platform.GOOS, platform.GOARCH, err, string(output))
	}
	fmt.Printf("[ok] build %s/%s -> %s\n", platform.GOOS, platform.GOARCH, outputFile)
	return nil
}
