package cmd

// version is the CLI version string. It defaults to "dev" for local builds and
// is overridden at release time via the linker:
//
//	go build -ldflags "-X github.com/NotNull92/hera-agent-godot/cmd.version=v0.1.0"
//
// The release workflow (.github/workflows/release.yml) injects the git tag.
var version = "dev"

// Version returns the CLI version string.
func Version() string { return version }
