package cmd

import (
	"reflect"
	"testing"
	"time"
)

func TestCommandOptions(t *testing.T) {
	t.Parallel()
	options, rest, err := parseCommandOptions([]string{"--json", "--instance=42", "--timeout", "1500", "--engine=engine", "--csharp-host", "host", "status"})
	want := commandOptions{OutputMode: "json", TargetPID: 42, RequestTimeout: 1500 * time.Millisecond, EnginePath: "engine", CSharpHostPath: "host"}
	if err != nil || options != want || !reflect.DeepEqual(rest, []string{"status"}) {
		t.Fatalf("%+v %v %v", options, rest, err)
	}
	empty, _, err := parseCommandOptions([]string{"status"})
	if err != nil || empty != (commandOptions{}) {
		t.Fatalf("options leaked: %+v %v", empty, err)
	}
}
func TestCommandOptionsErrors(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--instance"}, {"--timeout"}, {"--instance=bad"}, {"--timeout=0"}, {"--engine"}} {
		if _, _, err := parseCommandOptions(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
