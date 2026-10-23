package cmd

import (
	"reflect"
	"testing"
)

func TestReadNarrowingFlags(t *testing.T) {
	p, err := parseSceneArgs([]string{"tree", "Child", "--depth=0", "--type=Node2D", "--props=visible,position"})
	want := map[string]any{"action": "tree", "path": "Child", "depth": 0, "type": "Node2D", "props": []string{"visible", "position"}}
	if err != nil || !reflect.DeepEqual(p, want) {
		t.Fatalf("tree: %v %v", p, err)
	}
	p, err = parseNodeArgs([]string{"get", ".", "--changed"})
	if err != nil || p["changed"] != true {
		t.Fatalf("changed: %v %v", p, err)
	}
	p, err = parseSignalArgs([]string{"list", ".", "--connected"})
	if err != nil || p["connected"] != true {
		t.Fatalf("connected: %v %v", p, err)
	}
	for _, args := range [][]string{{"get", ".", "--changed", "--prop=name"}, {"get", ".", "--props=name", "--changed"}, {"get", ".", "--changed", "--snapshot"}} {
		if _, err := parseNodeArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, args := range [][]string{{"tree", "--depth=-1"}, {"tree", "--depth=1.5"}, {"tree", "--props="}, {"tree", "--type", ""}, {"tree", "a", "b"}, {"tree", "--unknown"}} {
		if _, err := parseSceneArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
