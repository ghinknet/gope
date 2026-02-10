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

type Platform struct {
	GOOS   string
	GOARCH string
}

func main() {
	var namePrefix string
	var parallel int
	flag.StringVar(&namePrefix, "n", "GoPE", "Prefix of file")
	flag.IntVar(&parallel, "p", runtime.NumCPU()-1, "Number of parallel tasks (default: CPU count)")
	flag.Parse()

	if err := os.RemoveAll("dists"); err != nil {
		fmt.Printf("❌ Failed to remove dists: %v\n", err)
		return
	}

	if err := os.RemoveAll("decompressorSourceCode"); err != nil {
		fmt.Printf("❌ Failed to remove decompressorSourceCode: %v\n", err)
		return
	}

	cmd := exec.Command("go", "run", "./packer/main.go")
	if lines, err := cmd.Output(); err != nil {
		fmt.Printf("❌ Failed to run packer/main.go generate: %v\n", err)
		fmt.Printf("  %s\n", string(lines))
		return
	}

	if err := os.MkdirAll("dists", 0755); err != nil {
		fmt.Printf("❌ Failed to create dists: %v\n", err)
		return
	}

	os.Setenv("CGO_ENABLED", "0")

	platforms, err := getPlatforms()
	if err != nil {
		fmt.Printf("❌ Get support platforms: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)

	fmt.Printf("🚀 Start compile with prefix: %s\n", namePrefix)
	fmt.Printf("📦 Dest platforms: %d\n", len(platforms))
	fmt.Printf("⚡ Parallel tasks: %d\n", parallel)

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
	fmt.Printf("\n🎉 All compile tasks done! Output file under dists.\n")

	if err = os.RemoveAll("decompressorSourceCode"); err != nil {
		fmt.Printf("❌ Failed to remove decompressorSourceCode: %v\n", err)
		return
	}
}

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

func compilePlatform(platform Platform, namePrefix string) {
	suffix := ""
	if platform.GOOS == "windows" {
		suffix = ".exe"
	}

	outputFile := fmt.Sprintf("dists/%s-%s-%s%s", namePrefix, platform.GOOS, platform.GOARCH, suffix)

	fmt.Printf("🛠  Compiling: %s/%s\n", platform.GOOS, platform.GOARCH)

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
		fmt.Printf("❌ Failed: %s/%s - %v\n%s", platform.GOOS, platform.GOARCH, err, string(output))
	} else {
		fmt.Printf("✅ Done: %s/%s → %s\n", platform.GOOS, platform.GOARCH, outputFile)
	}
}
