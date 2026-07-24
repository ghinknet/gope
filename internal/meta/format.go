package meta

import "slices"

// SupportedMethods lists available compression backends.
var SupportedMethods = []string{"zstd", "gzip"}

// UnsupportedSystems are Go targets that do not provide the process model
// required by the generated executable wrapper.
var UnsupportedSystems = []string{"android", "ios", "js", "plan9", "wasip1"}

func IsSupportedSystem(goos string) bool {
	return !slices.Contains(UnsupportedSystems, goos)
}
