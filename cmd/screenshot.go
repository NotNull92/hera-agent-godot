package cmd

import (
	"fmt"
	"os"
)

// runScreenshot implements `hera-agent-godot screenshot [--view 2d|3d] [--path <p>]`.
func runScreenshot(args []string) int {
	params, err := parseScreenshotArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		return 2
	}
	return dialPostPrint("screenshot", params, "screenshot")
}

func parseScreenshotArgs(args []string) (map[string]any, error) {
	params := map[string]any{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--view":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--view requires a value")
			}
			i++
			if args[i] != "2d" && args[i] != "3d" {
				return nil, fmt.Errorf("invalid --view %q (want 2d|3d)", args[i])
			}
			params["view"] = args[i]
		case "--path":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--path requires a value")
			}
			i++
			params["path"] = args[i]
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}
