package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

// runStatus implements `hera-agent-godot status`: find a live editor, ask it for
// status, and print the result as compact JSON.
func runStatus(args []string) int {
	_ = args

	instances, err := discovery.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	if len(instances) == 0 {
		fmt.Fprintln(os.Stderr, "status: no live Godot editor found (is the Hera Agent plugin enabled?)")
		return 1
	}

	inst := instances[0] // most recent
	resp, err := client.New(fmt.Sprintf("http://127.0.0.1:%d", inst.Port)).Post("status", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "status: %s\n", resp.Error)
		return 1
	}

	out, err := json.Marshal(resp.Data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}
