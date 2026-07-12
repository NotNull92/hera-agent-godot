package cmd

import (
	"fmt"
	"os"
	"strconv"
)

const maxScreenshotSize = 4096

// runScreenshot implements
// `hera screenshot [--path <p>] [--width N] [--height N] [--transparent]`.
//
// Renders the edited scene off-screen and saves a PNG; returns the absolute path.
func runScreenshot(args []string) int {
	params, err := parseScreenshotArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		return 2
	}
	if params["runtime"] == true {
		delete(params, "runtime")
		params["action"] = "screenshot"
		return dialPostPrint("game", params, "screenshot")
	}
	return dialPostPrint("screenshot", params, "screenshot")
}

func parseScreenshotArgs(args []string) (map[string]any, error) {
	params := map[string]any{}
	var runtime bool
	var sized bool
	var transparent bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--path requires a value")
			}
			i++
			params["path"] = args[i]
		case "--width":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--width requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 || n > maxScreenshotSize {
				return nil, fmt.Errorf("invalid --width %q (want 1-%d)", args[i], maxScreenshotSize)
			}
			params["width"] = n
			sized = true
		case "--height":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--height requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 || n > maxScreenshotSize {
				return nil, fmt.Errorf("invalid --height %q (want 1-%d)", args[i], maxScreenshotSize)
			}
			params["height"] = n
			sized = true
		case "--transparent":
			params["transparent"] = true
			transparent = true
		case "--runtime":
			params["runtime"] = true
			runtime = true
		case "--analyze":
			params["analyze"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if runtime && (sized || transparent) {
		return nil, fmt.Errorf("--runtime captures the running game viewport and cannot be combined with --width, --height, or --transparent")
	}
	return params, nil
}
