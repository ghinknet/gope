package meta

import "testing"

func TestSupportedMethodsNotEmpty(t *testing.T) {
	if len(SupportedMethods) == 0 {
		t.Fatalf("SupportedMethods is empty")
	}
}

func TestSupportedPlatformsNotEmpty(t *testing.T) {
	if len(SupportedPlatforms) == 0 {
		t.Fatalf("SupportedPlatforms is empty")
	}
}
