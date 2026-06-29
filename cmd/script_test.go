package cmd

import "testing"

func TestParseScriptOpenArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLine   any
		wantColumn any
		wantErr    bool
	}{
		{name: "open", args: []string{"open", "res://scripts/player.gd"}},
		{name: "open line column", args: []string{"open", "res://scripts/player.gd", "--line", "12", "--column", "3"}, wantLine: 12, wantColumn: 3},
		{name: "open missing path", args: []string{"open"}, wantErr: true},
		{name: "open dangling line", args: []string{"open", "res://a.gd", "--line"}, wantErr: true},
		{name: "open bad line", args: []string{"open", "res://a.gd", "--line", "x"}, wantErr: true},
		{name: "open zero line", args: []string{"open", "res://a.gd", "--line", "0"}, wantErr: true},
		{name: "open dangling column", args: []string{"open", "res://a.gd", "--column"}, wantErr: true},
		{name: "open bad column", args: []string{"open", "res://a.gd", "--column", "x"}, wantErr: true},
		{name: "open unknown flag", args: []string{"open", "res://a.gd", "--bad"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseScriptArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p["action"] != "open" {
				t.Fatalf("action = %v, want open", p["action"])
			}
			if p["path"] != "res://scripts/player.gd" {
				t.Fatalf("path = %v, want res://scripts/player.gd", p["path"])
			}
			if tt.wantLine != nil && p["line"] != tt.wantLine {
				t.Fatalf("line = %v, want %v", p["line"], tt.wantLine)
			}
			if tt.wantColumn != nil && p["column"] != tt.wantColumn {
				t.Fatalf("column = %v, want %v", p["column"], tt.wantColumn)
			}
		})
	}
}

func TestScriptOpenActionMutates(t *testing.T) {
	if !scriptActionMutates("open") {
		t.Fatal("scriptActionMutates(open) = false, want true")
	}
}
