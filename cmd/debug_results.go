package cmd

import (
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/dap"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"os"
	"sort"
	"time"
)

// readVariables returns the locals and members of a frame, capped for token
// economy. One retry is allowed only when the adapter rejected a variables or
// scopes read and the first attempt's paused state is certain (no resume, or a
// resume the adapter confirmed). Transport failures, unconfirmed resumes and
// protocol corruption are never retried, because a retry could repeat a mutation
// whose outcome is unknown.
func readVariables(client *dap.Client, frameIndex int, wait time.Duration, thread ...int) ([]map[string]any, bool, error) {
	entries, truncated, certain, err := collectVariables(client, frameIndex, wait, true, thread...)
	if err == nil {
		return entries, truncated, nil
	}
	if !certain || !isStaleReferenceRejection(err) {
		return nil, false, err
	}
	// Recovery must be read-only: no second implicit resume is allowed, so a
	// target that resumed in the meantime reports its own failure (not_paused).
	entries, truncated, _, retryErr := collectVariables(client, frameIndex, wait, false, thread...)
	if retryErr != nil {
		return nil, false, fmt.Errorf("recovery after %v failed: %w", err, retryErr)
	}
	return entries, truncated, nil
}

// isStaleReferenceRejection reports whether the adapter refused a variables or
// scopes read, which is the only failure a read-only retry may recover from.
func isStaleReferenceRejection(err error) bool {
	var protocolErr *dap.ProtocolError
	if !errors.As(err, &protocolErr) {
		return false
	}
	return protocolErr.Command == "variables" || protocolErr.Command == "scopes"
}

// collectVariables resolves the frame and reads every scope once.
func collectVariables(client *dap.Client, frameIndex int, wait time.Duration, allowResume bool, thread ...int) ([]map[string]any, bool, bool, error) {
	frameID, certain, err := pausedFrameFor(client, frameIndex, wait, allowResume, thread...)
	if err != nil {
		return nil, false, certain, err
	}
	scopes, err := client.Request("scopes", map[string]any{"frameId": frameID}, wait)
	if err != nil {
		return nil, false, certain, err
	}
	entries := []map[string]any{}
	for _, scope := range scopes.Map("body").List("scopes") {
		reference := scope.Int("variablesReference")
		if reference == 0 {
			continue
		}
		response, err := client.Request("variables", map[string]any{"variablesReference": reference}, wait)
		if err != nil {
			return nil, false, certain, err
		}
		for _, variable := range response.Map("body").List("variables") {
			if len(entries) >= maxDAPVariables {
				return entries, true, certain, nil
			}
			entries = append(entries, map[string]any{
				"scope": scope.String("name"), "name": variable.String("name"), "value": variable.String("value"),
			})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return fmt.Sprint(entries[i]["scope"]) < fmt.Sprint(entries[j]["scope"])
	})
	return entries, false, certain, nil
}

func debugActionFailure(result map[string]any, action string, err error) int {
	if errors.Is(err, errDebuggeeRunning) || errors.Is(err, dap.ErrNotObserved) {
		result["error"] = "not_paused"
		result["detail"] = fmt.Sprintf("%s needs a paused debuggee: %v", action, err)
		return debugFail(result, "not_paused", fmt.Sprint(result["detail"]))
	}
	return debugFail(result, "dap_request_failed", fmt.Sprintf("%s: %v", action, err))
}

// debugFail prints the structured failure, a stderr diagnostic, and returns 1.
func debugFail(result map[string]any, code, detail string) int {
	result["error"], result["detail"] = code, detail
	result["ok"] = false
	fmt.Fprintf(os.Stderr, "error: %s: %s\n", code, detail)
	if printed := printData(&protocol.Response{OK: true, Data: result}); printed != 0 {
		return printed
	}
	return 1
}

func finishDebugResult(result map[string]any) int {
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}
