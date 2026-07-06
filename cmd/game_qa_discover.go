package cmd

import (
	"fmt"
	"os"
)

func runGameQADiscover(args []string) int {
	params, err := parseGameQADiscoverArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa discover: %v\n", err)
		return 2
	}
	return dialPostPrint("game", params, "game qa discover")
}

func parseGameQADiscoverArgs(args []string) (map[string]any, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("usage: game qa discover [path]")
	}
	params := map[string]any{"action": "qa_discover"}
	if len(args) == 1 {
		params["path"] = normalizeGameNodePath(args[0])
	}
	return params, nil
}

func qaDiscoverParamsFromQAStep(step gameQAStep) map[string]any {
	params := map[string]any{"action": "qa_discover"}
	if step.Path != "" {
		params["path"] = normalizeGameNodePath(step.Path)
	}
	return params
}
