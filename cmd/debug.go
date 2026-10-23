package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
