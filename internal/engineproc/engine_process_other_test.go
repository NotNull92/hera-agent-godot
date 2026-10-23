//go:build !windows

package engineproc

import "testing"

// TestEngineCommandDoesNotOwnProcessGroup pins the Unix decision: Hera signals
// only the started process, so it must not create a group it would then have to
// target by a recycled numeric id.
func TestEngineCommandDoesNotOwnProcessGroup(t *testing.T) {
	command := newEngineCommand("godot", []string{"--version"})
	if command.SysProcAttr != nil && command.SysProcAttr.Setpgid {
		t.Fatalf("Unix must not own a process group: %+v", command.SysProcAttr)
	}
}
