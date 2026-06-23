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
	inst, err := selectEditor(instances, requireSingle)
	if err != nil {
		return nil, err
	}
	return client.New(fmt.Sprintf("http://127.0.0.1:%d", inst.Port)), nil
}

func selectEditor(instances []discovery.Instance, requireSingle bool) (discovery.Instance, error) {
	if len(instances) == 0 {
		return discovery.Instance{}, fmt.Errorf("no live Godot editor found (is the Hera Agent plugin enabled?)")
	}
	if requireSingle && len(instances) > 1 {
		return discovery.Instance{}, fmt.Errorf("multiple live Godot editors found (%s); close extra editors before running mutation commands", instancePIDs(instances))
	}
	return instances[0], nil // most recent
}

func instancePIDs(instances []discovery.Instance) string {
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, fmt.Sprintf("pid %d", inst.PID))
	}
	return strings.Join(ids, ", ")
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
			return last, fmt.Errorf("run state: %s", resp.Error)
		}
		if resp.OK && playingFlag(resp) == want {
			return resp, nil
		}
		if !time.Now().Before(deadline) {
			return last, fmt.Errorf("timed out waiting for playing=%t", want)
		}
		time.Sleep(150 * time.Millisecond)
	}
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
