package cmd

func gameActionMutates(action any) bool {
	switch action {
	case "set", "call", "click":
		return true
	default:
		return false
	}
}
