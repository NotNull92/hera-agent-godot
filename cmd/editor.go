package cmd

import (
	"fmt"
	"os"
)

func runEditor(args []string) int {
	params, err := parseEditorArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "editor: %v\n", err)
		return 2
	}
	if editorActionMutates(params["action"]) {
		return dialMutationPostPrint("editor", params, "editor")
	}
	return dialPostPrint("editor", params, "editor")
}

func editorActionMutates(action any) bool {
	switch action {
	case "select", "clear_selection":
		return true
	default:
		return false
	}
}

func parseEditorArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: editor <state|selected|select|clear-selection> ...")
	}
	switch args[0] {
	case "state":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: editor state")
		}
		return map[string]any{"action": "state"}, nil
	case "selected":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: editor selected")
		}
		return map[string]any{"action": "selected"}, nil
	case "select":
		return parseEditorSelectArgs(args[1:])
	case "clear-selection":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: editor clear-selection")
		}
		return map[string]any{"action": "clear_selection"}, nil
	default:
		return nil, fmt.Errorf("unknown editor subcommand %q (want state|selected|select|clear-selection)", args[0])
	}
}

func parseEditorSelectArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: editor select <node-path> [--add]")
	}
	params := map[string]any{"action": "select", "path": args[0]}
	for _, arg := range args[1:] {
		switch arg {
		case "--add":
			params["add"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", arg)
		}
	}
	return params, nil
}
