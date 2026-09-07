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
		switch args[i] {
		case "--pause":
			if pauseSet {
				return nil, fmt.Errorf("game clock accepts only one of --pause or --resume")
			}
			params["paused"] = true
			pauseSet = true
		case "--resume":
			if pauseSet {
				return nil, fmt.Errorf("game clock accepts only one of --pause or --resume")
			}
			params["paused"] = false
			pauseSet = true
		case "--time-scale":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--time-scale requires a value")
			}
			i++
			scale, err := strconv.ParseFloat(args[i], 64)
			if err != nil || scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
				return nil, fmt.Errorf("invalid --time-scale %q (want a number greater than 0)", args[i])
			}
			params["time_scale"] = scale
		case "--step":
			params["step"] = true
		case "--physics":
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
