// Command hera-agent-godot is a low-token CLI that lets AI agents inspect and
// control a live Godot 4.x editor over localhost HTTP.
//
// See docs/ARCHITECTURE.md for the design and docs/COMMANDS.md for the surface.
package main

import (
	"os"

	"github.com/NotNull92/hera-agent-godot/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
