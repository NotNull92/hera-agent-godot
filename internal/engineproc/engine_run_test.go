package engineproc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoundedOutputTruncates(t *testing.T) {
	output := newBoundedOutput(8)
	if _, err := output.Write([]byte("0123456789")); err != nil {
		t.Fatal(err)
	}
	if string(output.text) != "01234567" || !output.truncated {
		t.Fatalf("expected truncation: %q truncated=%v", output.text, output.truncated)
	}
	zero := &boundedOutput{}
	if _, err := zero.Write([]byte("kept")); err != nil {
		t.Fatal(err)
	}
	if string(zero.text) != "kept" {
		t.Fatalf("zero-value limit must keep output: %q", zero.text)
	}
}

// blockingGuard blocks termination so the test can prove the guard is not closed
// while a termination worker is still using it.
type blockingGuard struct {
	mu       sync.Mutex
	delegate Guard
	entered  chan struct{}
	release  chan struct{}
	closes   int
}

func (g *blockingGuard) Resume() string {
	if g.delegate != nil {
		return g.delegate.Resume()
	}
	return ""
}

func (g *blockingGuard) Terminate() string {
	close(g.entered)
	<-g.release
	return "termination blocked past the bound"
}

func (g *blockingGuard) Close() {
	g.mu.Lock()
	g.closes++
	g.mu.Unlock()
	if g.delegate != nil {
		g.delegate.Close()
	}
}

func (g *blockingGuard) closeCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.closes
}

// TestRunEngineBoundedDefersCloseToARunningTerminationWorker pins NEW-1: a caller
// that times out must not close the guard a termination worker is still using.
func TestRunEngineBoundedDefersCloseToARunningTerminationWorker(t *testing.T) {
	guard := &blockingGuard{entered: make(chan struct{}), release: make(chan struct{})}
	previousFactory := GuardFactory
	previousDelay := engineWaitDelay
	GuardFactory = func(pid int, process *os.Process) Guard {
		guard.delegate = newProcessGuard(pid, process)
		return guard
	}
	engineWaitDelay = 200 * time.Millisecond
	t.Cleanup(func() {
		GuardFactory = previousFactory
		engineWaitDelay = previousDelay
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SLEEP_MS=6000")
	result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	if !result.Abandoned {
		t.Fatalf("the caller should have returned with an unobserved exit: %+v", result)
	}
	select {
	case <-guard.entered:
	default:
		t.Fatal("termination was never attempted")
	}
	if closes := guard.closeCount(); closes != 0 {
		t.Fatalf("the guard was closed %d time(s) while termination is still running", closes)
	}
	// Releasing termination hands the close to the worker, exactly once.
	close(guard.release)
	deadline := time.Now().Add(5 * time.Second)
	for guard.closeCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if closes := guard.closeCount(); closes != 1 {
		t.Fatalf("the worker must close the guard exactly once, got %d", closes)
	}
}

// TestHelperProcessSleep is not a test: it is the child process used by the
// bounded-runner tests below.
func TestHelperProcessSleep(t *testing.T) {
	if os.Getenv("HERA_HELPER_SLEEP") != "1" {
		return
	}
	if os.Getenv("HERA_HELPER_SPAWN") == "1" {
		// Spawn exactly one long-lived descendant, record its pid, then exit
		// first. The descendant must not recurse and must still belong to Hera's
		// owned tree.
		child := exec.Command(os.Args[0], "-test.run=TestHelperProcessSleep")
		child.Env = append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SPAWN=0")
		if err := child.Start(); err == nil {
			if file := os.Getenv("HERA_HELPER_PID_FILE"); file != "" {
				_ = os.WriteFile(file, []byte(strconv.Itoa(child.Process.Pid)), 0o644)
			}
		} else if file := os.Getenv("HERA_HELPER_PID_FILE"); file != "" {
			_ = os.WriteFile(file, []byte("spawn-failed: "+err.Error()), 0o644)
		}
		os.Exit(0)
	}
	if millis := os.Getenv("HERA_HELPER_SLEEP_MS"); millis != "" {
		if parsed, err := strconv.Atoi(millis); err == nil && parsed > 0 {
			time.Sleep(time.Duration(parsed) * time.Millisecond)
			os.Exit(0)
		}
	}
	time.Sleep(60 * time.Second)
	os.Exit(0)
}

// fakeGuard injects ownership outcomes while delegating the real launch/reap
// behaviour: `resume` and `close` act on the process Hera started, so an injected
// failure simulates a real one instead of leaving a suspended child behind.
type fakeGuard struct {
	resumeResult    string
	terminateResult string
	delegate        Guard
	resumed         bool
	terminated      bool
}

func (g *fakeGuard) Resume() string {
	g.resumed = true
	if g.resumeResult != "" {
		return g.resumeResult
	}
	if g.delegate != nil {
		return g.delegate.Resume()
	}
	return ""
}

func (g *fakeGuard) Terminate() string {
	g.terminated = true
	return g.terminateResult
}

func (g *fakeGuard) Close() {
	if g.delegate != nil {
		g.delegate.Close()
	}
}

// installFakeGuard routes the runner through a fake guard for one test.
func installFakeGuard(t *testing.T, guard *fakeGuard) {
	t.Helper()
	previous := GuardFactory
	GuardFactory = func(pid int, process *os.Process) Guard {
		guard.delegate = newProcessGuard(pid, process)
		return guard
	}
	t.Cleanup(func() { GuardFactory = previous })
}

// TestRunEngineBoundedRefusesExecutionWithoutOwnership pins the invariant: when
// ownership cannot be established the child must not be left running and the
// command must fail.
func TestRunEngineBoundedRefusesExecutionWithoutOwnership(t *testing.T) {
	guard := &fakeGuard{resumeResult: "job ownership was not established"}
	installFakeGuard(t, guard)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SLEEP_MS=800")
	started := time.Now()
	result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	elapsed := time.Since(started)
	if !result.Started {
		t.Fatalf("the helper should have started: %+v", result)
	}
	if !guard.resumed || !guard.terminated {
		t.Fatalf("the refused launch must be terminated: %+v", guard)
	}
	if result.RunErr == nil || !strings.Contains(result.RunErr.Error(), "engine launch failed") ||
		!strings.Contains(result.RunErr.Error(), "job ownership was not established") {
		t.Fatalf("the refusal must be a launch error naming the ownership failure: %+v", result)
	}
	// A refused launch must report the ownership failure. A bounded termination
	// note is legitimate: on Unix nothing else can stop the child (the fake guard
	// does not), while Windows kills it through the owned job on close.
	if result.Termination != "" && !strings.Contains(result.Termination, "did not exit after termination") {
		t.Fatalf("an unexpected termination report: %q", result.Termination)
	}
	if elapsed > 8*time.Second {
		t.Fatalf("the refusal path was not bounded: %s", elapsed)
	}
}

// TestRunEngineBoundedWaitsForHealthyWorkloads pins the invariant the merged-state
// review caught: the termination bound must never apply to a run that is still
// working. A child that outlives the delay but finishes inside its own deadline
// must report its own exit code.
func TestRunEngineBoundedWaitsForHealthyWorkloads(t *testing.T) {
	installFakeGuard(t, &fakeGuard{})
	previousDelay := engineWaitDelay
	engineWaitDelay = 200 * time.Millisecond
	t.Cleanup(func() { engineWaitDelay = previousDelay })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SLEEP_MS=700")
	result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	if result.TimedOut || result.Abandoned {
		t.Fatalf("a healthy run must not be cut short: %+v", result)
	}
	if result.Duration < 600*time.Millisecond {
		t.Fatalf("the run returned before the child finished: %s", result.Duration)
	}
	if result.ExitCode != 0 {
		t.Fatalf("the child's own exit code must be reported: %+v", result)
	}
	if result.Termination != "" {
		t.Fatalf("a healthy run needs no termination: %+v", result)
	}
}

// TestRunEngineBoundedReportsIncompleteTermination pins the honest-failure path:
// a termination that cannot stop the process is reported instead of hanging the
// caller, and a child that outlives the bound is called out.
func TestRunEngineBoundedReportsIncompleteTermination(t *testing.T) {
	guard := &fakeGuard{terminateResult: "job termination failed; nothing could be terminated"}
	installFakeGuard(t, guard)
	// Shorten the bound instead of waiting for the production value.
	previousDelay := engineWaitDelay
	engineWaitDelay = 250 * time.Millisecond
	t.Cleanup(func() { engineWaitDelay = previousDelay })

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	// The child deliberately outlives the bound: termination is simulated as
	// unable to stop it, so nothing but the bound can unblock the caller.
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SLEEP_MS=6000")
	started := time.Now()
	result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	elapsed := time.Since(started)
	if !result.TimedOut || !result.Abandoned {
		t.Fatalf("the run should have timed out without observing the exit: %+v", result)
	}
	if result.Termination == "" || !strings.Contains(result.Termination, "nothing could be terminated") {
		t.Fatalf("incomplete termination must be reported: %+v", result)
	}
	if !strings.Contains(result.Termination, "exit was not observed within the wait bound") {
		t.Fatalf("an unobserved exit must be called out: %+v", result)
	}
	if result.ExitCode != -1 {
		t.Fatalf("a process that never exited must not report an exit code: %+v", result)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("a failed termination must not block the caller: %s", elapsed)
	}
}

func TestRunEngineBoundedRejectsExpiredContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := Run(ctx, filepath.Join(t.TempDir(), "does-not-exist.exe"), []string{"--help"}, "", nil, 1024)
	if result.Started {
		t.Fatal("an expired context must not start a process")
	}
	if !result.TimedOut || result.ExitCode != -1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunEngineBoundedTimeoutIsBounded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1")
	started := time.Now()
	result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	elapsed := time.Since(started)
	if !result.Started {
		t.Fatalf("helper should have started: %+v", result)
	}
	if !result.TimedOut {
		t.Fatalf("timeout was not reported: %+v", result)
	}
	if elapsed > 20*time.Second {
		t.Fatalf("timeout path took %s; cleanup is not bounded", elapsed)
	}
}

// TestEngineCommandHardening pins the property that keeps a hanging engine from
// blocking the caller forever: bounded pipe draining. Windows-specific launch
// flags are asserted in the Windows test file.
func TestEngineCommandHardening(t *testing.T) {
	command := newEngineCommand("engine.exe", []string{"--help"})
	if command.WaitDelay != engineWaitDelay || command.WaitDelay <= 0 {
		t.Fatalf("WaitDelay must bound pipe draining, got %s", command.WaitDelay)
	}
}

// lifecycleBlockingGuard injects native stalls without leaving helper processes alive.
type lifecycleBlockingGuard struct {
	delegate                 Guard
	refusal                  bool
	blockTerminate           bool
	entered, release, closed chan struct{}
}

func (g *lifecycleBlockingGuard) Resume() string {
	if g.refusal {
		return "injected ownership refusal"
	}
	return g.delegate.Resume()
}
func (g *lifecycleBlockingGuard) Terminate() string {
	if g.blockTerminate {
		close(g.entered)
		<-g.release
	}
	return g.delegate.Terminate()
}
func (g *lifecycleBlockingGuard) Close() {
	if !g.blockTerminate {
		close(g.entered)
		<-g.release
	}
	g.delegate.Close()
	close(g.closed)
}
func TestRunEngineBoundedNativeStalls(t *testing.T) {
	for _, name := range []string{"refusal termination", "natural close", "cancel close"} {
		t.Run(name, func(t *testing.T) {
			guard := &lifecycleBlockingGuard{refusal: name == "refusal termination", blockTerminate: name == "refusal termination", entered: make(chan struct{}), release: make(chan struct{}), closed: make(chan struct{})}
			oldFactory, oldDelay := GuardFactory, engineWaitDelay
			engineWaitDelay = 100 * time.Millisecond
			GuardFactory = func(pid int, process *os.Process) Guard {
				guard.delegate = newProcessGuard(pid, process)
				return guard
			}
			t.Cleanup(func() { GuardFactory, engineWaitDelay = oldFactory, oldDelay })
			timeout, sleep := 5*time.Second, "80"
			if name == "cancel close" {
				timeout, sleep = 200*time.Millisecond, "6000"
			}
			if guard.refusal {
				sleep = "6000"
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SLEEP_MS="+sleep)
			result := Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
			// Always unblock the worker, including on an assertion failure.
			close(guard.release)
			select {
			case <-guard.closed:
			case <-time.After(3 * time.Second):
				t.Fatal("worker did not finish cleanup")
			}
			if !result.CleanupPending {
				t.Fatalf("native stall hidden: %+v", result)
			}
			if result.Duration > 3*time.Second {
				t.Fatalf("native call blocked caller: %s", result.Duration)
			}
			if name == "natural close" && (result.Abandoned || result.ExitCode != 0) {
				t.Fatalf("lost observed exit: %+v", result)
			}
			if guard.refusal && (result.ExitCode != -1 || result.RunErr == nil) {
				t.Fatalf("lost refusal: %+v", result)
			}
		})
	}
}
