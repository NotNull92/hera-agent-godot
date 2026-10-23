package cmd

import (
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"os"
	"os/exec"
)

// runResultFields is the shared status/output block of an engine-backed result.
// Commands add their own artifact fields on top.
func runResultFields(run engineproc.Result) map[string]any {
	fields := map[string]any{
		"exit_code": run.ExitCode, "timed_out": run.TimedOut,
		"duration_ms": run.Duration.Milliseconds(),
		"output":      run.Output, "output_truncated": run.Truncated,
	}
	if run.Termination != "" {
		fields["termination_error"] = run.Termination
	}
	if errors.Is(run.RunErr, exec.ErrWaitDelay) {
		// The process exited but something kept its pipes open, so the captured
		// output may be incomplete.
		fields["output_incomplete"] = true
	}
	if run.CleanupPending {
		fields["cleanup_pending"] = true
	}
	if run.Abandoned {
		fields["abandoned"] = true
	}
	return fields
}

// mergeResult copies shared run fields into a command result.
func mergeResult(result map[string]any, run engineproc.Result) map[string]any {
	for key, value := range runResultFields(run) {
		result[key] = value
	}
	return result
}

// runFailureCode classifies an engine run that must not be reported as success.
func runFailureCode(run engineproc.Result) string {
	switch {
	case run.Abandoned:
		// The exit status was never observed, so no work can be claimed.
		return "engine_abandoned"
	case run.TimedOut:
		return "engine_timeout"
	case run.CleanupPending:
		return "engine_cleanup_incomplete"
	case errors.Is(run.RunErr, exec.ErrWaitDelay):
		return "output_incomplete"
	case run.ExitCode > 0:
		return "engine_exit_nonzero"
	case run.ExitCode == -1 && run.RunErr != nil:
		return run.RunErr.Error()
	}
	return ""
}

// runFailed reports whether a run failed, including incomplete output.
func runFailed(run engineproc.Result) bool {
	return runFailureCode(run) != ""
}

// resultError returns the recorded failure code of a command result.
func resultError(result map[string]any) string {
	code, _ := result["error"].(string)
	return code
}

// failEngineResult prints the result with the stderr diagnostic the stream
// contract requires and returns 1.
func failEngineResult(result map[string]any) int {
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	reason := resultError(result)
	if reason == "" {
		reason = "failed"
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", reason)
	return 1
}

// finishEngineResult applies the run verdict to a command result, prints it
// once, and returns the command exit code.
func finishEngineResult(result map[string]any, run engineproc.Result) int {
	if reason := runFailureCode(run); reason != "" && resultError(result) == "" {
		result["error"] = reason
	}
	if resultError(result) != "" {
		return failEngineResult(result)
	}
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}
