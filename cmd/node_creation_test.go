package cmd

import "testing"

func TestNodeCreationProperties(t *testing.T) {
	for _, action := range []string{"add", "instance"} {
		p, err := parseNodeArgs([]string{action, "Node", "--set=name=", "--set", "visible=false"})
		if err != nil {
			t.Fatal(err)
		}
		props := p["props"].(map[string]any)
		if props["name"] != "" || props["visible"] != "false" {
			t.Fatal(props)
		}
		for _, invalid := range []string{"name", "=value"} {
			if _, err := parseNodeArgs([]string{action, "Node", "--set", invalid}); err == nil {
				t.Fatal("accepted", invalid)
			}
		}
	}
}
