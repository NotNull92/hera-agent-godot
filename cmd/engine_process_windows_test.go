//go:build windows

package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestWindowsGuardOwnsJobObject pins the regression where LazyProc error
// handling prevented the Job Object from ever being retained.
func TestWindowsGuardOwnsJobObject(t *testing.T) {
	command := newEngineCommand(os.Args[0], []string{"-test.run=TestHelperProcessSleep"})
	command.Env = append(os.Environ(), "HERA_HELPER_SLEEP=1")
	if err := command.Start(); err != nil {
		t.Fatalf("helper failed to start: %v", err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	guard := newProcessGuard(command.Process.Pid, command.Process)
	defer guard.close()
	if guard.job == 0 {
		t.Fatal("a started process must be owned by a job object")
	}
	if guard.handle == 0 {
		t.Fatal("a started process must keep a retained handle")
	}
	if failure := guard.resume(); failure != "" {
		t.Fatalf("the suspended child must resume: %q", failure)
	}
	if !guardAlive(guard) {
		t.Fatal("the helper should be running when the guard is created")
	}
	if termination := guard.terminate(); termination != "" {
		t.Fatalf("terminating an owned job must be clean: %q", termination)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !guardAlive(guard) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the job termination did not stop the owned process")
}

func guardAlive(guard *processGuard) bool {
	status, err := syscall.WaitForSingleObject(guard.handle, 0)
	return err == nil && status == syscall.WAIT_TIMEOUT
}

// TestWindowsJobCapturesDescendant pins the ordering fix: the child is launched
// suspended, assigned to the job, then resumed, so a process it spawns while
// running belongs to the owned tree and dies with it. Without the suspended
// launch, a fast spawner can create the descendant before the assignment.
func TestWindowsJobCapturesDescendant(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	env := append(os.Environ(), "HERA_HELPER_SLEEP=1", "HERA_HELPER_SPAWN=1", "HERA_HELPER_PID_FILE="+pidFile)
	result := runEngineBounded(ctx, os.Args[0], []string{"-test.run=TestHelperProcessSleep"}, "", env, 4096)
	if !result.Started {
		t.Fatalf("helper should have started: %+v", result)
	}
	if result.Termination != "" {
		t.Fatalf("owned cleanup must be clean: %q", result.Termination)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the helper did not record a descendant pid: %v", err)
	}
	payload := strings.TrimSpace(string(data))
	if strings.HasPrefix(payload, "spawn-failed") {
		t.Fatalf("the helper could not spawn its descendant: %s", payload)
	}
	pid, err := strconv.Atoi(payload)
	if err != nil || pid <= 0 {
		t.Fatalf("invalid descendant pid %q", payload)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !windowsProcessAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("descendant %d survived the owned tree cleanup", pid)
}

// windowsProcessAlive reports whether a process id is still running. Only a
// confirmed absence counts as dead: an unobservable process is reported alive so
// the caller cannot pass the test by accident.
func windowsProcessAlive(pid int) bool {
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// ERROR_INVALID_PARAMETER means the process no longer exists.
		if errno, ok := err.(syscall.Errno); ok && errno == 87 {
			return false
		}
		return true
	}
	defer syscall.CloseHandle(handle)
	status, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return true
	}
	return status == syscall.WAIT_TIMEOUT
}

// TestWindowsCommandLaunchesSuspended pins the launch flag that makes verified
// job membership possible before the engine executes anything.
func TestWindowsCommandLaunchesSuspended(t *testing.T) {
	command := newEngineCommand("engine.exe", []string{"--help"})
	if command.SysProcAttr == nil || command.SysProcAttr.CreationFlags&createSuspended == 0 {
		t.Fatalf("Windows children must launch suspended: %+v", command.SysProcAttr)
	}
}

// TestWindowsResumeRequiresOwnership pins the invariant: a child whose job setup
// failed is never allowed to run.
func TestWindowsResumeRequiresOwnership(t *testing.T) {
	guard := &processGuard{pid: 4242}
	if failure := guard.resume(); failure == "" {
		t.Fatal("resuming without an owned job must fail")
	}
	if guard.resumed {
		t.Fatal("a refused resume must not mark the guard resumed")
	}
}

// TestWindowsGuardReportsUnconfiguredOwnership keeps the honest-failure path
// covered: a guard without a job must report incomplete cleanup instead of
// claiming a tree kill.
func TestWindowsGuardReportsUnconfiguredOwnership(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=TestHelperProcessSleep")
	command.Env = append(os.Environ(), "HERA_HELPER_SLEEP=1")
	if err := command.Start(); err != nil {
		t.Fatalf("helper failed to start: %v", err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	guard := &processGuard{pid: command.Process.Pid, process: command.Process}
	if termination := guard.terminate(); termination == "" {
		t.Fatal("cleanup without an owned job must be reported as incomplete")
	}
}
