"""Check W1 dispatch on standard editors without touching a user project."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, action="append", required=True)
args = parser.parse_args()
repo = Path(__file__).resolve().parents[3]
work = Path(tempfile.mkdtemp(prefix="hera-w1-standard-"))
print(f"Evidence: {work}", flush=True)
env = os.environ.copy()
env["HERA_AGENT_GODOT_TOKEN"] = ""
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
    target = work / "home" / key.lower()
    target.mkdir(parents=True)
    env[key] = str(target)
for index, engine in enumerate(args.engine):
    for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
        target = work / f"home-{index}" / key.lower()
        target.mkdir(parents=True)
        env[key] = str(target)
    project = work / str(index)
    shutil.copytree(repo / "addons", project / "addons", ignore=shutil.ignore_patterns("*.import"))
    (project / "tests/headless").mkdir(parents=True)
    shutil.copyfile(repo / "tests/headless/eval_result_test.gd", project / "tests/headless/eval_result_test.gd")
    (project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="W1 standard checks"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
    try:
        imported = subprocess.run([str(engine.resolve()), "--headless", "--editor", "--path", str(project), "--import"], env=env, capture_output=True, timeout=60)
    except subprocess.TimeoutExpired as error:
        (work / f"{index}-import.log").write_bytes((error.stdout or b"") + (error.stderr or b""))
        raise
    (work / f"{index}-import.log").write_bytes(imported.stdout + imported.stderr)
    assert imported.returncode == 0, imported.stderr
    with (project / "project.godot").open("a") as settings:
        settings.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")\n')
    for name, extra, marker in [
        ("eval-check", ["--check-only", "--script", "res://addons/hera_agent_godot/tools/eval_tool.gd"], "Godot Engine"),
        ("eval-test-check", ["--check-only", "--script", "res://tests/headless/eval_result_test.gd"], "Godot Engine"),
        ("eval-test", ["--script", "res://tests/headless/eval_result_test.gd"], "eval_result_test: PASS"),
        ("plugin-load", ["--quit-after", "180"], "[hera] Hera Agent Godot listening"),
    ]:
        result = subprocess.run([str(engine.resolve()), "--headless", "--editor", "--path", str(project)] + extra,
                                env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=60)
        output = result.stdout.decode("utf-8", errors="replace")
        (work / f"{index}-{name}.log").write_text(output, encoding="utf-8")
        assert result.returncode == 0 and marker in output and "SCRIPT ERROR" not in output, output
        print(f"PASS {index}: {name}", flush=True)
