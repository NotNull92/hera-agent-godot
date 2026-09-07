package cmd

import (
	"fmt"
	"strconv"
)

func parseGameInvocation(args []string) (int, []string, error) {
	if len(args) == 0 || args[0] != "--pid" {
		return 0, args, nil
	}
	if len(args) < 3 {
		return 0, nil, fmt.Errorf("usage: game --pid <positive integer> <subcommand> ...")
	}
	pid, err := strconv.Atoi(args[1])
	if err != nil || pid <= 0 {
		return 0, nil, fmt.Errorf("invalid --pid %q (want a positive integer)", args[1])
	}
	return pid, args[2:], nil
}

func targetGameParams(params map[string]any, pid int) map[string]any {
	if pid > 0 {
		params["pid"] = pid
	}
	return params
}
