package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/discovery"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func dialEditor() (*client.Client, error) {
	return dialEditorWithMode(false)
}

func dialMutationEditor() (*client.Client, error) {
	return dialEditorWithMode(true)
}

func dialEditorWithMode(requireSingle bool) (*client.Client, error) {
	scan, err := discovery.DiscoverScan()
	if err != nil {
		return nil, err
	}
	inst, err := selectEditor(scan, requireSingle, targetPID)
	if err != nil {
		return nil, err
	}
	c := client.NewWithTimeout(fmt.Sprintf("http://127.0.0.1:%d", inst.Port), requestTimeout)
	c.Token = client.LoadSharedToken()
	return c, nil
}

// dialPostPrint dials the editor, sends one tool request, and prints the
// response Data as compact JSON. label is used in error messages.
func dialPostPrint(tool string, params map[string]any, label string) int {
	return dialAndPostPrint(dialEditor, postPrintRequest{tool: tool, params: params, label: label})
}

func dialMutationPostPrint(tool string, params map[string]any, label string) int {
	return dialAndPostPrint(dialMutationEditor, postPrintRequest{tool: tool, params: params, label: label})
}

type postPrintRequest struct {
	tool    string
	params  map[string]any
	label   string
	reshape func(any) any
}

func dialAndPostPrint(dial func() (*client.Client, error), request postPrintRequest) int {
	c, err := dial()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", request.label, err)
		return 1
	}
	needed := make([]string, 0, 3)
	guard := requiresNodeGuard(request.tool, request.params)
	evidenceRequested := wantsLinkedEvidence(request.tool, request.params)
	logs := editorLogRequest(request.tool, request.params)
	if guard {
		needed = append(needed, "node_set_guard")
	}
	if evidenceRequested {
		needed = append(needed, "linked_evidence")
	}
	if logs {
		needed = append(needed, "editor_log_cursor")
	}
	if len(needed) > 0 {
		caps, status, statusErr := negotiate(c, needed...)
		if statusErr != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", request.label, statusErr)
			return 1
		}
		if logs {
			if !status.OK {
				fmt.Fprintf(os.Stderr, "%s: %s\n", request.label, status.Error)
				return 1
			}
			if capabilityValue(caps, "editor_log_cursor") != "supported" {
				return printData(editorLogUnavailable(status, caps))
			}
		}
		if guard {
			if !status.OK {
				fmt.Fprintf(os.Stderr, "%s: %s\n", request.label, status.Error)
				return 1
			}
			if capabilityValue(caps, "node_set_guard") != "supported" {
				fmt.Fprintf(os.Stderr, "%s: capability_unavailable: addon does not support node_set_guard\n", request.label)
				return 1
			}
		}
		if evidenceRequested && (!status.OK || capabilityValue(caps, "linked_evidence") != "supported") {
			fmt.Fprintf(os.Stderr, "%s: capability_unavailable: addon does not support linked_evidence\n", request.label)
			return 1
		}
	}
	protectActions(request.tool, request.params)
	resp, err := c.Post(request.tool, request.params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", request.label, err)
		return 1
	}
	if !resp.OK {
		if evidenceRequested && resp.Data != nil {
			printData(resp)
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", request.label, resp.Error)
		return 1
	}
	if evidenceRequested && !linkedEvidenceAvailable(request.tool, request.params, resp.Data) {
		printData(resp)
		fmt.Fprintf(os.Stderr, "%s: evidence_unavailable\n", request.label)
		return 1
	}
	if request.reshape != nil {
		resp.Data = request.reshape(resp.Data)
	}
	return printData(resp)
}

func forEachBatchChild(params map[string]any, fn func(tool string, params map[string]any)) {
	commands, _ := params["commands"].([]any)
	for _, command := range commands {
		entry, _ := command.(map[string]any)
		subTool, _ := entry["tool"].(string)
		subParams, _ := entry["params"].(map[string]any)
		if subParams == nil {
			subParams = map[string]any{}
			if entry != nil {
				entry["params"] = subParams
			}
		}
		fn(subTool, subParams)
	}
}

func negotiate(c *client.Client, _ ...string) (map[string]any, *protocol.Response, error) {
	status, err := c.Post("status", nil)
	if err != nil {
		return nil, nil, err
	}
	data, _ := status.Data.(map[string]any)
	caps, _ := data["capabilities"].(map[string]any)
	return caps, status, nil
}

func capabilityValue(caps map[string]any, name string) string {
	if caps == nil {
		return ""
	}
	value, _ := caps[name].(string)
	return value
}

func editorLogRequest(tool string, params map[string]any) bool {
	return (tool == "output" || tool == "diagnostics") && params["source"] == "editor"
}

func protectActions(tool string, params map[string]any) {
	if tool == "batch" {
		forEachBatchChild(params, protectActions)
		return
	}
	useGuardedNodeAction(tool, params)
	prepareLinkedEvidence(tool, params)
}

func pollPlaying(c *client.Client, want bool, timeout time.Duration) (*protocol.Response, error) {
	deadline := time.Now().Add(timeout)
	var last *protocol.Response
	for {
		resp, err := c.Post("run", map[string]any{"action": "state"})
		if err != nil {
			return nil, err
		}
		last = resp
		if !resp.OK {
			return last, enrichLaunchError(c, fmt.Errorf("run state: %s", resp.Error))
		}
		if resp.OK && playingFlag(resp) == want {
			return resp, nil
		}
		if !time.Now().Before(deadline) {
			return last, enrichLaunchError(c, fmt.Errorf("timed out waiting for playing=%t", want))
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func pollGameReady(c *client.Client, expectedScene string, timeout time.Duration) (*protocol.Response, error) {
	deadline := time.Now().Add(timeout)
	var last *protocol.Response
	var lastErr error
	for {
		resp, err := c.Post("game", map[string]any{"action": "tree"})
		if err != nil {
			lastErr = err
		} else {
			last = resp
			if resp.OK && gameSceneMatches(resp, expectedScene) {
				return resp, nil
			}
			if !resp.OK {
				lastErr = fmt.Errorf("game tree: %s", resp.Error)
			}
		}
		if !time.Now().Before(deadline) {
			if lastErr != nil {
				return last, enrichLaunchError(c, fmt.Errorf("timed out waiting for game scene %q: %w", expectedScene, lastErr))
			}
			return last, enrichLaunchError(c, fmt.Errorf("timed out waiting for game scene %q", expectedScene))
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func pollGameInstancesStopped(c *client.Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		resp, err := c.Post("game", map[string]any{"action": "instances"})
		if err != nil {
			lastErr = err
		} else if !resp.OK {
			lastErr = fmt.Errorf("game instances: %s", resp.Error)
		} else if gameInstanceCount(resp) == 0 {
			return nil
		} else {
			_, _ = c.Post("run", map[string]any{"action": "stop"})
		}
		if !time.Now().Before(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for game instances to stop: %w", lastErr)
			}
			return fmt.Errorf("timed out waiting for game instances to stop")
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func gameInstanceCount(resp *protocol.Response) int {
	m, ok := resp.Data.(map[string]any)
	if !ok {
		return 0
	}
	instances, ok := m["instances"].([]any)
	if !ok {
		return 0
	}
	return len(instances)
}

// playingFlag extracts the boolean "playing" field from a run/state response.
func playingFlag(resp *protocol.Response) bool {
	m, ok := resp.Data.(map[string]any)
	if !ok {
		return false
	}
	b, _ := m["playing"].(bool)
	return b
}

func gameSceneMatches(resp *protocol.Response, expectedScene string) bool {
	if expectedScene == "" {
		return true
	}
	m, ok := resp.Data.(map[string]any)
	if !ok {
		return false
	}
	scene, _ := m["scene"].(string)
	return scene == expectedScene
}

func sceneFromResponse(resp *protocol.Response) string {
	if resp == nil {
		return ""
	}
	m, ok := resp.Data.(map[string]any)
	if !ok {
		return ""
	}
	scene, _ := m["scene"].(string)
	return scene
}

func enrichLaunchError(c *client.Client, base error) error {
	details := make([]string, 0, 2)
	if resp, err := c.Post("diagnostics", map[string]any{"lines": 20}); err == nil {
		if resp.OK {
			details = append(details, "diagnostics: "+compactJSON(resp.Data))
		} else if resp.Error != "" {
			details = append(details, "diagnostics: "+resp.Error)
		}
	}
	if resp, err := c.Post("output", map[string]any{"type": "error", "lines": 40}); err == nil {
		if resp.OK {
			details = append(details, "output: "+compactJSON(resp.Data))
		} else if resp.Error != "" {
			details = append(details, "output: "+resp.Error)
		}
	}
	if len(details) == 0 {
		return base
	}
	return fmt.Errorf("%w\n%s", base, strings.Join(details, "\n"))
}

func compactJSON(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(out)
}

// printData prints a response's Data as compact JSON. Returns a process exit code.
func printData(resp *protocol.Response) int {
	switch outputMode {
	case "json":
		out, err := json.MarshalIndent(resp.Data, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		fmt.Println(string(out))
	case "ids":
		printIDs(resp.Data)
	default:
		out, err := json.Marshal(resp.Data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		fmt.Println(string(out))
	}
	return 0
}

// printIDs prints just the paths from a response carrying a "nodes" or
// "controls" array; otherwise it falls back to compact JSON.
func printIDs(data any) {
	if m, ok := data.(map[string]any); ok {
		if printPathList(m, "nodes") || printPathList(m, "controls") {
			return
		}
	}
	out, _ := json.Marshal(data)
	fmt.Println(string(out))
}

func hasPathList(data map[string]any, key string) bool {
	_, ok := data[key].([]any)
	return ok
}

func printPathList(data map[string]any, key string) bool {
	items, ok := data[key].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		entry, _ := item.(map[string]any)
		if path, ok := entry["path"].(string); ok {
			fmt.Println(path)
		}
	}
	return true
}
