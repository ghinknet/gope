package main

import "testing"

func TestGetPlatformsNotEmpty(t *testing.T) {
	platforms, err := getPlatforms()
	if err != nil {
		t.Fatalf("getPlatforms: %v", err)
	}
	if len(platforms) == 0 {
		t.Fatalf("expected at least one platform")
	}
}

