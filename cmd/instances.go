package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func runInstances(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "instances: does not accept arguments")
		return 2
	}
	scan, err := discovery.DiscoverScan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "instances: %v\n", err)
		return 1
	}
	data := map[string]any{"count": len(scan.Live), "instances": scan.Live}
	if len(scan.Stale) > 0 {
		data["stale"] = scan.Stale
	}
	var out []byte
	if outputMode == "json" {
		out, err = json.MarshalIndent(data, "", "  ")
	} else {
		out, err = json.Marshal(data)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "instances: %v\n", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}
