package meta

import "testing"

func TestSupportedMethodsNotEmpty(t *testing.T) {
	if len(SupportedMethods) == 0 {
		t.Fatalf("SupportedMethods is empty")
	}
}

func TestSupportedSystems(t *testing.T) {
	if !IsSupportedSystem("linux") {
		t.Fatal("linux should be supported")
	}
	if IsSupportedSystem("plan9") {
		t.Fatal("plan9 should be rejected")
	}
}
