package cmd

import (
	"strings"
	"testing"
)

func TestOperationParsing(t *testing.T) {
	for _, args := range [][]string{{}, {"status"}, {"cancel", "bad"}, {"submit", "session:1:x"}, {"status", "s:1:x", "extra"}} {
		if _, _, err := parseOperation(args); err == nil {
			t.Fatalf("accepted malformed arguments %v", args)
		}
	}
	params, verbose, err := parseOperation([]string{"submit", "session:1:x", "--request", `{"tool":"game","params":{"action":"call"}}`})
	if err != nil || verbose || len(params["digest"].(string)) != 64 {
		t.Fatalf("valid submit: %v verbose=%v err=%v", params, verbose, err)
	}
	params, verbose, err = parseOperation([]string{"status", "session:1:x", "--verbose"})
	if err != nil || !verbose || params["action"] != "status" {
		t.Fatalf("verbose status: %v verbose=%v err=%v", params, verbose, err)
	}
	if _, _, err := parseOperation([]string{"submit", "session:1:x", "--request", strings.Repeat("x", 16385)}); err == nil {
		t.Fatal("accepted oversized input")
	}
}
