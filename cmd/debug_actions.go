package cmd

import (
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/dap"
	"math"
	"path/filepath"
	"sort"
	"time"
)

// stopState is what a command learned about the debuggee.
type stopState struct {
	ThreadID        int
	Stopped         bool
	Frames          []map[string]any
	Reason          string
	Running         bool
	ResumeConfirmed bool
}

// topFrame returns the first observed frame, if any.
func (s stopState) topFrame() map[string]any {
	if len(s.Frames) == 0 {
		return nil
	}
	return s.Frames[0]
}

const maxDebugThreadID = 1<<31 - 1

func debugThreadID(raw any) (int, error) {
	value, ok := raw.(float64)
	if !ok || value <= 0 || value > maxDebugThreadID || value != math.Trunc(value) {
		return 0, fmt.Errorf("the adapter returned an invalid thread id")
	}
	return int(value), nil
}

func debugThread(client *dap.Client, wait time.Duration, requested ...int) (int, error) {
	response, err := client.Request("threads", nil, wait)
	if err != nil {
		return 0, err
	}
	raw, ok := response.Map("body")["threads"].([]any)
	if !ok || len(raw) == 0 {
		return 0, fmt.Errorf("the adapter reported no usable threads")
	}
	ids := make(map[int]bool)
	for i, entry := range raw {
		object, ok := entry.(map[string]any)
		if !ok {
			return 0, fmt.Errorf("invalid thread entry %d", i)
		}
		id, err := debugThreadID(object["id"])
		if err != nil {
			return 0, err
		}
		if ids[id] {
			return 0, fmt.Errorf("duplicate thread id")
		}
		ids[id] = true
	}
	id := 0
	if len(requested) > 0 {
		id = requested[0]
	}
	if id == 0 {
		for _, event := range client.DrainEvents("stopped") {
			raw, present := event.Map("body")["threadId"]
			if !present {
				continue
			}
			candidate, err := debugThreadID(raw)
			if err != nil {
				return 0, err
			}
			if ids[candidate] {
				id = candidate
			}
		}
	}
	if id != 0 {
		if !ids[id] {
			return 0, fmt.Errorf("thread %d is not reported by the adapter", id)
		}
		return id, nil
	}
	if len(ids) == 1 {
		for id := range ids {
			return id, nil
		}
	}
	available := make([]int, 0, len(ids))
	for id := range ids {
		available = append(available, id)
	}
	sort.Ints(available)
	return 0, fmt.Errorf("multiple threads %v: select --thread ID", available)
}

// observeStop returns the current stop evidence without resuming anything. An
// empty frame list only counts as running when the adapter actually reported an
// empty list and no terminal event is pending.
func observeStop(client *dap.Client, wait time.Duration, thread ...int) (stopState, error) {
	if events := client.DrainEvents("terminated", "exited"); len(events) > 0 {
		return stopState{}, fmt.Errorf("the debuggee already terminated (%s)", events[0].String("event"))
	}
	id, err := debugThread(client, wait, thread...)
	if err != nil {
		return stopState{}, err
	}
	response, err := client.Request("stackTrace", map[string]any{"threadId": id}, wait)
	if err != nil {
		return stopState{}, err
	}
	body := response.Map("body")
	if body == nil {
		return stopState{}, fmt.Errorf("the adapter returned no stackTrace body")
	}
	raw, present := body["stackFrames"]
	if !present {
		return stopState{}, fmt.Errorf("the adapter returned a stackTrace without stackFrames")
	}
	frames, isArray := raw.([]any)
	if !isArray {
		return stopState{}, fmt.Errorf("the adapter returned stackFrames that are not an array")
	}
	if len(frames) == 0 {
		if events := client.DrainEvents("terminated", "exited"); len(events) > 0 {
			return stopState{}, fmt.Errorf("the debuggee terminated while observing it (%s)", events[0].String("event"))
		}
		return stopState{Running: true, ThreadID: id}, nil
	}
	observed := make([]map[string]any, 0, len(frames))
	for index, entry := range frames {
		object, isObject := entry.(map[string]any)
		if !isObject {
			return stopState{}, fmt.Errorf("the adapter returned frame %d that is not an object", index)
		}
		frame := dap.Message(object)
		if failure := validateFrameID(frame, index); failure != "" {
			return stopState{}, fmt.Errorf("%s", failure)
		}
		observed = append(observed, frameSummary(frame))
	}
	return stopState{Stopped: true, Frames: observed, ThreadID: id}, nil
}

// maxFrameID bounds a frame identifier: DAP ids are small integers, so a huge or
// fractional value is malformed evidence rather than a frame to address.
const maxFrameID = 1 << 30

// validateFrameID reports why a frame's id is unusable, or "" when it is a
// non-negative integer inside the supported range.
func validateFrameID(frame dap.Message, index int) string {
	raw, present := frame["id"]
	if !present {
		return fmt.Sprintf("the adapter returned frame %d without an id", index)
	}
	value, isNumber := raw.(float64)
	if !isNumber {
		return fmt.Sprintf("the adapter returned frame %d with a non-numeric id", index)
	}
	if value < 0 || value != math.Trunc(value) || value > maxFrameID {
		return fmt.Sprintf("the adapter returned frame %d with an out-of-range id", index)
	}
	return ""
}

func frameSummary(frame dap.Message) map[string]any {
	entry := map[string]any{"id": frame.Int("id"), "name": frame.String("name"), "line": frame.Int("line")}
	if source := frame.Map("source"); source != nil {
		path := filepath.ToSlash(source.String("path"))
		sourceEntry := map[string]any{"name": source.String("name"), "path": path, "sha256_available": false}
		for _, checksum := range source.List("checksums") {
			switch checksum.String("algorithm") {
			case "MD5":
				sourceEntry["md5"] = checksum.String("checksum")
			case "SHA256":
				sourceEntry["sha256"] = checksum.String("checksum")
				sourceEntry["sha256_available"] = true
			}
		}
		sourceEntry["checksums_source"] = "adapter_reported"
		entry["source"] = sourceEntry
	}
	return entry
}

// resumeAndObserve sends a resume/step request and returns the stop it caused.
// The action boundary is reconciled to idle first, so neither buffered nor
// unread events from earlier configuration can be reported as this action's
// result, and a resume the adapter never confirms is a failure rather than a
// success or an inferred running state.
func resumeAndObserve(client *dap.Client, command string, wait time.Duration, thread ...int) (stopState, error) {
	id, err := debugThread(client, wait, thread...)
	if err != nil {
		return stopState{}, err
	}
	// The idle window must fit inside the budget, so a small --wait-ms still runs.
	idle := actionIdle
	if budget := actionBudget(wait); idle > budget/4 {
		idle = budget / 4
	}
	_, terminal, err := client.ReconcileToIdle(idle, actionBudget(wait))
	if err != nil {
		return stopState{}, err
	}
	if terminal != nil {
		return stopState{}, fmt.Errorf("the debuggee already terminated (%s)", terminal.String("event"))
	}
	client.DrainEvents("stopped", "continued", "resumed")
	if _, err := client.Request(command, map[string]any{"threadId": id}, wait); err != nil {
		return stopState{}, err
	}
	deadline := time.Now().Add(wait)
	confirmed := false
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		event, err := client.NextEvent(remaining)
		if err != nil {
			if errors.Is(err, dap.ErrNotObserved) {
				break
			}
			return stopState{}, err
		}
		switch event.String("event") {
		case "continued", "resumed":
			body := event.Map("body")
			if body["allThreadsContinued"] != false || body.Int("threadId") == id {
				confirmed = true
			}
		case "stopped":
			if !confirmed {
				// A stop without a resume is not this action's result.
				return stopState{}, fmt.Errorf("the adapter reported a stop without confirming the resume, so the action's outcome is unknown")
			}
			state := stopState{Stopped: true, Reason: event.Map("body").String("reason"), ResumeConfirmed: true}
			stoppedThread := id
			if raw, present := event.Map("body")["threadId"]; present {
				var err error
				stoppedThread, err = debugThreadID(raw)
				if err != nil {
					return stopState{}, err
				}
			}
			observed, frameErr := observeStop(client, wait, stoppedThread)
			if frameErr != nil {
				return stopState{}, frameErr
			}
			if observed.Running {
				// The adapter reported a stop but no frame: contradictory evidence.
				return stopState{}, fmt.Errorf("the adapter reported a stop without any stack frame, so the stop is not usable")
			}
			state.Frames, state.ThreadID = observed.Frames, observed.ThreadID
			return state, nil
		case "exited", "terminated":
			return stopState{}, fmt.Errorf("the debuggee terminated during the action (%s)", event.String("event"))
		}
	}
	if confirmed {
		return stopState{Running: true, ResumeConfirmed: true, ThreadID: id}, nil
	}
	return stopState{}, fmt.Errorf("the adapter sent no resume evidence within %s, so the debuggee's state is unknown", wait)
}

// actionIdle is the quiet window an action wants before it starts, and
// actionBudget its total reconciliation budget.
const actionIdle = 250 * time.Millisecond

// actionBudget bounds reconciliation for one action. It is capped by the
// command's own wait, except for a 20 ms floor so the smallest accepted
// --wait-ms still reconciles (documented in docs/COMMANDS.md).
func actionBudget(wait time.Duration) time.Duration {
	budget := actionIdle * 4
	if wait < budget {
		budget = wait
	}
	if budget < 20*time.Millisecond {
		budget = 20 * time.Millisecond
	}
	return budget
}

// pausedFrameFor resolves a frame id from the current stop. It reports whether
// the paused state is certain: either no resume was needed, or the resume was
// confirmed by the adapter, so a recovery attempt cannot repeat an uncertain
// mutation.
func pausedFrameFor(client *dap.Client, frameIndex int, wait time.Duration, allowResume bool, thread ...int) (int, bool, error) {
	state, err := observeStop(client, wait, thread...)
	if err != nil {
		return 0, false, err
	}
	if state.Running {
		if !allowResume {
			return 0, false, errDebuggeeRunning
		}
		state, err = resumeAndObserve(client, "continue", wait, state.ThreadID)
		if err != nil {
			return 0, false, err
		}
		if !state.Stopped || !state.ResumeConfirmed {
			return 0, false, errDebuggeeRunning
		}
	}
	if frameIndex >= len(state.Frames) {
		return 0, false, fmt.Errorf("frame %d is outside the %d reported frame(s)", frameIndex, len(state.Frames))
	}
	id, ok := state.Frames[frameIndex]["id"].(int)
	if !ok {
		return 0, false, fmt.Errorf("the adapter reported frame %d without an id", frameIndex)
	}
	return id, true, nil
}

// errDebuggeeRunning means the debuggee did not stop, so no frame is available.
var errDebuggeeRunning = errors.New("the debuggee is running and did not reach a stop")
