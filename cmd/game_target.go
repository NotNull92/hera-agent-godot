package cmd

import (
	"fmt"
	"strconv"
)

func parseGameInvocation(args []string) (int, []string, error) {
	if len(args) == 0 || !isFlag(args[0], "--pid") {
		return 0, args, nil
	}
	raw, next, err := takeFlagValue(args, 0, "--pid")
	if err != nil || next >= len(args) {
		return 0, nil, fmt.Errorf("usage: game --pid <positive integer> <subcommand> ...")
	}
	pid, err := strconv.Atoi(raw)
	if err != nil || pid <= 0 {
		return 0, nil, fmt.Errorf("invalid --pid %q (want a positive integer)", raw)
	}
	return pid, args[next:], nil
}

func targetGameParams(params map[string]any, pid int) map[string]any {
	if pid > 0 {
		params["pid"] = pid
	}
	return params
}
