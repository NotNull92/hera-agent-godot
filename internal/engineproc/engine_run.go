package engineproc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const engineOutputLimit = 64 * 1024

// engineWaitDelay bounds both the inherited-pipe wait after the started process
// exits and Hera's own wait for a process it asked to terminate, so a surviving
// descendant or a termination that failed cannot block the caller. Tests shorten
// it instead of sleeping for the production value.
var engineWaitDelay = 5 * time.Second

// newEngineCommand builds the engine process with its hardening applied:
// bounded pipe draining and platform-specific launch ownership.
func newEngineCommand(executable string, args []string) *exec.Cmd {
	command := exec.Command(executable, args...)
	command.WaitDelay = engineWaitDelay
	applyProcessGroup(command)
	return command
}

// boundedOutput keeps at most limit bytes and records truncation. Writes and
// snapshots are mutex-guarded, so a bounded wait that abandons a live engine can
// still read what was drained without racing the pipe copier.
type boundedOutput struct {
	mu        sync.Mutex
	text      []byte
	truncated bool
	limit     int
}

func newBoundedOutput(limit int) *boundedOutput {
	if limit <= 0 {
		limit = engineOutputLimit
	}
	return &boundedOutput{limit: limit}
}

func (o *boundedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.limit <= 0 {
		o.limit = engineOutputLimit
	}
	remaining := o.limit - len(o.text)
	if len(p) > remaining {
		if remaining > 0 {
			o.text = append(o.text, p[:remaining]...)
		}
		o.truncated = true
	} else {
		o.text = append(o.text, p...)
	}
	return len(p), nil
}

// snapshot returns the output collected so far and whether it was truncated.
func (o *boundedOutput) snapshot() (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return string(o.text), o.truncated
}

// Guard is the ownership contract Run relies on. Tests
// replace GuardFactory to inject setup, resume and termination failures
// deterministically.
type Guard interface {
	Resume() string
	Terminate() string
	Close()
}

// GuardFactory creates the guard for a started process.
var GuardFactory = func(pid int, process *os.Process) Guard {
	return newProcessGuard(pid, process)
}

// Result is one bounded engine process run.
type Result struct {
	Started   bool
	ExitCode  int
	Output    string
	Truncated bool
	TimedOut  bool
	// Abandoned reports that the wait for the process was not observed to finish
	// (including its pipe drain) inside the bound after cancellation, so nothing
	// about its work or its exit status is proven.
	Abandoned      bool
	CleanupPending bool
	Duration       time.Duration
	RunErr         error
	Termination    string
}

// Run runs the engine with bounded output. Ownership of the
// process is established at launch:
//
//   - Windows: the child is assigned to an owned Job Object (kill-on-close)
//     immediately after Start, so the tree can be terminated through a handle
//     even when the started process already exited;
//   - Unix: only the started process is signalled, through Go's own Process
//     bookkeeping, which refuses to signal after the child was reaped. Unix
//     descendants are deliberately not signalled by numeric id (see the
//     platform file);
//   - the timeout path records incomplete cleanup instead of reporting success,
//     and pipe draining is bounded so a surviving descendant cannot block the
//     caller forever.
//
// An already-canceled context starts nothing.
func Run(ctx context.Context, executable string, args []string, dir string, env []string, limit int) Result {
	started := time.Now()
	result := Result{ExitCode: -1}
	if ctx.Err() != nil {
		// Never launch work for a deadline that already passed.
		result.TimedOut = true
		result.RunErr = errors.New("context already expired before launch")
		result.Duration = time.Since(started)
		return result
	}
	output := newBoundedOutput(limit)
	command := newEngineCommand(executable, args)
	command.Dir = dir
	if len(env) > 0 {
		command.Env = env
	}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		result.RunErr = err
		result.Duration = time.Since(started)
		return result
	}
	result.Started = true
	guard := GuardFactory(command.Process.Pid, command.Process)
	// One worker owns termination and close, including refusal. The caller never
	// touches the guard again, even if a native call outlives its wait bound.
	failure := guard.Resume()
	stopped := make(chan struct{})
	cleanup := make(chan string, 1)
	go func() {
		termination := ""
		if failure != "" {
			termination = guard.Terminate()
		} else {
			select {
			case <-ctx.Done():
				termination = guard.Terminate()
			case <-stopped:
			}
		}
		if failure == "" {
			<-stopped
		}
		guard.Close()
		cleanup <- termination
	}()

	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	var waitErr error
	exitObserved := false
	if failure == "" {
		select {
		case waitErr = <-waited:
			exitObserved = true
		case <-ctx.Done():
		}
	}
	if !exitObserved {
		select {
		case waitErr = <-waited:
			exitObserved = true
		case <-time.After(engineWaitDelay):
		}
	}
	close(stopped)
	select {
	case result.Termination = <-cleanup:
	case <-time.After(engineWaitDelay):
		result.CleanupPending = true
		result.Termination = "termination or guard close did not report back within the wait bound"
	}
	if !exitObserved {
		result.Termination = strings.TrimSpace(result.Termination + "; the process exit was not observed within the wait bound")
	}
	if exitObserved && failure == "" && command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	result.TimedOut = ctx.Err() != nil
	result.Abandoned = !exitObserved
	result.RunErr = waitErr
	if failure != "" {
		result.RunErr = errors.New("engine launch failed: " + failure)
	}
	text, truncated := output.snapshot()
	result.Output = strings.TrimSpace(text)
	result.Truncated = truncated
	result.Duration = time.Since(started)
	return result
}
