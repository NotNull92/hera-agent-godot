"""Drive an isolated editor with the actual CLI, including its panel observations."""
import argparse
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
parser.add_argument("--cli", type=Path, required=True)
parser.add_argument("--legacy-cli", type=Path)
args = parser.parse_args()
repo = Path(__file__).resolve().parents[2]
work = Path(tempfile.mkdtemp(prefix="hera-v2-smoke-"))
print(f"Evidence: {work}", flush=True)
project = work / "project"
shutil.copytree(repo / "addons", project / "addons", ignore=shutil.ignore_patterns("*.import"))
shutil.copyfile(Path(__file__).with_name("editor_import_probe.gd"), project / "import_probe.gd")
(project / "Main.tscn").write_text('[gd_scene format=3]\n[node name="Main" type="Node2D"]\n')
(project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera isolated smoke"\nrun/main_scene="res://Main.tscn"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
env = os.environ.copy()
env["HERA_AGENT_GODOT_TOKEN"] = ""
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
    target = work / "home" / key.lower()
    target.mkdir(parents=True)
    env[key] = str(target)
engine = str(args.engine.resolve())
with (work / "import.log").open("wb") as log:
    r = subprocess.run([engine, "--headless", "--editor", "--path", str(project), "--script", "res://import_probe.gd"], env=env, stdout=log, stderr=subprocess.STDOUT, timeout=300)
assert r.returncode == 0
with (project / "project.godot").open("a") as settings:
    settings.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")\n')
with (work / "editor.log").open("wb") as log:
    editor = subprocess.Popen([engine, "--headless", "--editor", "--path", str(project), "res://Main.tscn"], env=env, stdout=log, stderr=subprocess.STDOUT)
    try:
        heartbeat = Path(env["USERPROFILE"]) / ".hera-agent-godot/instances" / f"{editor.pid}.json"
        for _ in range(150):
            if heartbeat.exists(): break
            assert editor.poll() is None
            time.sleep(0.2)
        assert heartbeat.exists()
        port = json.loads(heartbeat.read_text())["port"]
        def rpc(tool, params, version=""):
            headers = {"Content-Type": "application/json"}
            if version:
                headers["X-Hera-Cli-Version"] = version
            req = urllib.request.Request(f"http://127.0.0.1:{port}/rpc", data=json.dumps({"tool": tool, "params": params}).encode(), headers=headers)
            with urllib.request.urlopen(req, timeout=10) as response:
                return json.load(response)
        def call(name, arguments):
            r = subprocess.run([str(args.cli.resolve()), "--instance", str(editor.pid), "--timeout", "10000"] + arguments, cwd=project, env=env, capture_output=True, timeout=40)
            (work / (name + ".json")).write_bytes(r.stdout)
            (work / (name + ".stderr")).write_bytes(r.stderr)
            assert r.returncode == 0, (name, r.stdout, r.stderr)
            return json.loads(r.stdout)
        call("scene", ["scene", "open", "res://Main.tscn"])
        call("status", ["status"])
        call("smoke", ["smoke", "--skip-game"])
        assert call("eval", ["eval", "--lang", "gdscript", "1+1"])["result_value"] == 2
        expression = 'get_tree().root.find_child("HeraAgentMainScreen", true, false).rows.version.text'
        rpc("status", {}, "1.1.0")
        time.sleep(0.7)
        mismatch = rpc("eval", {"lang": "gdscript", "expr": expression}, "1.1.0")
        assert "Upgrade CLI and addon together" in mismatch["data"]["result_value"], mismatch
        (work / "version-mismatch.json").write_text(json.dumps(mismatch))
        if args.legacy_cli:
            legacy = subprocess.run([str(args.legacy_cli.resolve()), "--instance", str(editor.pid), "status"], cwd=project, env=env, capture_output=True, timeout=15)
            assert legacy.returncode == 0, legacy.stderr
            time.sleep(0.7)
            unknown = rpc("eval", {"lang": "gdscript", "expr": expression})
            assert "CLI version unknown" in unknown["data"]["result_value"], unknown
            (work / "version-legacy.json").write_text(json.dumps(unknown))
        call("diagnostics", ["diagnostics", "--lines", "20"])
        subprocess.run([str(args.cli.resolve()), "--instance", str(editor.pid), "eval", "--lang", "gdscript", "get_tree().quit()"], cwd=project, env=env, capture_output=True, timeout=15)
        assert editor.wait(timeout=15) == 0
        print("PASS isolated editor: status, smoke, GDScript eval, diagnostics", flush=True)
    finally:
        if editor.poll() is None:
            editor.kill()
            editor.wait(timeout=10)
