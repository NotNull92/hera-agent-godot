package cmd

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEnginePrecedence(t *testing.T) {
	executable, err := os.CreateTemp(t.TempDir(), "engine-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	executable.Close()
	path := executable.Name()
	other, err := os.CreateTemp(t.TempDir(), "engine-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	other.Close()

	previous := engineFlagPath
	t.Cleanup(func() { engineFlagPath = previous })

	engineFlagPath = path
	t.Setenv("HERA_GODOT_ENGINE", other.Name())
	t.Setenv("GODOT_BIN", "")
	ref, err := resolveEngine()
	if err != nil || ref.Path != path || ref.Source != "flag" {
		t.Fatalf("flag should win: %+v err=%v", ref, err)
	}

	engineFlagPath = ""
	ref, err = resolveEngine()
	if err != nil || ref.Path != other.Name() || ref.Source != "env:HERA_GODOT_ENGINE" {
		t.Fatalf("HERA_GODOT_ENGINE should win over GODOT_BIN: %+v err=%v", ref, err)
	}

	t.Setenv("HERA_GODOT_ENGINE", "")
	t.Setenv("GODOT_BIN", path)
	ref, err = resolveEngine()
	if err != nil || ref.Path != path || ref.Source != "env:GODOT_BIN" {
		t.Fatalf("GODOT_BIN fallback: %+v err=%v", ref, err)
	}

	t.Setenv("GODOT_BIN", "")
	if _, err := resolveEngine(); err == nil || !strings.Contains(err.Error(), "--engine") {
		t.Fatalf("missing engine should name the flag: %v", err)
	}

	engineFlagPath = filepath.Join(t.TempDir(), "missing.exe")
	if _, err := resolveEngine(); err == nil {
		t.Fatal("missing engine path should fail")
	}
}

func TestParseEngineVersion(t *testing.T) {
	version := parseEngineVersion("4.7.2.stable.mono.official.ed1daf0bf")
	if version.String != "4.7.2.stable.mono.official.ed1daf0bf" || version.Status != "stable" ||
		!version.Mono || version.Commit != "ed1daf0bf" {
		t.Fatalf("unexpected parse: %+v", version)
	}
	plain := parseEngineVersion("4.2.1.stable")
	if plain.Mono || plain.Commit != "" || plain.Status != "stable" {
		t.Fatalf("unexpected plain parse: %+v", plain)
	}
	if empty := parseEngineVersion("nonsense"); empty.Status != "" || empty.Commit != "" {
		t.Fatalf("garbage must not be parsed into fields: %+v", empty)
	}
}

func TestScanHelpFlags(t *testing.T) {
	help := scanHelpFlags("Option legend (this build = editor):\n  -h, --help   Display this help message.\n  -e, --editor   Start the editor.\n  --dap-port <port>   GDScript Debug Adapter Protocol.\n")
	if !help.Available || !help.Editor {
		t.Fatalf("editor build should be detected: %+v", help)
	}
	if !help.Present["--dap-port"] || !help.Present["--editor"] {
		t.Fatalf("flags should be present: %+v", help.Present)
	}
	if help.Present["--write-movie"] {
		t.Fatalf("absent flags must stay false: %+v", help.Present)
	}
	colored := scanHelpFlags("\x1b[92m--headless\x1b[0m                        \x1b[92mR\x1b[0m  Enable headless mode.\n\x1b[92m--write-movie <file>\x1b[0m              \x1b[92mR\x1b[0m  Write a video.\n")
	if !colored.Present["--headless"] {
		t.Fatal("ANSI-colored flags must be detected")
	}
	if !colored.Present["--write-movie"] {
		t.Fatalf("flag with a value must be detected: %+v", colored.Present)
	}
	template := scanHelpFlags("General options:\n  -h, --help   Display this help message.\n")
	if template.Editor {
		t.Fatal("template build must not report an editor flag")
	}
}

func TestReadExportPresetNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export_presets.cfg")
	contents := "[preset.0]\n\nname=\"Windows Desktop\"\nplatform=\"Windows Desktop\"\nrunnable=true\n\n[preset.0.options]\nbinary_format/embed_pck=false\nname=\"not-a-preset-name\"\n\n[preset.1]\n\nname=\"Windows \\\"QA\\\"\"\n\n[preset.2]\n\n[preset.2.options]\nname=\"ignored-options-name\"\n\n[preset.3]\n\nname=\"Linux/X11\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := readExportPresetNames(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Windows Desktop", `Windows "QA"`, "Linux/X11"}
	if len(names) != len(want) {
		t.Fatalf("names=%v want=%v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names=%v want=%v", names, want)
		}
	}
	if _, err := readExportPresetNames(filepath.Join(dir, "absent.cfg")); err == nil {
		t.Fatal("missing export_presets.cfg must be an error")
	}
}

func TestParseScriptRunArgs(t *testing.T) {
	previous := engineFlagPath
	t.Cleanup(func() { engineFlagPath = previous })
	engineFlagPath = ""

	params, err := parseScriptRunArgs([]string{"res://tools/probe.gd", "--quit-after", "5", "--", "a", "b", "--flag"})
	if err != nil {
		t.Fatal(err)
	}
	if params["action"] != "run" || params["path"] != "res://tools/probe.gd" || params["quit_after"] != 5 {
		t.Fatalf("unexpected params: %v", params)
	}
	args, ok := params["arguments"].([]string)
	if !ok || len(args) != 3 || args[2] != "--flag" {
		t.Fatalf("arguments after -- must pass through: %v", params["arguments"])
	}

	params, err = parseScriptRunArgs([]string{"res://a.gd", "--engine=tool.exe"})
	if err != nil || engineFlagPath != "tool.exe" || params["arguments"] != nil {
		t.Fatalf("engine flag should be accepted: %v err=%v", params, err)
	}

	for _, bad := range [][]string{
		{},
		{"scripts/probe.gd"},
		{"res://a.gd", "--quit-after", "-1"},
		{"res://a.gd", "--frames", "2"},
	} {
		if _, err := parseScriptRunArgs(bad); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestParseCaptureMovieArgs(t *testing.T) {
	params, err := parseCaptureMovieArgs([]string{"out.avi"})
	if err != nil {
		t.Fatal(err)
	}
	if params["frames"] != 60 || params["fps"] != 30 {
		t.Fatalf("defaults changed: %v", params)
	}
	output, _ := params["output"].(string)
	if !filepath.IsAbs(output) {
		t.Fatalf("output should be absolute: %v", output)
	}
	params, err = parseCaptureMovieArgs([]string{"frames.png", "--frames", "10", "--fps", "24", "--resolution", "320x180", "--scene", "uid://abc"})
	if err != nil || params["frames"] != 10 || params["fps"] != 24 || params["resolution"] != "320x180" || params["scene"] != "uid://abc" {
		t.Fatalf("unexpected params: %v err=%v", params, err)
	}
	for _, bad := range [][]string{
		{},
		{"out.avi", "--frames", "0"},
		{"out.avi", "--fps", "x"},
		{"out.avi", "--resolution", "320"},
		{"out.avi", "--headless"},
	} {
		if _, err := parseCaptureMovieArgs(bad); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestXmlGenerationIgnoresDestinationArtifacts(t *testing.T) {
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(destination, "A.xml"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage, err := stageArtifacts(destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "notes.txt"), []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n, err := countXMLFilesFollowingLinks(stage, maxDocsScanDepth); err != nil || n != 0 {
		t.Fatalf("old XML or unrelated output accepted: %d %v", n, err)
	}
	if err := os.WriteFile(filepath.Join(stage, "A.xml"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n, err := countXMLFilesFollowingLinks(stage, maxDocsScanDepth); err != nil || n != 1 {
		t.Fatalf("identical generated XML lost: %d %v", n, err)
	}
	if err := os.Mkdir(filepath.Join(stage, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "nested", "B.xml"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n, err := countXMLFilesFollowingLinks(stage, maxDocsScanDepth); err != nil || n != 2 {
		t.Fatalf("nested generated XML lost: %d %v", n, err)
	}
	if _, err := countXMLFilesFollowingLinks(stage, 0); err == nil {
		t.Fatal("over-depth scan must fail")
	}
}

func TestExportArtifactProblem(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "game.exe")
	if err := os.WriteFile(file, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A regular-file target is what ordinary binary, pack and patch exports use;
	// a directory is accepted too because plugin-driven targets write one.
	if info, err := os.Stat(file); err != nil {
		t.Fatal(err)
	} else if code := exportArtifactProblem(nil); code != "" || info.IsDir() {
		t.Fatalf("a regular-file export target must be accepted: code=%q isDir=%v", code, info.IsDir())
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	} else if code := exportArtifactProblem(err); code != "" {
		t.Fatalf("a directory export target must be accepted: %q", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "absent.pck")); exportArtifactProblem(err) != "no_artifact_produced" {
		t.Fatalf("a missing export target must fail: %v", err)
	}
}

func TestResolveOutputPath(t *testing.T) {
	project := t.TempDir()
	resolved := resolveOutputPath(project, "res://build/shot.png")
	if !strings.HasPrefix(filepath.ToSlash(resolved), filepath.ToSlash(project)) {
		t.Fatalf("res:// output must resolve into the project: %q", resolved)
	}
	absolute := filepath.Join(project, "shot.avi")
	if got := resolveOutputPath(project, absolute); got != absolute {
		t.Fatalf("absolute output must pass through: %q", got)
	}
}

func TestParsePerfSampleArgs(t *testing.T) {
	params, err := parsePerfSampleArgs(nil)
	if err != nil || params["top"] != 10 || params["headless"] != false {
		t.Fatalf("defaults changed: %v err=%v", params, err)
	}
	params, err = parsePerfSampleArgs([]string{"--top", "3", "--headless"})
	if err != nil || params["top"] != 3 || params["headless"] != true {
		t.Fatalf("unexpected params: %v err=%v", params, err)
	}
	if _, err := parsePerfSampleArgs([]string{"--top", "nope"}); err == nil {
		t.Fatal("non-numeric top must fail")
	}
	if _, err := parsePerfSampleArgs([]string{"--frames", "30"}); err == nil {
		t.Fatal("--frames is gone: the engine ignores --quit-after under benchmark flags")
	}
}

func TestReadBenchmarkFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bench.json")
	contents := "{\n\t\"[Servers] Rendering\": 0.728383,\n\t\"[Startup] Servers\": 1.230992,\n\t\"label\": \"ignored\"\n}"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	values, err := readBenchmarkFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values["[Servers] Rendering"] != 0.728383 {
		t.Fatalf("values=%v", values)
	}
	entries := sortedBenchmarkEntries(values)
	if entries[0]["name"] != "[Startup] Servers" || entries[1]["name"] != "[Servers] Rendering" {
		t.Fatalf("ordering should be cost-descending: %v", entries)
	}
	if _, err := readBenchmarkFile(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("missing benchmark file must fail")
	}
}

func TestParseDocsDumpArgs(t *testing.T) {
	if _, err := parseDocsDumpArgs(nil); err == nil {
		t.Fatal("--out is required")
	}
	params, err := parseDocsDumpArgs([]string{"--out", "docs", "--format", "xml"})
	if err != nil || params["out"] != "docs" || params["format"] != "xml" {
		t.Fatalf("unexpected params: %v err=%v", params, err)
	}
	if _, err := parseDocsDumpArgs([]string{"--out", "docs", "--scripts", "--format", "json"}); err == nil {
		t.Fatal("--scripts with json must fail")
	}
	if _, err := parseDocsDumpArgs([]string{"--out", "docs", "--format", "yaml"}); err == nil {
		t.Fatal("unknown format must fail")
	}
}

func TestParseClassDBDumpArgs(t *testing.T) {
	if _, err := parseClassDBDumpArgs(nil); err == nil {
		t.Fatal("--out is required")
	}
	params, err := parseClassDBDumpArgs([]string{"--out", "api"})
	if err != nil || params["out"] != "api" {
		t.Fatalf("unexpected params: %v err=%v", params, err)
	}
}

func TestParseProjectEngineArgs(t *testing.T) {
	params, err := parseProjectImportArgs([]string{"--project", "game"})
	if err != nil || params["action"] != "import" || params["project"] != "game" {
		t.Fatalf("unexpected import params: %v err=%v", params, err)
	}
	params, err = parseProjectExportArgs([]string{"Windows Desktop", "build/game.pck"})
	if err != nil || params["kind"] != "pack" || params["preset"] != "Windows Desktop" {
		t.Fatalf("unexpected export params: %v err=%v", params, err)
	}
	params, err = parseProjectExportArgs([]string{"Web", "out.zip", "--kind", "release", "--patches", "a.pck,b.pck"})
	if err != nil || params["kind"] != "release" || params["patches"] != "a.pck,b.pck" {
		t.Fatalf("unexpected export params: %v err=%v", params, err)
	}
	for _, bad := range [][]string{
		{"only-preset"},
		{"Web", "out.zip", "--kind", "unknown"},
		{"Web", "out.zip", "--patches"},
	} {
		if _, err := parseProjectExportArgs(bad); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestProjectStateHelpers(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".godot", "editor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".godot", "uid_cache.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "main.gd.uid"), []byte("uid://x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := projectCacheState(project)
	if cache["uid_cache"] != true || cache["editor_state"] != true || cache["imported"] != false {
		t.Fatalf("cache state=%v", cache)
	}
	if count := countScriptUIDFiles(project); count != 1 {
		t.Fatalf("uid sidecar count=%d", count)
	}
	addon := projectHeraAddonState(project)
	if addon["installed"] != false || addon["listed_in_project_file"] != false {
		t.Fatalf("addon state=%v", addon)
	}
}

func TestIsolatedUserEnvCoversEveryRoot(t *testing.T) {
	values := isolatedUserEnv("C:/tmp/isolated")
	joined := strings.Join(values, "\n")
	for _, key := range []string{"HOME=", "USERPROFILE=", "APPDATA=", "XDG_DATA_HOME=", "XDG_CONFIG_HOME="} {
		if !strings.Contains(joined, key) {
			t.Fatalf("missing %s in %v", key, values)
		}
	}
}

func TestResolveProjectDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := resolveProjectDir(dir); err == nil || strings.Contains(err.Error(), "--project") {
		t.Fatalf("explicit missing project should not suggest --project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte("config_version=5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := resolveProjectDir(dir)
	if err != nil || filepath.Base(resolved) != filepath.Base(dir) {
		t.Fatalf("resolved=%q err=%v", resolved, err)
	}
	if _, err := resolveProjectDir(filepath.Join(dir, "nested")); err == nil {
		t.Fatal("explicit missing project must fail")
	}
	t.Chdir(t.TempDir())
	if _, err := resolveProjectDir(""); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("default lookup should suggest --project: %v", err)
	}
}

func TestEngineCheckDetail(t *testing.T) {
	if detail := editorBuildDetail(true); !strings.Contains(detail, "editor") {
		t.Fatalf("detail=%q", detail)
	}
	if detail := editorBuildDetail(false); !strings.Contains(detail, "template") {
		t.Fatalf("detail=%q", detail)
	}
}

func TestScriptExtendsBase(t *testing.T) {
	cases := map[string]string{
		"extends SceneTree\n":                                      "SceneTree",
		"@tool\nextends MainLoop\n":                                "MainLoop",
		"# comment\n\nextends Node2D # trailing\n":                 "Node2D",
		"class_name Harness\nextends SceneTree\n":                  "SceneTree",
		"extends \"res://base.gd\"\n":                              "",
		"var x := 1\n":                                             "",
		"\ufeffextends SceneTree\n":                                "SceneTree",
		"@icon(\"res://i.svg\")\nclass_name H\nextends MainLoop\n": "MainLoop",
	}
	for source, want := range cases {
		if got := scriptExtendsBase(source); got != want {
			t.Fatalf("scriptExtendsBase(%q)=%q want=%q", source, got, want)
		}
	}
}

func TestEngineHarnessError(t *testing.T) {
	project := t.TempDir()
	write := func(name, contents string) {
		if err := os.WriteFile(filepath.Join(project, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("harness.gd", "extends SceneTree\n")
	write("notaharness.gd", "extends Node2D\n")
	write("node.gd", "@tool\nextends Node\n")
	write("custom.gd", "extends MyHarness\n")
	if err := engineHarnessError(project, "res://harness.gd"); err != nil {
		t.Fatalf("SceneTree harness must pass: %v", err)
	}
	for _, path := range []string{"res://notaharness.gd", "res://node.gd"} {
		err := engineHarnessError(project, path)
		if err == nil || !strings.Contains(err.Error(), "SceneTree") {
			t.Fatalf("%s should be rejected with guidance: %v", path, err)
		}
	}
	if err := engineHarnessError(project, "res://custom.gd"); err != nil {
		t.Fatalf("a custom base class must reach the engine: %v", err)
	}
	if err := engineHarnessError(project, "res://missing.gd"); err != nil {
		t.Fatalf("unreadable scripts are left to the engine: %v", err)
	}
	if err := engineHarnessError(project, "res://Player.cs"); err == nil {
		t.Fatal("C# preflight must reject an unreadable harness")
	}
}

func TestFirstFrameSize(t *testing.T) {
	dir := t.TempDir()
	header := make([]byte, 24)
	copy(header, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	binary.BigEndian.PutUint32(header[8:12], 13)
	copy(header[12:16], "IHDR")
	binary.BigEndian.PutUint32(header[16:20], 1152)
	binary.BigEndian.PutUint32(header[20:24], 648)
	stale := filepath.Join(dir, "shot00000000.png")
	if err := os.WriteFile(stale, header, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shot00000001.png"), header, 0o644); err != nil {
		t.Fatal(err)
	}
	width, height, ok := firstFrameSize(filepath.Join(dir, "shot.png"))
	if !ok || width != 1152 || height != 648 {
		t.Fatalf("size=%dx%d ok=%v", width, height, ok)
	}
	// Rewriting an existing first frame makes it this run's output again.
	if err := os.WriteFile(stale, append(header, 0), 0o644); err != nil {
		t.Fatal(err)
	}
	width, height, ok = firstFrameSize(filepath.Join(dir, "shot.png"))
	if !ok || width != 1152 || height != 648 {
		t.Fatalf("rewritten: size=%dx%d ok=%v", width, height, ok)
	}
	if _, _, ok := firstFrameSize(filepath.Join(dir, "shot.avi")); ok {
		t.Fatal("only PNG sequences report a frame size")
	}
	if _, _, ok := firstFrameSize(filepath.Join(dir, "empty.png")); ok {
		t.Fatal("missing frames must not report a size")
	}
}

func TestPublishClassDBFiles(t *testing.T) {
	source := t.TempDir()
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "class_db_api.json"), []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "class_db_api_editor.json"), []byte("editor"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "unrelated.json"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory at the destination must fail that copy, not be reported OK.
	if err := os.Mkdir(filepath.Join(out, "class_db_api_editor.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := publishClassDBFiles(source, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two ClassDB files, got %v", files)
	}
	ok, failed := 0, 0
	for _, file := range files {
		if file.OK {
			ok++
		} else if file.Error != "" {
			failed++
		}
	}
	if ok != 1 || failed != 1 {
		t.Fatalf("ok=%d failed=%d files=%v", ok, failed, files)
	}
	if data, err := os.ReadFile(filepath.Join(out, "class_db_api.json")); err != nil || string(data) != "runtime" {
		t.Fatalf("published content mismatch: %q err=%v", data, err)
	}
}

func TestCountXMLFilesFollowingLinks(t *testing.T) {
	dir := t.TempDir()
	if count, err := countXMLFilesFollowingLinks(filepath.Join(dir, "absent"), 12); err != nil || count != 0 {
		t.Fatalf("missing directory counts as empty: %d err=%v", count, err)
	}
	nested := filepath.Join(dir, "doc", "modules")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "doc", "A.xml"), filepath.Join(nested, "B.xml"), filepath.Join(dir, "README.md")} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if count, err := countXMLFilesFollowingLinks(dir, 12); err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	// A linked documentation tree must not hide existing XML behind the link.
	linked := filepath.Join(dir, "linked")
	if err := os.Symlink(nested, linked); err != nil {
		t.Logf("symlinks unavailable on this host: %v", err)
		return
	}
	if count, err := countXMLFilesFollowingLinks(linked, 12); err != nil || count != 1 {
		t.Fatalf("linked tree count=%d err=%v", count, err)
	}
}

// TestXMLScanDepthFailsClosed pins the guard rule that an incomplete scan must
// not look like a clean destination: a tree deeper than the bound returns an
// error, so the command refuses instead of regenerating over hidden XML.
func TestXMLScanDepthFailsClosed(t *testing.T) {
	root := t.TempDir()
	deep := root
	for i := 0; i < 14; i++ {
		deep = filepath.Join(deep, "level")
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "Sentinel.xml"), []byte("<class/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := countXMLFilesFollowingLinks(root, 12); err == nil {
		t.Fatal("a tree deeper than the scan bound must fail closed")
	}
}

func TestResolveDestination(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "linked-docs")
	if err := os.Symlink(target, link); err != nil {
		t.Logf("symlinks unavailable on this host: %v", err)
		return
	}
	resolved, err := resolveDestination(link)
	if err != nil || resolved == link {
		t.Fatalf("a linked destination must resolve to its target: %q err=%v", resolved, err)
	}
	missing := filepath.Join(t.TempDir(), "fresh")
	if resolved, err := resolveDestination(missing); err != nil || resolved != missing {
		t.Fatalf("a missing destination stays as given: %q err=%v", resolved, err)
	}
}

func TestRunFailureClassification(t *testing.T) {
	if code := runFailureCode(engineproc.Result{ExitCode: -1, RunErr: errors.New("signal: killed")}); code == "" {
		t.Fatal("a reaped signal exit must fail even without abandonment")
	}
	if code := runFailureCode(engineproc.Result{ExitCode: 1}); code != "engine_exit_nonzero" {
		t.Fatalf("nonzero exit must fail: %q", code)
	}
	if code := runFailureCode(engineproc.Result{ExitCode: 0, RunErr: exec.ErrWaitDelay}); code != "output_incomplete" {
		t.Fatalf("WaitDelay must fail: %q", code)
	}
	if code := runFailureCode(engineproc.Result{ExitCode: 0}); code != "" {
		t.Fatalf("a clean run must not fail: %q", code)
	}
	if !runFailed(engineproc.Result{ExitCode: 1}) || runFailed(engineproc.Result{ExitCode: 0}) {
		t.Fatal("runFailed must follow runFailureCode")
	}
	fields := runResultFields(engineproc.Result{ExitCode: 0, RunErr: exec.ErrWaitDelay})
	if fields["output_incomplete"] != true {
		t.Fatalf("incomplete output must be reported: %v", fields)
	}
}

func TestSnapshotFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "extension_api.json")
	if snapshot := snapshotFile(path); snapshot.Exists {
		t.Fatalf("missing file must not exist: %+v", snapshot)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := snapshotFile(path)
	if !snapshot.Exists || snapshot.IsDir || snapshot.Bytes != 2 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if dirSnapshot := snapshotFile(t.TempDir()); !dirSnapshot.Exists || !dirSnapshot.IsDir {
		t.Fatalf("directory snapshot: %+v", dirSnapshot)
	}
}

func TestUnescapeGodotString(t *testing.T) {
	cases := map[string]string{
		`Windows \"QA\"`: "Windows \"QA\"",
		`back\\slash`:    `back\slash`,
		`plain`:          "plain",
		`tab\there`:      "tab\there",
	}
	for input, want := range cases {
		if got := unescapeGodotString(input); got != want {
			t.Fatalf("unescape(%q)=%q want=%q", input, got, want)
		}
	}
}

func TestEngineInfoExplicitProject(t *testing.T) {
	project := t.TempDir()
	stdout, stderr, code := captureContractRun([]string{"engine", "info", "--project", project})
	if code != 1 || !strings.Contains(stderr, "project.godot") || stdout != "" {
		t.Fatalf("ignored invalid project: %d %q %q", code, stdout, stderr)
	}
	executable := os.Getenv("HERA_TEST_GODOT")
	if executable == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(project, "project.godot"), []byte("config_version=5\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code = captureContractRun([]string{"--engine", executable, "engine", "info", "--project", project})
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || code != 0 || result["project_path"] != filepath.ToSlash(project) {
		t.Fatalf("project: %d %s %s %v", code, stdout, stderr, err)
	}
}

func TestProcessLifecycleFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		run  engineproc.Result
		code string
	}{
		{engineproc.Result{ExitCode: 0}, ""},
		{engineproc.Result{ExitCode: -1, Abandoned: true}, "engine_abandoned"},
		{engineproc.Result{ExitCode: 0, CleanupPending: true}, "engine_cleanup_incomplete"},
	} {
		if got := runFailureCode(tc.run); got != tc.code {
			t.Fatalf("%+v: %q, want %q", tc.run, got, tc.code)
		}
	}
}
