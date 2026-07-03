package cmd

import "testing"

func TestParseClassDBArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantClass  any
		wantBase   any
		wantOwn    any
		wantErr    bool
	}{
		{name: "info", args: []string{"info", "Node2D"}, wantAction: "info", wantClass: "Node2D"},
		{name: "methods", args: []string{"methods", "EditorInterface"}, wantAction: "methods", wantClass: "EditorInterface"},
		{name: "properties", args: []string{"properties", "Sprite2D"}, wantAction: "properties", wantClass: "Sprite2D"},
		{name: "signals", args: []string{"signals", "Button"}, wantAction: "signals", wantClass: "Button"},
		{name: "signals own", args: []string{"signals", "Button", "--own"}, wantAction: "signals", wantClass: "Button", wantOwn: true},
		{name: "constants", args: []string{"constants", "Control"}, wantAction: "constants", wantClass: "Control"},
		{name: "constants own", args: []string{"constants", "Control", "--own"}, wantAction: "constants", wantClass: "Control", wantOwn: true},
		{name: "enums", args: []string{"enums", "Control"}, wantAction: "enums", wantClass: "Control"},
		{name: "enums own", args: []string{"enums", "Control", "--own"}, wantAction: "enums", wantClass: "Control", wantOwn: true},
		{name: "inherits", args: []string{"inherits", "Sprite2D", "Node2D"}, wantAction: "inherits", wantClass: "Sprite2D", wantBase: "Node2D"},
		{name: "missing subcommand", wantErr: true},
		{name: "info missing class", args: []string{"info"}, wantErr: true},
		{name: "methods extra", args: []string{"methods", "Node", "x"}, wantErr: true},
		{name: "signals extra", args: []string{"signals", "Button", "x"}, wantErr: true},
		{name: "signals own missing class", args: []string{"signals", "--own"}, wantErr: true},
		{name: "inherits missing base", args: []string{"inherits", "Sprite2D"}, wantErr: true},
		{name: "unknown", args: []string{"exists", "Node"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := parseClassDBArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if params["action"] != tt.wantAction {
				t.Fatalf("action = %v, want %v", params["action"], tt.wantAction)
			}
			if tt.wantClass != nil && params["class"] != tt.wantClass {
				t.Fatalf("class = %v, want %v", params["class"], tt.wantClass)
			}
			if tt.wantBase != nil && params["base"] != tt.wantBase {
				t.Fatalf("base = %v, want %v", params["base"], tt.wantBase)
			}
			if tt.wantOwn != nil && params["own"] != tt.wantOwn {
				t.Fatalf("own = %v, want %v", params["own"], tt.wantOwn)
			}
		})
	}
}
