package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

const waitTimeout = 10 * time.Second

// runRun implements `hera-agent-godot run [--scene <res://...>] [--current] [--wait]`.
//
// Default (no flag) plays the main scene; --current plays the edited scene;
// --scene plays a specific scene. --wait polls until the play session starts.
func runRun(args []string) int {
	params, wait, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 2
	}

	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}
	if err := resolveMainSceneRunParams(params); err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}

	resp, err := c.Post("run", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "run: %s\n", resp.Error)
		return 1
	}

	if wait {
		resp, err = pollPlaying(c, true, waitTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run: %v\n", err)
			return 1
		}
		if _, err := pollGameReady(c, sceneFromResponse(resp), waitTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "run: %v\n", err)
			return 1
		}
	}
	return printData(resp)
}

// runStop implements `hera-agent-godot stop [--wait]` (addon `run` tool, stop action).
func runStop(args []string) int {
	wait := false
	for _, a := range args {
		switch a {
		case "--wait":
			wait = true
		default:
			fmt.Fprintf(os.Stderr, "stop: unknown flag %q\n", a)
			return 2
		}
	}

	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop: %v\n", err)
		return 1
	}

	resp, err := c.Post("run", map[string]any{"action": "stop"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "stop: %s\n", resp.Error)
		return 1
	}

	if wait {
		resp, err = pollPlaying(c, false, waitTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stop: %v\n", err)
			return 1
		}
		if err := pollGameInstancesStopped(c, waitTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "stop: %v\n", err)
			return 1
		}
	}
	return printData(resp)
}

// parseRunArgs turns CLI args into the addon `run` params and the --wait flag.
func parseRunArgs(args []string) (params map[string]any, wait bool, err error) {
	var scene string
	var current bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scene":
			if i+1 >= len(args) {
				return nil, false, fmt.Errorf("--scene requires a path")
			}
			i++
			scene = args[i]
		case "--current":
			current = true
		case "--wait":
			wait = true
		default:
			return nil, false, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if scene != "" && current {
		return nil, false, fmt.Errorf("--scene and --current are mutually exclusive")
	}

	params = map[string]any{}
	switch {
	case scene != "":
		params["action"] = "play_custom"
		params["scene"] = scene
	case current:
		params["action"] = "play_current"
	default:
		params["action"] = "play_main"
	}
	return params, wait, nil
}

func resolveMainSceneRunParams(params map[string]any) error {
	if params["action"] != "play_main" {
		return nil
	}
	instances, err := discovery.Discover()
	if err != nil {
		return err
	}
	inst, err := selectEditor(instances, true, targetPID)
	if err != nil {
		return err
	}
	scenePath, err := readMainSceneFromProjectFile(inst.ProjectPath)
	if err != nil {
		return err
	}
	if scenePath == "" {
		return nil
	}
	params["action"] = "play_custom"
	params["scene"] = scenePath
	return nil
}
