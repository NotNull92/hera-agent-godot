package cmd

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/dap"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const (
	defaultDAPPort    = 6006
	defaultDAPTimeout = 10 * time.Second
	defaultStopWaitMS = 10000
	pauseWait         = 5 * time.Second
	actionWait        = 5 * time.Second
	maxWaitMS         = 3600000
	maxDAPVariables   = 50
)

// debugSessionIDPattern keeps a session identifier to one path component.
var debugSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// debugSession is the endpoint recipe one attach records so later calls can
// reconnect. It is not evidence that the original debuggee is still running:
// reconnects address whatever now answers on that host and port.
type debugSession struct {
	ID          string              `json:"id"`
	Host        string              `json:"host"`
	Port        int                 `json:"port"`
	Project     string              `json:"project"`
	Breakpoints []sessionBreakpoint `json:"breakpoints"`
	AttachedAt  string              `json:"attached_at"`
}

type sessionBreakpoint struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// dapSessionRoot returns the private session directory. A missing or unusable
// home fails instead of silently sharing a predictable temporary namespace.
func dapSessionRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("no user home directory for debug sessions: %v", err)
	}
	return filepath.Join(home, ".hera-agent-godot", "debug"), nil
}

// validSessionID rejects separators, traversal and anything that is not a single
// safe path component.
func validSessionID(id string) bool {
	if !debugSessionIDPattern.MatchString(id) || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`)
}

func writeDebugSession(session debugSession) error {
	root, err := dapSessionRoot()
	if err != nil {
		return err
	}
	if !validSessionID(session.ID) {
		return fmt.Errorf("refusing to persist an invalid session id %q", session.ID)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	recordPath := filepath.Join(root, session.ID+".json")
	if err := writeFileAtomic(recordPath, data); err != nil {
		return err
	}
	// The pointer is a convenience, not part of the record's durability.
	if err := writeFileAtomic(filepath.Join(root, "latest"), []byte(session.ID)); err != nil {
		return fmt.Errorf("session record written but the latest pointer failed: %w", err)
	}
	return nil
}

// loadDebugSession resolves an id (explicit, or the latest pointer) and validates
// both the id and the record before anything else happens.
func loadDebugSession(id string) (debugSession, string, error) {
	root, err := dapSessionRoot()
	if err != nil {
		return debugSession{}, "", err
	}
	if id == "" || id == "latest" {
		data, readErr := os.ReadFile(filepath.Join(root, "latest"))
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				return debugSession{}, "", fmt.Errorf("no debug session recorded; run debug attach first")
			}
			return debugSession{}, "", fmt.Errorf("cannot read the latest session pointer: %w", readErr)
		}
		id = strings.TrimSpace(string(data))
	}
	if !validSessionID(id) {
		return debugSession{}, "", fmt.Errorf("invalid session id %q", id)
	}
	recordPath := filepath.Join(root, id+".json")
	raw, readErr := os.ReadFile(recordPath)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return debugSession{}, "", fmt.Errorf("debug session %q not found", id)
		}
		return debugSession{}, "", fmt.Errorf("cannot read debug session %q: %w", id, readErr)
	}
	var session debugSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return debugSession{}, "", fmt.Errorf("debug session %q is unreadable: %w", id, err)
	}
	if session.ID != id || !validSessionID(session.ID) {
		return debugSession{}, "", fmt.Errorf("debug session %q does not describe itself correctly", id)
	}
	if session.Port <= 0 || session.Port > 65535 || session.Host == "" {
		return debugSession{}, "", fmt.Errorf("debug session %q has no usable endpoint", id)
	}
	return session, recordPath, nil
}

// dapAddress joins host and port, including IPv6 hosts.
func dapAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// connectDebug attaches to the recorded endpoint, configures breakpoints once and
// completes configuration once. Adapter rejections surface as errors.
func connectDebug(session debugSession, wait time.Duration) (*dap.Client, []map[string]any, error) {
	client, err := dap.Dial(dapAddress(session.Host, session.Port), wait)
	if err != nil {
		return nil, nil, err
	}
	if _, err := client.Request("initialize", map[string]any{
		"clientID": "hera-godot", "adapterID": "godot", "pathFormat": "path",
		"linesStartAt1": true, "columnsStartAt1": true,
	}, wait); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("initialize: %w", err)
	}
	if _, err := client.Request("attach", map[string]any{"type": "godot", "request": "attach", "project": session.Project}, wait); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("attach: %w", err)
	}
	results, err := applyBreakpoints(client, session.Breakpoints, wait)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	return client, results, nil
}

// applyBreakpoints sends every source's breakpoints and then completes
// configuration exactly once for the connection.
func applyBreakpoints(client *dap.Client, breakpoints []sessionBreakpoint, wait time.Duration) ([]map[string]any, error) {
	byPath := map[string][]int{}
	order := []string{}
	for _, breakpoint := range breakpoints {
		if _, seen := byPath[breakpoint.Path]; !seen {
			order = append(order, breakpoint.Path)
		}
		byPath[breakpoint.Path] = append(byPath[breakpoint.Path], breakpoint.Line)
	}
	results := []map[string]any{}
	for _, path := range order {
		lines := byPath[path]
		entries := make([]map[string]any, 0, len(lines))
		for _, line := range lines {
			entries = append(entries, map[string]any{"line": line})
		}
		source := map[string]any{"name": filepath.Base(path), "path": path}
		checksums, hashErr := fileChecksums(path)
		if hashErr == nil {
			source["checksums"] = checksums
		}
		response, err := client.Request("setBreakpoints", map[string]any{
			"source": source, "breakpoints": entries, "lines": lines,
		}, wait)
		if err != nil {
			return nil, fmt.Errorf("setBreakpoints %s: %w", path, err)
		}
		// Reconcile: every requested breakpoint must be accounted for.
		reported := response.Map("body").List("breakpoints")
		for index, line := range lines {
			entry := map[string]any{
				"path": filepath.ToSlash(path), "line": line, "verified": false,
				"checksums_source": "local_disk",
			}
			if hashErr != nil {
				entry["checksums_available"] = false
			}
			if index < len(reported) {
				entry["line"] = reported[index].Int("line")
				entry["verified"] = reported[index].Bool("verified")
			} else {
				entry["unavailable"] = true
			}
			results = append(results, entry)
		}
	}
	if _, err := client.Request("configurationDone", nil, wait); err != nil {
		return nil, fmt.Errorf("configurationDone: %w", err)
	}
	return results, nil
}

// fileChecksums returns the DAP checksum entries for a local script. These are
// local disk hashes sent to the adapter; they do not prove which revision the
// running game loaded.
func fileChecksums(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	md5sum := md5.Sum(data)
	sha := sha256.Sum256(data)
	return []map[string]any{
		{"algorithm": "MD5", "checksum": hex.EncodeToString(md5sum[:])},
		{"algorithm": "SHA256", "checksum": hex.EncodeToString(sha[:])},
	}, nil
}

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

func argsChanged(params map[string]any, key string) bool {
	value, ok := params[key+"_set"].(bool)
	return ok && value
}

func parseDebugArgs(args []string, allowBreakpoints bool) (map[string]any, error) {
	return parseDebugArgsWithExpression(args, allowBreakpoints, false)
}

// parseDebugArgsWithExpression parses shared debug flags. Only debug evaluate
// accepts the `--` expression terminator.
func parseDebugArgsWithExpression(args []string, allowBreakpoints bool, allowExpression bool) (map[string]any, error) {
	params := map[string]any{"host": "127.0.0.1", "port": defaultDAPPort, "wait_ms": defaultStopWaitMS, "frame": 0}
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--":
			if !allowExpression {
				return nil, fmt.Errorf("-- is only accepted by debug evaluate")
			}
			if _, seen := params["expression"]; seen {
				return nil, fmt.Errorf("only one expression is accepted")
			}
			rest := args[i+1:]
			if len(rest) != 1 {
				return nil, fmt.Errorf("-- must be followed by exactly one expression")
			}
			params["expression"] = rest[0]
			return params, nil
		case args[i] == "--break" || strings.HasPrefix(args[i], "--break="):
			if !allowBreakpoints {
				return nil, fmt.Errorf("--break is only accepted by debug attach")
			}
			value, next, err := takeFlagValue(args, i, "--break")
			if err != nil {
				return nil, err
			}
			path, line, err := parseBreakpoint(value)
			if err != nil {
				return nil, err
			}
			existing, _ := params["breaks"].([]sessionBreakpoint)
			params["breaks"] = append(existing, sessionBreakpoint{Path: path, Line: line})
			i = next - 1
		case isFlag(args[i], "--session"):
			value, next, err := takeFlagValue(args, i, "--session")
			if err != nil {
				return nil, err
			}
			if !validSessionID(value) {
				return nil, fmt.Errorf("--session must be a session id from debug session")
			}
			params["session"] = value
			i = next - 1
		case isFlag(args[i], "--host"):
			value, next, err := takeFlagValue(args, i, "--host")
			if err != nil {
				return nil, err
			}
			if value == "" {
				return nil, fmt.Errorf("--host requires a value")
			}
			params["host"], params["host_set"] = value, true
			i = next - 1
		case isFlag(args[i], "--dap-port"):
			value, next, err := takeFlagValue(args, i, "--dap-port")
			if err != nil {
				return nil, err
			}
			port, convErr := strconv.Atoi(value)
			if convErr != nil || port <= 0 || port > 65535 {
				return nil, fmt.Errorf("--dap-port must be a port number")
			}
			params["port"], params["port_set"] = port, true
			i = next - 1
		case isFlag(args[i], "--wait-ms"):
			value, next, err := takeFlagValue(args, i, "--wait-ms")
			if err != nil {
				return nil, err
			}
			millis, convErr := strconv.Atoi(value)
			if convErr != nil || millis <= 0 || millis > maxWaitMS {
				return nil, fmt.Errorf("--wait-ms must be between 1 and %d", maxWaitMS)
			}
			params["wait_ms"], params["wait_ms_set"] = millis, true
			i = next - 1
		case isFlag(args[i], "--thread"):
			value, next, err := takeFlagValue(args, i, "--thread")
			if err != nil {
				return nil, err
			}
			id, err := strconv.Atoi(value)
			if err != nil || id <= 0 || id > maxDebugThreadID {
				return nil, fmt.Errorf("--thread must be a positive thread id")
			}
			params["thread"], params["thread_set"] = id, true
			i = next - 1
		case isFlag(args[i], "--frame"):
			value, next, err := takeFlagValue(args, i, "--frame")
			if err != nil {
				return nil, err
			}
			frame, convErr := strconv.Atoi(value)
			if convErr != nil || frame < 0 {
				return nil, fmt.Errorf("--frame must be a non-negative integer")
			}
			params["frame"], params["frame_set"] = frame, true
			i = next - 1
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

// parseBreakpoint accepts res://script.gd:LINE and refuses escaping paths.
func parseBreakpoint(value string) (string, int, error) {
	index := strings.LastIndex(value, ":")
	if index <= 0 {
		return "", 0, fmt.Errorf("--break must look like res://script.gd:12")
	}
	path, lineText := value[:index], value[index+1:]
	line, err := strconv.Atoi(lineText)
	if err != nil || line <= 0 {
		return "", 0, fmt.Errorf("--break line must be a positive integer")
	}
	if !strings.HasPrefix(path, "res://") {
		return "", 0, fmt.Errorf("--break path must start with res://")
	}
	trimmed := strings.TrimPrefix(path, "res://")
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "..") {
		return "", 0, fmt.Errorf("--break path must stay inside the project")
	}
	return path, line, nil
}

// resolveBreakpointPath maps a res:// path into the project, refusing escapes
// after normalisation and after resolving links, and requires a readable script.
func resolveBreakpointPath(project, scriptPath string) (string, error) {
	if !strings.HasPrefix(scriptPath, "res://") {
		return scriptPath, nil
	}
	if project == "" {
		return "", fmt.Errorf("a res:// breakpoint needs --project so the script can be hashed")
	}
	resolved := filepath.Join(project, filepath.FromSlash(strings.TrimPrefix(scriptPath, "res://")))
	relative, err := filepath.Rel(project, resolved)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("breakpoint %s escapes the project directory", scriptPath)
	}
	// A link inside the project must not point outside it.
	realProject, projectErr := filepath.EvalSymlinks(project)
	realScript, scriptErr := filepath.EvalSymlinks(resolved)
	if projectErr == nil && scriptErr == nil {
		realRelative, relErr := filepath.Rel(realProject, realScript)
		if relErr != nil || strings.HasPrefix(realRelative, "..") {
			return "", fmt.Errorf("breakpoint %s resolves outside the project directory", scriptPath)
		}
	}
	if _, readErr := os.ReadFile(resolved); readErr != nil {
		return "", fmt.Errorf("breakpoint script %s is not readable: %w", scriptPath, readErr)
	}
	return resolved, nil
}

func sessionWait(params map[string]any) time.Duration {
	if millis, ok := params["wait_ms"].(int); ok && millis > 0 {
		return time.Duration(millis) * time.Millisecond
	}
	return defaultStopWaitMS * time.Millisecond
}

func runDebug(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: debug attach|session|stack|variables|evaluate|continue|next|step-in|step-out|detach ...")
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "attach":
		params, err := parseDebugArgs(rest, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug attach: %v\n", err)
			return 2
		}
		if _, ok := params["session"]; ok {
			fmt.Fprintln(os.Stderr, "debug attach: --session is not used by attach; a new session is recorded")
			return 2
		}
		if argsChanged(params, "frame") {
			fmt.Fprintln(os.Stderr, "debug attach: --frame is not used by attach")
			return 2
		}
		return runDebugAttach(params)
	case "session":
		params, err := parseDebugArgs(rest, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug session: %v\n", err)
			return 2
		}
		if _, ok := params["project"]; ok {
			fmt.Fprintln(os.Stderr, "debug session: --project is not used by session")
			return 2
		}
		if argsChanged(params, "frame") {
			fmt.Fprintln(os.Stderr, "debug session: --frame is not used by session")
			return 2
		}
		for _, unused := range []string{"host", "port", "wait_ms", "thread"} {
			if argsChanged(params, unused) {
				fmt.Fprintf(os.Stderr, "debug session: --%s is not used by session\n", strings.ReplaceAll(unused, "_", "-"))
				return 2
			}
		}
		return runDebugSession(params)
	case "stack", "variables", "continue", "next", "step-in", "step-out", "detach":
		params, err := parseDebugArgs(rest, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug %s: %v\n", sub, err)
			return 2
		}
		params["action"] = sub
		return runDebugAction(params)
	case "evaluate":
		expression := ""
		flagArgs := rest
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			expression, flagArgs = rest[0], rest[1:]
		}
		params, err := parseDebugArgsWithExpression(flagArgs, false, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug evaluate: %v\n", err)
			return 2
		}
		if explicit, ok := params["expression"].(string); ok {
			if expression != "" {
				fmt.Fprintln(os.Stderr, "debug evaluate: only one expression is accepted")
				return 2
			}
			expression = explicit
		}
		if expression == "" {
			fmt.Fprintln(os.Stderr, "usage: debug evaluate <expression> [--session ID] [--frame N]  (use -- for expressions that start with -)")
			return 2
		}
		params["action"], params["expression"] = "evaluate", expression
		return runDebugAction(params)
	default:
		fmt.Fprintf(os.Stderr, "unknown debug subcommand %q\n", sub)
		return 2
	}
}

// runDebugAttach records the session and reports the observed state.
func runDebugAttach(params map[string]any) int {
	if breaks, ok := params["breaks"].([]sessionBreakpoint); ok {
		for _, breakpoint := range breaks {
			if strings.EqualFold(filepath.Ext(breakpoint.Path), ".cs") {
				failStructured(map[string]any{
					"error":    "csharp_debug_unsupported",
					"detail":   "Godot's DAP endpoint does not debug C#; use Visual Studio, Rider, or VS Code with a .NET debugger",
					"docs_url": "https://docs.godotengine.org/en/stable/tutorials/scripting/c_sharp/c_sharp_basics.html#configuring-an-external-editor",
					"guidance": []string{
						"Visual Studio: open the project's solution and use its Godot C# debug target",
						"Rider: open the project and use its Godot run/debug configuration",
						"VS Code: configure a .NET launch/attach target as described in the linked Godot C# documentation",
					},
				})
				return 2
			}
		}
	}
	wait := sessionWait(params)
	project := stringParam(params, "project")
	if project != "" {
		resolved, err := resolveProjectDir(project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug attach: %v\n", err)
			return 1
		}
		project = resolved
	}
	session := debugSession{
		ID:         fmt.Sprintf("%d-%d", os.Getpid(), time.Now().Unix()),
		Host:       fmt.Sprint(params["host"]),
		Port:       intValue(params, "port", defaultDAPPort),
		Project:    project,
		AttachedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if breaks, ok := params["breaks"].([]sessionBreakpoint); ok {
		for _, breakpoint := range breaks {
			resolved, err := resolveBreakpointPath(project, breakpoint.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "debug attach: %v\n", err)
				return 1
			}
			session.Breakpoints = append(session.Breakpoints, sessionBreakpoint{Path: resolved, Line: breakpoint.Line})
		}
	}
	client, verified, err := connectDebug(session, wait)
	if err != nil {
		fmt.Fprintf(os.Stderr, "debug attach: %v\n", err)
		return 1
	}
	defer client.Close()
	result := map[string]any{
		"session_id": session.ID, "host": session.Host, "port": session.Port,
		"project": project, "breakpoints": session.Breakpoints, "source": "dap",
		"breakpoint_results": verified,
		"notes": []string{
			"debug attaches to an already running editor debug server; the editor and game are started by you",
			"checksums are local disk hashes sent with setBreakpoints; source.sha256 in a stop is adapter-reported and does not prove which revision the running game loaded",
			"a session record is an endpoint recipe: reconnecting addresses whatever now answers on that host and port",
		},
		"limits": map[string]any{
			"host_override":  argsChanged(params, "host"),
			"port_override":  argsChanged(params, "port"),
			"identity_check": "none: Hera cannot prove the endpoint still serves the original project",
		},
	}
	unverified := []map[string]any{}
	for _, entry := range verified {
		if entry["verified"] != true {
			unverified = append(unverified, entry)
		}
	}
	if len(unverified) > 0 {
		result["unverified_breakpoints"] = unverified
		result["notes"] = append(result["notes"].([]string),
			"an unverified breakpoint means the adapter could not bind it to executable code; check the line number")
	}
	// Observe before acting: an already-paused debuggee is reported as paused.
	state, observeErr := observeStop(client, wait, intValue(params, "thread", 0))
	switch {
	case observeErr != nil:
		return debugFail(result, "dap_observe_failed", observeErr.Error())
	case state.Stopped:
		result["stopped"] = map[string]any{"thread_id": state.ThreadID, "frame": state.topFrame(), "frame_count": len(state.Frames), "observed": "already_paused",
			"attribution": "not_applicable: the debuggee was paused before Hera asked for a resume"}
	default:
		stopState, resumeErr := resumeAndObserve(client, "continue", wait, state.ThreadID)
		switch {
		case resumeErr != nil:
			return debugFail(result, "dap_resume_failed", resumeErr.Error())
		case stopState.Stopped:
			result["stopped"] = map[string]any{"thread_id": stopState.ThreadID, "frame": stopState.topFrame(), "frame_count": len(stopState.Frames),
				"reason": stopState.Reason, "observed": "after_arming", "resume_confirmed": stopState.ResumeConfirmed,
				"attribution": "unproven: the stop followed an observed resume event, but the adapter provides no causal request boundary"}
		default:
			result["running"] = true
			result["notes"] = append(result["notes"].([]string),
				"the debuggee is running; breakpoints are armed, so the next stop happens when one is reached")
		}
	}
	if err := writeDebugSession(session); err != nil {
		return debugFail(result, "session_store_failed", err.Error())
	}
	return finishDebugResult(result)
}

func runDebugSession(params map[string]any) int {
	session, _, err := loadDebugSession(stringParam(params, "session"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "debug session: %v\n", err)
		return 1
	}
	return finishDebugResult(map[string]any{
		"session_id": session.ID, "host": session.Host, "port": session.Port,
		"project": session.Project, "breakpoints": session.Breakpoints,
		"attached_at": session.AttachedAt, "source": "dap",
		"notes": []string{"a session record is an endpoint recipe, not proof that the original debuggee is still running"},
	})
}

// runDebugAction reconnects to the recorded endpoint and performs one request.
func runDebugAction(params map[string]any) int {
	session, recordPath, err := loadDebugSession(stringParam(params, "session"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "debug %v: %v\n", params["action"], err)
		return 1
	}
	if argsChanged(params, "host") {
		session.Host = fmt.Sprint(params["host"])
	}
	if argsChanged(params, "port") {
		session.Port = intValue(params, "port", defaultDAPPort)
	}
	wait := sessionWait(params)
	action := fmt.Sprint(params["action"])
	if _, ok := params["project"]; ok {
		fmt.Fprintln(os.Stderr, "debug "+action+": --project is only accepted by debug attach")
		return 2
	}
	if action == "stack" && argsChanged(params, "frame") {
		fmt.Fprintln(os.Stderr, "debug stack: --frame is not used by stack; the top frame is reported")
		return 2
	}
	if action != "variables" && action != "evaluate" && argsChanged(params, "frame") {
		fmt.Fprintf(os.Stderr, "debug %s: --frame is only used by variables and evaluate\n", action)
		return 2
	}
	if action == "detach" && argsChanged(params, "thread") {
		return debugFail(map[string]any{}, "invalid_argument", "--thread is not used by detach")
	}
	client, _, err := connectDebug(session, wait)
	if err != nil {
		return debugFail(map[string]any{"session_id": session.ID, "action": action, "source": "dap"},
			"dap_unavailable", err.Error())
	}
	defer client.Close()
	result := map[string]any{"session_id": session.ID, "action": action, "source": "dap"}
	if argsChanged(params, "host") || argsChanged(params, "port") {
		result["limits"] = map[string]any{
			"host_override":  argsChanged(params, "host"),
			"port_override":  argsChanged(params, "port"),
			"identity_check": "none: Hera cannot prove the endpoint still serves the original project",
		}
	}

	switch action {
	case "stack":
		state, err := observeStop(client, wait, intValue(params, "thread", 0))
		if err != nil {
			return debugFail(result, "dap_observe_failed", err.Error())
		}
		result["thread_id"] = state.ThreadID
		if state.Running {
			result["frames"] = []map[string]any{}
			result["running"] = true
			result["notes"] = []string{"no frames were reported, so the debuggee is running"}
			return finishDebugResult(result)
		}
		result["frames"] = state.Frames
	case "variables":
		values, truncated, err := readVariables(client, intValue(params, "frame", 0), wait, intValue(params, "thread", 0))
		if err != nil {
			return debugActionFailure(result, "variables", err)
		}
		result["variables"] = values
		if truncated {
			result["variables_truncated"] = true
		}
	case "evaluate":
		expression := fmt.Sprint(params["expression"])
		frameID, _, err := pausedFrameFor(client, intValue(params, "frame", 0), wait, true, intValue(params, "thread", 0))
		if err != nil {
			return debugActionFailure(result, "evaluate", err)
		}
		response, err := client.Request("evaluate", map[string]any{"expression": expression, "frameId": frameID, "context": "watch"}, wait)
		if err != nil {
			return debugActionFailure(result, "evaluate", err)
		}
		result["expression"], result["result"] = expression, response.Map("body").String("result")
	case "continue", "next", "step-in", "step-out":
		command := map[string]string{"continue": "continue", "next": "next", "step-in": "stepIn", "step-out": "stepOut"}[action]
		state, err := resumeAndObserve(client, command, wait, intValue(params, "thread", 0))
		if err != nil {
			return debugActionFailure(result, action, err)
		}
		if state.Stopped {
			result["stopped"] = map[string]any{"thread_id": state.ThreadID, "frame": state.topFrame(), "frame_count": len(state.Frames), "reason": state.Reason, "resume_confirmed": state.ResumeConfirmed,
				"attribution": "unproven: the stop followed an observed resume event, but the adapter provides no causal request boundary"}
		} else {
			result["running"] = true
			result["resume_confirmed"] = true
		}
	case "detach":
		if _, err := client.Request("disconnect", nil, wait); err != nil {
			return debugActionFailure(result, "detach", err)
		}
		result["detached"] = true
		result["notes"] = []string{"the editor and game keep running; Hera only closes the debug connection"}
		if err := removeDebugSession(recordPath); err != nil {
			return debugFail(result, "session_remove_failed", err.Error())
		}
	}
	return finishDebugResult(result)
}

// removeDebugSession leaves latest alone: a read-then-remove could erase a newer attach.
func removeDebugSession(recordPath string) error {
	if err := os.Remove(recordPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

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
	result["ok"], result["message"] = false, code+": "+detail
	fmt.Fprintf(os.Stderr, "error: %s\n", result["message"])
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
