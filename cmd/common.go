package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/discovery"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

// dialEditor finds the most recent live editor and returns a client to it.
//
// TODO(later): honor a --instance <pid> flag to disambiguate when several
// editors are running (discovery currently just picks the most recent).
func dialEditor() (*client.Client, error) {
	instances, err := discovery.Discover()
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("no live Godot editor found (is the Hera Agent plugin enabled?)")
	}
	inst := instances[0] // most recent
	return client.New(fmt.Sprintf("http://127.0.0.1:%d", inst.Port)), nil
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
	out, err := json.Marshal(resp.Data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}
