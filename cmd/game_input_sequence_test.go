package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameInputSequence(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`[{"frame":0,"action":"ui_accept","pressed":true},{"frame":2,"action":"ui_accept","pressed":false}]`, true},
		{`[]`, false},
		{`null`, false},
		{`[{"action":"x","pressed":true}]`, false},
		{`[{"frame":0,"action":"x"}]`, false},
		{`[{"frame":-1,"action":"x","pressed":true}]`, false},
		{`[{"frame":121,"action":"x","pressed":true}]`, false},
		{`[{"frame":0.5,"action":"x","pressed":true}]`, false},
		{`[{"frame":0,"action":" ","pressed":true}]`, false},
		{`[{"frame":0,"action":"x","pressed":true,"extra":1}]`, false},
		{`[{"frame":1,"action":"x","pressed":true},{"frame":0,"action":"x","pressed":false}]`, false},
		{`[] []`, false},
		{`[{"frame":0,"action":"x","pressed":true}]` + strings.Repeat(" ", 65536), false},
		{"[" + strings.Repeat(`{"frame":0,"action":"x","pressed":true},`, 128) + `{"frame":0,"action":"x","pressed":true}]`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.json")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			params, err := parseGameInputArgs([]string{"sequence", "--file", path})
			if (err == nil) != tc.valid {
				t.Fatalf("params=%v err=%v valid=%v", params, err, tc.valid)
			}
			if tc.valid && (params["action"] != "input" || params["kind"] != "sequence") {
				t.Fatalf("wrong route: %v", params)
			}
		})
	}
}
