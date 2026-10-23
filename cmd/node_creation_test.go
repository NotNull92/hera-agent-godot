package cmd

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNodeCreationProperties(t *testing.T) {
	for _, action := range []string{"add", "instance"} {
		p, err := parseNodeArgs([]string{action, "Node", "--set=name=", "--set", "visible=false", "--set=name=Last"})
		if err != nil {
			t.Fatal(err)
		}
		props := p["props"].([][]string)
		if !reflect.DeepEqual(props, [][]string{{"name", ""}, {"visible", "false"}, {"name", "Last"}}) {
			t.Fatal(props)
		}
		encoded, err := json.Marshal(props)
		if err != nil || string(encoded) != `[["name",""],["visible","false"],["name","Last"]]` {
			t.Fatalf("wire order %s: %v", encoded, err)
		}
		for _, invalid := range []string{"name", "=value"} {
			if _, err := parseNodeArgs([]string{action, "Node", "--set", invalid}); err == nil {
				t.Fatal("accepted", invalid)
			}
		}
	}
}
