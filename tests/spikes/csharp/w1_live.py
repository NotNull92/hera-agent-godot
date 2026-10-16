"""Owned editor integration for the in-progress C# W1 implementation."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
import urllib.request

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
parser.add_argument("--host", type=Path, required=True)
args = parser.parse_args()
repo = Path(__file__).resolve().parents[3]
work = Path(tempfile.mkdtemp(prefix="hera-csharp-w1-"))
print(f"Evidence: {work}", flush=True)
project = work / "project"
shutil.copytree(repo / "addons", project / "addons")
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "DOTNET_CLI_HOME"):
    target = work / "home" / key.lower()
    target.mkdir(parents=True)
    env[key] = str(target)
env.update(HERA_AGENT_GODOT_TOKEN="", DOTNET_GENERATE_ASPNET_CERTIFICATE="false", MSBuildEnableWorkloadResolver="false", HERA_GODOT_CSHARP_HOST=str(args.host.resolve()))
env.pop("HERA_GODOT_EVAL_LANG", None)
engine = args.engine.resolve()
feed = engine.parent / "GodotSharp/Tools/nupkgs"
(project / "NuGet.Config").write_text(f'<configuration><packageSources><clear/><add key="godot" value="{feed}"/></packageSources></configuration>')
(project / "Probe.csproj").write_text('<Project Sdk="Godot.NET.Sdk/4.7.2"><PropertyGroup><TargetFramework>net10.0</TargetFramework><EnableDynamicLoading>true</EnableDynamicLoading><TreatWarningsAsErrors>true</TreatWarningsAsErrors></PropertyGroup></Project>')
(project / "Probe.cs").write_text('using Godot; public partial class Probe : Node2D { public int CustomValue => 123; }')
(project / "Stamp.cs").write_text('public static class Stamp { public static int Value => 1; }')
(project / "Tool.cs").write_text('using Godot; public partial class Tool : SceneTree { public override void _Initialize() { GD.Print("W1_RUN:", string.Join(",", OS.GetCmdlineUserArgs())); Quit(); } }')
(project / "Main.tscn").write_text('[gd_scene load_steps=2 format=3]\n[ext_resource type="Script" path="res://Probe.cs" id="1"]\n[node name="Main" type="Node2D"]\nposition=Vector2(12,34)\nscript=ExtResource("1")\n')
(project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera W1"\nrun/main_scene="res://Main.tscn"\n[dotnet]\nproject/assembly_name="Probe"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')


def command(argv, name, cwd=project, timeout=120):
    r = subprocess.run([str(x) for x in argv], cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout)
    (work / (name + ".log")).write_bytes(r.stdout)
    assert r.returncode == 0, r.stdout.decode("utf-8", errors="replace")
    return r.stdout


cli = work / "hera.exe"
command(["go", "build", "-o", cli, "."], "cli-build", repo)
command(["dotnet", "build", "Probe.csproj", "--nologo"], "project-build")
command([engine, "--headless", "--editor", "--path", project, "--import"], "import")
with (project / "project.godot").open("a") as f:
    f.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")\n')


def call(arguments, expected=0):
    deadline = [] if arguments[:2] == ["script", "validate"] else ["--timeout", "60000"]
    r = subprocess.run([str(cli), "--instance", str(editor.pid)] + deadline + arguments,
                       cwd=project, env=env, capture_output=True, timeout=80)
    with (work / "calls.jsonl").open("a", encoding="utf-8") as f:
        f.write(json.dumps({"args": arguments, "exit": r.returncode, "stdout": r.stdout.decode("utf-8", errors="replace"), "stderr": r.stderr.decode("utf-8", errors="replace")}) + "\n")
    assert r.returncode == expected, (arguments, r.stdout, r.stderr)
    return json.loads(r.stdout) if r.stdout else None


with (work / "editor.log").open("w", encoding="utf-8") as log:
    editor = subprocess.Popen([str(engine), "--headless", "--editor", "--path", str(project), "res://Main.tscn"], env=env, stdout=log, stderr=subprocess.STDOUT)
    try:
        heartbeat = Path(env["USERPROFILE"]) / ".hera-agent-godot/instances" / (str(editor.pid) + ".json")
        for _ in range(150):
            if heartbeat.exists(): break
            assert editor.poll() is None, "editor exited"
            time.sleep(0.2)
        assert heartbeat.exists(), "editor not ready"
        call(["scene", "open", "res://Main.tscn"])
        endpoint = json.loads(heartbeat.read_text())["port"]
        request = urllib.request.Request(f"http://127.0.0.1:{endpoint}/rpc", data=json.dumps({"tool": "eval", "params": {"lang": "csharp", "action": "csharp-context"}}).encode(), headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(request) as response:
            (work / "context.json").write_bytes(response.read())
        assert call(["eval", "1+1"])["result_value"] == 2
        assert call(["eval", "--lang", "gdscript", "1+1"])["result_value"] == 2
        assert call(["eval", "((Node2D)SceneRoot).Position"])["result_value"] == "Vector2(12, 34)"
        assert call(["eval", "SceneRoot is Probe"])["result_value"] is False
        assert call(["eval", "new Godot.Collections.Dictionary { [\"a\"] = 2 }"])["result_value"] == {"a": 2}
        assert call(["eval", "double.NaN"])["result_value"] == "nan"
        assert call(["eval", "9007199254740993L"])["result_value"] == "9007199254740993"
        assert call(["eval", 'SceneRoot.SetMeta("w1", 42)'])["result_value"] is None
        assert call(["eval", 'SceneRoot.GetMeta("w1")'])["result_value"] == 42
        assert call(["eval", "1+"] , 1)["error"] == "parse_error"
        call(["eval", '((Probe)SceneRoot).CustomValue'], 1)
        batch = work / "batch.json"
        batch.write_text(json.dumps([{"tool": "eval", "params": {"expr": "3+4"}}, {"tool": "eval", "params": {"expr": "1+2", "lang": "gdscript"}}]))
        result = call(["batch", "--file", str(batch)])
        assert "7" in json.dumps(result), result
        env["HERA_GODOT_EVAL_LANG"] = "gdscript"
        assert call(["eval", "Vector2(1,2)"])["result_value"] == "Vector2(1, 2)"
        del env["HERA_GODOT_EVAL_LANG"]
        assert call(["script", "validate", "res://Probe.cs"])["valid"]
        result = call(["--engine", str(engine), "script", "run", "res://Tool.cs", "--", "one", "two"])
        assert "W1_RUN:one,two" in result["output"]
        (project / "Stamp.cs").write_text('public static class Stamp { public static int Value => 2; }')
        command(["dotnet", "build", "Probe.csproj", "--nologo"], "rebuild")
        time.sleep(2)
        assert call(["eval", "Stamp.Value"])["result_value"] == 2
        # Quit closes the HTTP connection before its queued reply is flushed.
        call(["eval", "SceneRoot.GetTree().Quit()"], 1)
        assert editor.wait(timeout=20) == 0
        print("PASS W1: C# eval/validate/run, GDScript opt-in, batch, placeholder boundary, same-editor reload", flush=True)
    finally:
        if editor.poll() is None:
            editor.kill()
            editor.wait(timeout=10)
