package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestOutputShape(t *testing.T) {
	cases := []struct {
		name, tool, params, input, want string
	}{
		{"nodes", "scene", `{}`, `{"count":2,"truncated":false,"nodes":[{"path":".","name":"Main"},{"path":"Parent/Child","name":"Child"}]}`, `{"nodes":[{"path":".","name":"Main"},{"path":"Parent/Child"}]}`},
		{"explicit UI fields", "game", `{"action":"ui_tree","fields":["path","name"]}`, `{"controls":[{"path":"/root/Main/Play","name":"Play"}]}`, `{"controls":[{"path":"/root/Main/Play","name":"Play"}]}`},
		{"default UI fields", "game", `{"action":"ui_tree"}`, `{"controls":[{"path":"/root/Main/Play","name":"Play"}]}`, `{"controls":[{"path":"/root/Main/Play"}]}`},
		{"batch explicit UI fields", "batch", `{"commands":[{"tool":"game","params":{"action":"ui_tree","fields":["path","name"]}}]}`, `{"results":[{"tool":"game","ok":true,"data":{"controls":[{"path":"/root/Main/Play","name":"Play"}]}}]}`, `{"results":[{"tool":"game","ok":true,"data":{"controls":[{"path":"/root/Main/Play","name":"Play"}]}}]}`},
		{"ambiguous count", "fixture", `{}`, `{"count":2,"items":[1,2,3],"warnings":["a","b"]}`, `{"count":2,"items":[1,2,3],"warnings":["a","b"]}`},
		{"two matching arrays", "fixture", `{}`, `{"count":1,"items":[1],"warnings":["a"]}`, `{"count":1,"items":[1],"warnings":["a"]}`},
		{"empty array", "fixture", `{}`, `{"count":0,"items":[]}`, `{"items":[]}`},
		{"null is not an array", "fixture", `{}`, `{"count":0,"items":null}`, `{"count":0,"items":null}`},
		{"capped", "scene", `{}`, `{"count":3,"truncated":true,"nodes":[{"path":"Child","name":"Alias"}]}`, `{"count":3,"truncated":true,"nodes":[{"path":"Child","name":"Alias"}]}`},
		{"user values", "game", `{"action":"call"}`, `{"result_value":{"count":1,"truncated":false,"nodes":[{"path":"Child","name":"Child"}]},"evidence":{"truncated":false}}`, `{"result_value":{"count":1,"truncated":false,"nodes":[{"path":"Child","name":"Child"}]},"evidence":{"truncated":false}}`},
		{"file log", "output", `{}`, `{"available":true,"log_path":"log.txt","source":"file","lines":[]}`, `{"available":true,"lines":[]}`},
		{"unreadable log", "diagnostics", `{}`, `{"available":false,"log_path":"log.txt","source":"file"}`, `{"available":false,"log_path":"log.txt"}`},
		{"editor log", "output", `{}`, `{"available":true,"log_path":"log.txt","source":"editor","editor_session_id":"session"}`, `{"available":true,"log_path":"log.txt","source":"editor","editor_session_id":"session"}`},
		{"input echo", "game", `{"action":"input","kind":"key","mode":"press","key":"a","physical":true}`, `{"kind":"key","mode":"press","key":"A","keycode":65,"physical":true,"route":"input","pid":42,"log_count":1}`, `{"key":"A","keycode":65,"route":"input","pid":42,"log_count":1}`},
		{"default status", "status", `{}`, `{"pid":42,"project_path":"project","editor_session_id":"session","capabilities":{"test":"supported"},"godot_commit":"abc"}`, `{"pid":42}`},
		{"node get", "node", `{"action":"get"}`, `{"path":"Child","name":"Child","properties":{"name":"Child"}}`, `{"path":"Child","properties":{"name":"Child"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var params map[string]any
			var input, want any
			for i, text := range []string{tc.params, tc.input, tc.want} {
				target := []any{&params, &input, &want}[i]
				if err := json.Unmarshal([]byte(text), target); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := json.Marshal(input)
			got := reshapeOutput(tc.tool, params, input)
			if !equalJSON(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("mutated internal RPC data")
			}
		})
	}
}

func TestBatchOutputMatchesStandalone(t *testing.T) {
	old := statusSDKProbe
	statusSDKProbe = func(context.Context) (string, string, string) { return "unsupported", "", "test" }
	defer func() { statusSDKProbe = old }()
	var commands []any
	var response map[string]any
	if err := json.Unmarshal([]byte(`[{"tool":"status","params":{"capabilities":true}},{"tool":"game","params":{"action":"input","kind":"text","text":"hello"}}]`), &commands); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"count":2,"results":[{"tool":"status","ok":true,"data":{"pid":42,"editor_session_id":"session","project_path":"project","capabilities":{"test":"supported"}}},{"tool":"game","ok":true,"data":{"kind":"text","text":"hello","pid":43,"log_count":1}}]}`), &response); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(response)
	got := reshapeOutput("batch", map[string]any{"commands": commands}, response).(map[string]any)
	if _, exists := got["count"]; exists {
		t.Fatal("redundant batch count")
	}
	for i, item := range got["results"].([]any) {
		command := commands[i].(map[string]any)
		original := response["results"].([]any)[i].(map[string]any)
		want := reshapeOutput(command["tool"].(string), command["params"].(map[string]any), original["data"])
		if !equalJSON(item.(map[string]any)["data"], want) {
			t.Fatalf("batch child %d differs", i)
		}
	}
	after, _ := json.Marshal(response)
	if string(before) != string(after) {
		t.Fatal("mutated batch RPC data")
	}
	status := got["results"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if status["capabilities"].(map[string]any)["test"] != "supported" {
		t.Fatal("lost capability block")
	}
}
