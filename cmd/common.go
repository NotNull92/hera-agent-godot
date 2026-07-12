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
	instances, err := discovery.Discover()
	if err != nil {
		return nil, err
	}
	inst, err := selectEditor(instances, requireSingle, targetPID)
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
	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return 1
	}
	resp, err := c.Post(tool, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "%s: %s\n", label, resp.Error)
		return 1
	}
	return printData(resp)
}

func dialMutationPostPrint(tool string, params map[string]any, label string) int {
	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return 1
	}
	resp, err := c.Post(tool, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "%s: %s\n", label, resp.Error)
		return 1
	}
	return printData(resp)
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

// printIDs prints just the node paths from a response carrying a "nodes" array
// (scene tree / node find); otherwise it falls back to compact JSON.
func printIDs(data any) {
	if m, ok := data.(map[string]any); ok {
		if nodes, ok := m["nodes"].([]any); ok {
			for _, n := range nodes {
				if nm, ok := n.(map[string]any); ok {
					if p, ok := nm["path"].(string); ok {
						fmt.Println(p)
					}
				}
			}
			return
		}
	}
	out, _ := json.Marshal(data)
	fmt.Println(string(out))
}
