package cmd

import (
	"fmt"
	"math"
	"strconv"
)

func parseGameClockArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "clock"}
	pauseSet := false
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--pause":
			if pauseSet {
				return nil, fmt.Errorf("game clock accepts only one of --pause or --resume")
			}
			params["paused"] = true
			pauseSet = true
		case args[i] == "--resume":
			if pauseSet {
				return nil, fmt.Errorf("game clock accepts only one of --pause or --resume")
			}
			params["paused"] = false
			pauseSet = true
		case isFlag(args[i], "--time-scale"):
			raw, next, err := takeFlagValue(args, i, "--time-scale")
			if err != nil {
				return nil, err
			}
			i = next - 1
			scale, err := strconv.ParseFloat(raw, 64)
			if err != nil || scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
				return nil, fmt.Errorf("invalid --time-scale %q (want a number greater than 0)", raw)
			}
			params["time_scale"] = scale
		case args[i] == "--step":
			params["step"] = true
		case args[i] == "--physics":
			params["physics"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if boolVal(params["physics"]) && !boolVal(params["step"]) {
		return nil, fmt.Errorf("--physics requires --step")
	}
	if boolVal(params["step"]) {
		if paused, ok := params["paused"].(bool); ok && !paused {
			return nil, fmt.Errorf("--step leaves the tree paused; do not pass --resume")
		}
	}
	return params, nil
}

func gameClockMutates(params map[string]any) bool {
	if _, ok := params["paused"]; ok {
		return true
	}
	if _, ok := params["time_scale"]; ok {
		return true
	}
	return boolVal(params["step"])
}

func boolVal(value any) bool {
	flag, ok := value.(bool)
	return ok && flag
}
