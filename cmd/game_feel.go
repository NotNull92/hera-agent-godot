package cmd

import (
	"fmt"
	"os"
)

func runGameFeel(args []string) int {
	params := map[string]any{}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "game_feel: usage: game_feel [topic]")
		return 2
	}
	if len(args) == 1 {
		params["topic"] = args[0]
	}
	return dialPostPrint("game_feel", params, "game_feel")
}
