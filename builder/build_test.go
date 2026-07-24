package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"gope/internal/meta"
)

func TestGetPlatformsNotEmpty(t *testing.T) {
	platforms, err := getPlatforms()
	if err != nil {
		t.Fatalf("getPlatforms: %v", err)
	}
	if len(platforms) == 0 {
		t.Fatalf("expected at least one platform")
	}
	for _, platform := range platforms {
		if !meta.IsSupportedSystem(platform.GOOS) {
			t.Fatalf("unsupported platform returned: %s/%s", platform.GOOS, platform.GOARCH)
		}
	}
}

func TestDefaultParallelismIsPositive(t *testing.T) {
	if got := defaultParallelism(); got < 1 {
		t.Fatalf("default parallelism: got %d", got)
	}
}

func TestCompileAllRejectsInvalidParallelism(t *testing.T) {
	noop := func(Platform, string) error { return nil }
	if err := compileAllWith(nil, "GoPE", 0, noop); err == nil {
		t.Fatal("expected invalid parallelism error")
	}
}

func TestCompileAllAggregatesFailures(t *testing.T) {
	platforms := []Platform{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "windows", GOARCH: "amd64"}}
	var calls atomic.Int32
	expected := errors.New("compile failed")
	err := compileAllWith(platforms, "GoPE", 2, func(Platform, string) error {
		calls.Add(1)
		return expected
	})
	if calls.Load() != int32(len(platforms)) {
		t.Fatalf("compiler calls: got %d", calls.Load())
	}
	if err == nil || !strings.Contains(err.Error(), expected.Error()) {
		t.Fatalf("aggregate error: %v", err)
	}
}
