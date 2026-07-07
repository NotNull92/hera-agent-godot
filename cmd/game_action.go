package cmd

func gameActionMutates(action any) bool {
	switch action {
	case "set", "call", "click", "input", "input_log":
		return true
	default:
		return false
	}
}
