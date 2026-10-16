package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCSharpHarness(t *testing.T) {
	project := t.TempDir()
	for _, tc := range []struct {
		source string
		bad    bool
	}{
		{`using Godot; public partial class Tool : SceneTree {}`, false},
		{`public partial class Tool : global::Godot.MainLoop {}`, false},
		{`public partial class Tool : Godot.Node {}`, true},
		{"// class Tool : Node {}\nclass Tool : CustomHarness {}", false},
	} {
		if err := os.WriteFile(filepath.Join(project, "Tool.cs"), []byte(tc.source), 0600); err != nil {
			t.Fatal(err)
		}
		if err := engineHarnessError(project, "res://Tool.cs"); (err != nil) != tc.bad {
			t.Fatalf("%s: %v", tc.source, err)
		}
	}
	if err := engineHarnessError(project, "res://../Tool.cs"); err == nil {
		t.Fatal("escaped path accepted")
	}
}

func TestCSharpRunPreflight(t *testing.T) {
	for _, tc := range []struct {
		v  string
		ok bool
	}{{"4.7.2.stable.mono.official.abc1234", true}, {"4.7.stable.official.abc1234", false}, {"4.2.stable.mono.official.abc1234", false}} {
		if csharpEngineSupported(parseEngineVersion(tc.v)) != tc.ok {
			t.Fatal(tc)
		}
	}
	p := t.TempDir()
	os.WriteFile(filepath.Join(p, "project.godot"), []byte("[dotnet]\nproject/assembly_name=\"Custom\"\n"), 0600)
	if _, err := csharpAssembly(p); err == nil {
		t.Fatal("absent assembly accepted")
	}
	dir := filepath.Join(p, ".godot/mono/temp/bin/Debug")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "Custom.dll"), []byte("fixture"), 0600)
	if _, err := csharpAssembly(p); err != nil {
		t.Fatal(err)
	}
}
