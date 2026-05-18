package meta

// SupportedMethods lists available compression backends.
var SupportedMethods = []string{"zstd", "gzip"}

// SupportedPlatforms lists allowed GOOS/GOARCH targets.
var SupportedPlatforms = []string{
	// Darwin (from go tool dist list)
	"darwin/amd64", "darwin/arm64",
	// Linux (from go tool dist list)
	"linux/386", "linux/amd64",
	"linux/arm", "linux/arm64",
	"linux/loong64",
	"linux/mips", "linux/mips64",
	"linux/mips64le", "linux/mipsle",
	"linux/ppc64", "linux/ppc64le",
	"linux/riscv64",
	"linux/s390x",
}
