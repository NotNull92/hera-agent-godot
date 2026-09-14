package cmd

import (
	"strings"
	"testing"
)

func TestOperationParsing(t *testing.T) {
	for _, args := range [][]string{{}, {"status"}, {"cancel", "bad"}, {"submit", "session:1:x"}, {"status", "s:1:x", "extra"}} {
		if _, err := parseOperation(args); err == nil {
			t.Fatalf("accepted malformed arguments %v", args)
		}
	}
	params, err := parseOperation([]string{"submit", "session:1:x", "--request", `{"tool":"game","params":{"action":"call"}}`})
	if err != nil || len(params["digest"].(string)) != 64 {
		t.Fatalf("valid submit: %v, %v", params, err)
	}
	if _, err := parseOperation([]string{"submit", "session:1:x", "--request", strings.Repeat("x", 16385)}); err == nil {
		t.Fatal("accepted oversized input")
	}
}
