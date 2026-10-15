//go:build windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

// applyProcessGroup launches the child suspended so tree ownership can be
// established before it executes anything.
func applyProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createSuspended}
}

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procCreateJob  = kernel32.NewProc("CreateJobObjectW")
	procSetJobInfo = kernel32.NewProc("SetInformationJobObject")
	procAssignJob  = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJ = kernel32.NewProc("TerminateJobObject")
	procOpenProc   = kernel32.NewProc("OpenProcess")
	procSnapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procThread32F  = kernel32.NewProc("Thread32First")
	procThread32N  = kernel32.NewProc("Thread32Next")
	procOpenThread = kernel32.NewProc("OpenThread")
	procResumeThr  = kernel32.NewProc("ResumeThread")
	procCloseHnd   = kernel32.NewProc("CloseHandle")
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x2000
	processSetQuota                   = 0x0100
	processTerminate                  = 0x0001
	createSuspended                   = 0x00000004
	th32csSnapThread                  = 0x00000004
	threadSuspendResume               = 0x0002
	invalidHandle                     = ^uintptr(0)
	resumeThreadFailure               = 0xFFFFFFFF
)

// threadEntry32 mirrors THREADENTRY32.
type threadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePriority   int32
	DeltaPriority  int32
	Flags          uint32
}

// resume executes a suspended child once ownership is established. A child whose
// job setup failed is never allowed to run: the invariant is verified membership
// before execution, not merely an attempt at membership.
func (g *processGuard) resume() string {
	if g.resumed {
		return ""
	}
	if g.pid <= 0 {
		return "no process id to resume"
	}
	if g.job == 0 {
		return "job ownership was not established, so the engine was not allowed to run"
	}
	if failure := resumePrimaryThread(uint32(g.pid)); failure != "" {
		return failure
	}
	g.resumed = true
	return ""
}

// resumePrimaryThread resumes the suspended initial thread of pid. CREATE_SUSPENDED
// suspends exactly that thread, so finding one is enough.
func resumePrimaryThread(pid uint32) string {
	snapshot, _, snapErr := procSnapshot.Call(th32csSnapThread, 0)
	if snapshot == invalidHandle {
		return winCall(0, snapErr, "CreateToolhelp32Snapshot")
	}
	defer procCloseHnd.Call(snapshot)
	entry := threadEntry32{Size: uint32(unsafe.Sizeof(threadEntry32{}))}
	ok, _, firstErr := procThread32F.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return winCall(ok, firstErr, "Thread32First")
	}
	for {
		if entry.OwnerProcessID == pid {
			thread, _, openErr := procOpenThread.Call(threadSuspendResume, 0, uintptr(entry.ThreadID))
			if failure := winCall(thread, openErr, "OpenThread"); failure != "" {
				return failure
			}
			resumed, _, resumeErr := procResumeThr.Call(thread)
			procCloseHnd.Call(thread)
			// ResumeThread returns the thread's previous suspend count as a
			// 32-bit value; 0xFFFFFFFF means failure and anything above one
			// leaves the thread suspended.
			switch previous := uint32(resumed); {
			case previous == resumeThreadFailure:
				return winCall(0, resumeErr, "ResumeThread")
			case previous != 1:
				return fmt.Sprintf("ResumeThread reported a previous suspend count of %d; the engine would stay suspended", previous)
			}
			return ""
		}
		entry = threadEntry32{Size: uint32(unsafe.Sizeof(threadEntry32{}))}
		next, _, _ := procThread32N.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if next == 0 {
			break
		}
	}
	return "no thread found for the started process"
}

// jobObjectExtendedLimit mirrors JOBOBJECT_EXTENDED_LIMIT_INFORMATION so the
// native call receives the complete, correctly sized structure.
type jobObjectExtendedLimit struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	_                       uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	_                       uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
	// IO_COUNTERS
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
	// Memory limits that complete the native structure.
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// processGuard owns a started process through a Job Object (kill-on-close) plus
// a retained process handle. Terminating the job reaches the whole tree even
// when the started process already exited, and neither handle is affected by PID
// reuse.
type processGuard struct {
	pid     int
	job     syscall.Handle
	handle  syscall.Handle
	process *os.Process
	resumed bool
	once    sync.Once
}

// winCall reports the failure of a Win32 call. The primary return value decides
// success: syscall's LazyProc.Call always returns a non-nil error value, so the
// error is only meaningful once the primary result says the call failed.
func winCall(primary uintptr, err error, what string) string {
	if primary != 0 {
		return ""
	}
	if err != nil && err != syscall.Errno(0) {
		return what + ": " + err.Error()
	}
	return what + " failed"
}

func newProcessGuard(pid int, process *os.Process) *processGuard {
	guard := &processGuard{pid: pid, process: process}
	if handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_TERMINATE, false, uint32(pid)); err == nil {
		guard.handle = handle
	}
	job, _, jobErr := procCreateJob.Call(0, 0)
	if failure := winCall(job, jobErr, "CreateJobObject"); failure != "" {
		return guard
	}
	limits := jobObjectExtendedLimit{LimitFlags: jobObjectLimitKillOnJobClose}
	configured, _, configErr := procSetJobInfo.Call(job, jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if failure := winCall(configured, configErr, "SetInformationJobObject"); failure != "" {
		// Never claim ownership that was not configured.
		syscall.CloseHandle(syscall.Handle(job))
		return guard
	}
	child, _, openErr := procOpenProc.Call(processSetQuota|processTerminate, 0, uintptr(pid))
	if failure := winCall(child, openErr, "OpenProcess for job assignment"); failure != "" {
		syscall.CloseHandle(syscall.Handle(job))
		return guard
	}
	assigned, _, assignErr := procAssignJob.Call(job, child)
	syscall.CloseHandle(syscall.Handle(child))
	if failure := winCall(assigned, assignErr, "AssignProcessToJobObject"); failure != "" {
		syscall.CloseHandle(syscall.Handle(job))
		return guard
	}
	guard.job = syscall.Handle(job)
	return guard
}

func (g *processGuard) alive() bool {
	if g.handle == 0 {
		return false
	}
	status, err := syscall.WaitForSingleObject(g.handle, 0)
	if err != nil {
		return false
	}
	return status == syscall.WAIT_TIMEOUT
}

// terminate stops the owned tree and reports any incomplete cleanup. It does not
// depend on the started process still being alive.
func (g *processGuard) terminate() string {
	if g.job != 0 {
		stopped, _, err := procTerminateJ.Call(uintptr(g.job), 1)
		if failure := winCall(stopped, err, "TerminateJobObject"); failure == "" {
			return ""
		}
		return g.fallback("job termination failed")
	}
	return g.fallback("no owned job for the started process")
}

// fallback stops only the started process, through Go's own Process bookkeeping,
// and reports that cleanup was incomplete. It never reacquires a target from a
// numeric PID.
func (g *processGuard) fallback(reason string) string {
	if g.process != nil {
		if err := g.process.Kill(); err != nil {
			return reason + "; direct kill failed: " + err.Error()
		}
		return reason + "; only the started process was terminated"
	}
	if g.handle != 0 {
		if err := syscall.TerminateProcess(g.handle, 1); err != nil {
			return reason + "; TerminateProcess failed: " + err.Error()
		}
		return reason + "; only the started process was terminated"
	}
	return reason + "; nothing could be terminated"
}

func (g *processGuard) close() {
	g.once.Do(func() {
		if g.job != 0 {
			syscall.CloseHandle(g.job)
			g.job = 0
		}
		if g.handle != 0 {
			syscall.CloseHandle(g.handle)
			g.handle = 0
		}
	})
}
