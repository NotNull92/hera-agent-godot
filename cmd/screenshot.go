package cmd

import (
	"fmt"
	"os"
	"strconv"
)

const maxScreenshotSize = 4096

// runScreenshot implements
// `hera-godot screenshot [--path <p>] [--width N] [--height N] [--transparent]`.
//
// Renders the edited scene off-screen and saves a PNG; returns the absolute path.
func runScreenshot(args []string) int {
	// `diff` is a local subcommand: it compares two PNGs already on disk and
	// needs no editor. Flags all start with "--", so a bare word cannot collide.
	if len(args) > 0 && args[0] == "diff" {
		return runScreenshotDiff(args[1:])
	}
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
	args, params, err := parseEvidenceOptions(args, true)
	if err != nil {
		return nil, err
	}
	var runtime bool
	var sized bool
	var transparent bool
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--path"):
			raw, next, err := takeFlagValue(args, i, "--path")
			if err != nil {
				return nil, err
			}
			i = next - 1
			params["path"] = raw
		case isFlag(args[i], "--width"):
			raw, next, err := takeFlagValue(args, i, "--width")
			if err != nil {
				return nil, err
			}
			i = next - 1
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 || n > maxScreenshotSize {
				return nil, fmt.Errorf("invalid --width %q (want 1-%d)", raw, maxScreenshotSize)
			}
			params["width"] = n
			sized = true
		case isFlag(args[i], "--height"):
			raw, next, err := takeFlagValue(args, i, "--height")
			if err != nil {
				return nil, err
			}
			i = next - 1
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 || n > maxScreenshotSize {
				return nil, fmt.Errorf("invalid --height %q (want 1-%d)", raw, maxScreenshotSize)
			}
			params["height"] = n
			sized = true
		case args[i] == "--transparent":
			params["transparent"] = true
			transparent = true
		case args[i] == "--runtime":
			params["runtime"] = true
			runtime = true
		case args[i] == "--analyze":
			params["analyze"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if runtime && (sized || transparent) {
		return nil, fmt.Errorf("--runtime captures the running game viewport and cannot be combined with --width, --height, or --transparent")
	}
	if params["runtime_session_id"] != nil && !runtime {
		return nil, fmt.Errorf("--runtime-session requires --runtime")
	}
	return params, nil
}
