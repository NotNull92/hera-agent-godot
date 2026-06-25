package cmd

import (
	"fmt"
	"os"
)

func runClassDB(args []string) int {
	params, err := parseClassDBArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "classdb: %v\n", err)
		return 2
	}
	return dialPostPrint("classdb", params, "classdb")
}

func parseClassDBArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: classdb <info|methods|properties|inherits> <Class> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "info", "methods", "properties":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: classdb %s <Class>", sub)
		}
		return map[string]any{"action": sub, "class": rest[0]}, nil
	case "inherits":
		if len(rest) != 2 {
			return nil, fmt.Errorf("usage: classdb inherits <Class> <BaseClass>")
		}
		return map[string]any{"action": "inherits", "class": rest[0], "base": rest[1]}, nil
	default:
		return nil, fmt.Errorf("unknown classdb subcommand %q (want info|methods|properties|inherits)", sub)
	}
}
