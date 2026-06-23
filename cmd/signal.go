package cmd

import (
	"fmt"
	"os"
)

// runSignal implements the `signal` command (read + write).
//
//	list <node>                                signals a node exposes + connections
//	connect <from> <signal> <to> <method>      connect a signal (undoable)
//	disconnect <from> <signal> <to> <method>   disconnect a signal (undoable)
func runSignal(args []string) int {
	params, err := parseSignalArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signal: %v\n", err)
		return 2
	}
	if signalActionMutates(params["action"]) {
		return dialMutationPostPrint("signal", params, "signal")
	}
	return dialPostPrint("signal", params, "signal")
}

func signalActionMutates(action any) bool {
	switch action {
	case "connect", "disconnect":
		return true
	default:
		return false
	}
}

func parseSignalArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: signal <list|connect|disconnect> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: signal list <node>")
		}
		return map[string]any{"action": "list", "node": rest[0]}, nil

	case "connect", "disconnect":
		if len(rest) != 4 {
			return nil, fmt.Errorf("usage: signal %s <from> <signal> <to> <method>", sub)
		}
		return map[string]any{
			"action": sub,
			"from":   rest[0],
			"signal": rest[1],
			"to":     rest[2],
			"method": rest[3],
		}, nil

	default:
		return nil, fmt.Errorf("unknown signal subcommand %q (want list|connect|disconnect)", sub)
	}
}
