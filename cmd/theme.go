package cmd

import (
	"fmt"
	"os"
	"strings"
)

func runTheme(args []string) int {
	params, err := parseThemeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "theme: %v\n", err)
		return 2
	}
	if params["action"] == "set" {
		return dialMutationPostPrint("theme", params, "theme")
	}
	return dialPostPrint("theme", params, "theme")
}

func parseThemeArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: theme <get|set> <res://theme.tres> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "get":
		return parseThemeGetArgs(rest)
	case "set":
		return parseThemeSetArgs(rest)
	default:
		return nil, fmt.Errorf("unknown theme subcommand %q (want get|set)", sub)
	}
}

func parseThemeGetArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: theme get <res://theme.tres> [--type <ThemeType>]")
	}
	params := map[string]any{"action": "get", "path": args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a theme type")
			}
			params["type"] = args[i+1]
			i++
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseThemeSetArgs(args []string) (map[string]any, error) {
	usage := "usage: theme set <res://theme.tres> --type <ThemeType> [--color <name=Color(r,g,b,a)>] [--constant <name=int>] [--font-size <name=int>]"
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", usage)
	}
	params := map[string]any{"action": "set", "path": args[0]}
	colors := map[string]any{}
	constants := map[string]any{}
	fontSizes := map[string]any{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a theme type")
			}
			params["type"] = args[i+1]
			i++
		case "--color", "--constant", "--font-size":
			flag := args[i]
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a name=value", flag)
			}
			name, value, err := splitThemeItem(flag, args[i+1])
			if err != nil {
				return nil, err
			}
			switch flag {
			case "--color":
				colors[name] = value
			case "--constant":
				constants[name] = value
			default:
				fontSizes[name] = value
			}
			i++
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}

	if params["type"] == nil {
		return nil, fmt.Errorf("theme set requires --type")
	}
	if len(colors)+len(constants)+len(fontSizes) == 0 {
		return nil, fmt.Errorf("theme set requires at least one --color, --constant or --font-size")
	}
	if len(colors) > 0 {
		params["colors"] = colors
	}
	if len(constants) > 0 {
		params["constants"] = constants
	}
	if len(fontSizes) > 0 {
		params["font_sizes"] = fontSizes
	}
	return params, nil
}

// splitThemeItem splits name=value on the first "=", so Color(0.3, 0.8, 1, 1)
// keeps any "=" free content intact on the right-hand side.
func splitThemeItem(flag, arg string) (string, string, error) {
	name, value, found := strings.Cut(arg, "=")
	if !found {
		return "", "", fmt.Errorf("%s expects name=value, got %q", flag, arg)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("%s requires an item name before '='", flag)
	}
	if strings.TrimSpace(value) == "" {
		return "", "", fmt.Errorf("%s requires a value after '=' for %q", flag, name)
	}
	return name, value, nil
}
