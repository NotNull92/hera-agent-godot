//go:build !windows

package cmd

import (
	"os"
	"os/exec"
)

// applyProcessGroup is a no-op on Unix: Hera deliberately does not create or
// signal a process group by numeric id.
func applyProcessGroup(_ *exec.Cmd) {}

// processGuard owns only the started process. Detaching a numeric id from a
// reaped child is impossible without pidfd-style handles: after the child is
// reaped, a process or group id can belong to an unrelated process, and a
// wrong-target kill is worse than incomplete cleanup. Windows (the platform
// whose console wrapper spawns a child engine) gets real tree ownership through
// a Job Object instead.
//
// shortcut: Unix descendants are therefore not terminated on timeout; upgrade to
// pidfd/cgroup supervision if a wrapper-with-descendants case needs it.
type processGuard struct {
	pid     int
	process *os.Process
}

func newProcessGuard(pid int, process *os.Process) *processGuard {
	return &processGuard{pid: pid, process: process}
}

// resume is a no-op on Unix, where the child is not launched suspended.
func (g *processGuard) resume() string { return "" }

// terminate stops the started process and reports any incomplete cleanup.
func (g *processGuard) terminate() string {
	if g.process == nil {
		return "no process handle; nothing could be terminated"
	}
	if err := g.process.Kill(); err != nil {
		return "kill failed: " + err.Error()
	}
	return "only the started process was terminated; descendants are not signalled on this platform"
}

func (g *processGuard) close() {}
