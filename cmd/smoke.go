package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const smokeWaitTimeout = 10 * time.Second

type smokeOptions struct {
	runGame  bool
	skipGame bool
}

type smokeRunner struct {
	client *client.Client
	steps  []map[string]any
}

func runSmoke(args []string) int {
	opts, err := parseSmokeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoke: %v\n", err)
		return 2
	}
	var c *client.Client
	if smokeRequiresMutation(opts) {
		c, err = dialMutationEditor()
	} else {
		c, err = dialEditor()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoke: %v\n", err)
		return 1
	}
	runner := smokeRunner{client: c, steps: make([]map[string]any, 0, 6)}
	steps, err := runner.run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoke: %v\n", err)
		return 1
	}
	return printData(&protocol.Response{OK: true, Data: successData(steps)})
}

func smokeRequiresMutation(opts smokeOptions) bool {
	return opts.runGame
}

func parseSmokeArgs(args []string) (smokeOptions, error) {
	var opts smokeOptions
	for _, arg := range args {
		switch arg {
		case "--run-game":
			opts.runGame = true
		case "--skip-game":
			opts.skipGame = true
		default:
			return smokeOptions{}, fmt.Errorf("unknown flag %q", arg)
		}
	}
	if opts.runGame && opts.skipGame {
		return smokeOptions{}, fmt.Errorf("--run-game and --skip-game are mutually exclusive")
	}
	return opts, nil
}

func (r *smokeRunner) run(opts smokeOptions) ([]map[string]any, error) {
	if err := r.post("status", nil); err != nil {
		return nil, err
	}
	if err := r.post("diagnostics", map[string]any{"lines": 5}); err != nil {
		return nil, err
	}
	if err := r.post("scene", map[string]any{"action": "open_scenes"}); err != nil {
		return nil, err
	}
	if opts.skipGame {
		return r.steps, nil
	}
	if !opts.runGame {
		return r.steps, nil
	}
	if err := r.post("run", map[string]any{"action": "play_current"}); err != nil {
		return nil, err
	}
	if _, err := pollPlaying(r.client, true, smokeWaitTimeout); err != nil {
		return nil, err
	}
	if err := r.post("game", map[string]any{"action": "tree"}); err != nil {
		return nil, err
	}
	if err := r.post("run", map[string]any{"action": "stop"}); err != nil {
		return nil, err
	}
	if _, err := pollPlaying(r.client, false, smokeWaitTimeout); err != nil {
		return nil, err
	}
	return r.steps, nil
}

func (r *smokeRunner) post(tool string, params map[string]any) error {
	resp, err := r.client.Post(tool, params)
	if err != nil {
		return fmt.Errorf("%s: %w", tool, err)
	}
	if !resp.OK {
		return fmt.Errorf("%s: %s", tool, resp.Error)
	}
	r.steps = append(r.steps, map[string]any{"tool": tool, "ok": true})
	return nil
}

func successData(steps []map[string]any) map[string]any {
	return map[string]any{"ok": true, "count": len(steps), "steps": steps}
}
