package meta

// Name is the CLI name.
const Name = "GoPE"

// Version is the version text. It is overridden at release build time via
// -ldflags "-X gope/internal/meta.Version=<semver>"; the default marks an
// unversioned local build.
var Version = "develop"
