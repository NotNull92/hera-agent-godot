package cmd

import "testing"

func TestParseSignalArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		want       map[string]any
		wantErr    bool
	}{
		{name: "list", args: []string{"list", "Hero"}, wantAction: "list", want: map[string]any{"node": "Hero"}},
		{name: "list root", args: []string{"list", "."}, wantAction: "list", want: map[string]any{"node": "."}},
		{name: "list no node", args: []string{"list"}, wantErr: true},
		{name: "list extra", args: []string{"list", "a", "b"}, wantErr: true},

		{name: "connect", args: []string{"connect", "Hero", "died", "Root/UI", "_on_died"},
			wantAction: "connect", want: map[string]any{"from": "Hero", "signal": "died", "to": "Root/UI", "method": "_on_died"}},
		{name: "disconnect", args: []string{"disconnect", "Hero", "died", "Root/UI", "_on_died"},
			wantAction: "disconnect", want: map[string]any{"from": "Hero", "signal": "died", "to": "Root/UI", "method": "_on_died"}},

		{name: "connect too few", args: []string{"connect", "Hero", "died", "Root/UI"}, wantErr: true},
		{name: "connect too many", args: []string{"connect", "Hero", "died", "Root/UI", "_on_died", "x"}, wantErr: true},
		{name: "disconnect too few", args: []string{"disconnect", "Hero"}, wantErr: true},

		{name: "no subcommand", args: nil, wantErr: true},
		{name: "unknown subcommand", args: []string{"nope", "x"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseSignalArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p["action"] != tt.wantAction {
				t.Errorf("action = %v, want %v", p["action"], tt.wantAction)
			}
			for k, v := range tt.want {
				if p[k] != v {
					t.Errorf("%s = %v, want %v", k, p[k], v)
				}
			}
		})
	}
}

func TestSignalActionMutates(t *testing.T) {
	for _, tt := range []struct {
		action string
		want   bool
	}{
		{"list", false},
		{"connect", true},
		{"disconnect", true},
	} {
		if got := signalActionMutates(tt.action); got != tt.want {
			t.Errorf("signalActionMutates(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}
