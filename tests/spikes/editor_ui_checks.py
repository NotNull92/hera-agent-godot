"""Isolated editor checks and import timing for W2; never opens a user project."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
parser.add_argument("--all-scripts", action="store_true")
args = parser.parse_args()
repo = Path(__file__).resolve().parents[2]
work = Path(tempfile.mkdtemp(prefix="hera-w2-check-"))
print(f"Evidence: {work}", flush=True)
project = work / "project"
shutil.copytree(repo / "addons", project / "addons", ignore=shutil.ignore_patterns("*.import"))
(project / "tests/headless").mkdir(parents=True)
shutil.copyfile(Path(__file__).with_name("editor_import_probe.gd"), project / "import_probe.gd")
for name in ("main_screen_panel_test.gd", "http_server_test.gd", "eval_result_test.gd"):
    shutil.copyfile(repo / "tests/headless" / name, project / "tests/headless" / name)
(project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera W2 checks"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
env = os.environ.copy()
env["HERA_AGENT_GODOT_TOKEN"] = ""
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
    target = work / "home" / key.lower()
    target.mkdir(parents=True)
    env[key] = str(target)
results = []

def run(name, extra, marker="", timeout=60):
    start = time.monotonic()
    with (work / (name + ".log")).open("wb") as log:
        result = subprocess.run([str(args.engine.resolve()), "--headless", "--path", str(project)] + extra, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=timeout)
    output = (work / (name + ".log")).read_text(encoding="utf-8", errors="replace")
    elapsed = round(time.monotonic() - start, 3)
    results.append({"name": name, "exit": result.returncode, "seconds": elapsed})
    (work / "results.json").write_text(json.dumps(results, indent=2))
    assert result.returncode == 0 and marker in output and "SCRIPT ERROR" not in output, output
    print(f"PASS {name} ({elapsed}s)", flush=True)

scripts = sorted((project / "addons").rglob("*.gd")) if args.all_scripts else [project / "addons/hera_agent_godot/core/main_screen_panel.gd", project / "addons/hera_agent_godot/hera_agent_plugin.gd"]
for script in scripts:
    run("check-" + script.stem, ["--check-only", "--script", "res://" + script.relative_to(project).as_posix()])
help_text = subprocess.check_output([str(args.engine.resolve()), "--help"], env=env).decode("utf-8", errors="replace")
import_flags = ["--import"] if "--import" in help_text else ["--script", "res://import_probe.gd"]
run("import", ["--editor"] + import_flags, "" if "--import" in help_text else "HERA_IMPORT_OK", timeout=300)
for name in ("main_screen_panel_test", "http_server_test", "eval_result_test"):
    mode = ["--editor"] if name != "http_server_test" else []
    run(name, mode + ["--script", "res://tests/headless/" + name + ".gd"], "PASS")
with (project / "project.godot").open("a") as settings:
    settings.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")\n')
run("plugin-load", ["--editor", "--script", "res://import_probe.gd"], "Hera Agent Godot listening")
