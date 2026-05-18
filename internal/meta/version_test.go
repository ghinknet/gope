package meta

import "testing"

func TestVersionTextNotEmpty(t *testing.T) {
	if VersionText == "" {
		t.Fatalf("VersionText is empty")
	}
}
