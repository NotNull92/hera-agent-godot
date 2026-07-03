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
		return nil, fmt.Errorf("usage: classdb <info|methods|properties|signals|constants|enums|inherits> <Class> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "info", "methods", "properties":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: classdb %s <Class>", sub)
		}
		return map[string]any{"action": sub, "class": rest[0]}, nil
	case "signals", "constants", "enums":
		return parseClassDBMetadataArgs(sub, rest)
	case "inherits":
		if len(rest) != 2 {
			return nil, fmt.Errorf("usage: classdb inherits <Class> <BaseClass>")
		}
		return map[string]any{"action": "inherits", "class": rest[0], "base": rest[1]}, nil
	default:
		return nil, fmt.Errorf("unknown classdb subcommand %q (want info|methods|properties|signals|constants|enums|inherits)", sub)
	}
}

func parseClassDBMetadataArgs(sub string, args []string) (map[string]any, error) {
	if len(args) < 1 || len(args) > 2 || args[0] == "--own" {
		return nil, fmt.Errorf("usage: classdb %s <Class> [--own]", sub)
	}
	params := map[string]any{"action": sub, "class": args[0]}
	if len(args) == 2 {
		if args[1] != "--own" {
			return nil, fmt.Errorf("usage: classdb %s <Class> [--own]", sub)
		}
		params["own"] = true
	}
	return params, nil
}
