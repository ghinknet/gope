package constant

// SupportedMethods lists available compression backends.
var SupportedMethods = []string{"zstd", "gzip"}

// SupportedPlatforms lists allowed GOOS/GOARCH targets.
var SupportedPlatforms = []string{
	"linux/amd64", "windows/amd64", "darwin/amd64",
}
