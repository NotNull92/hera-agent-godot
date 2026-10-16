"""Standard-editor load and 4.2 theme spike, in an isolated addon copy."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument("--standard", type=Path, required=True)
parser.add_argument("--floor", type=Path, required=True)
args = parser.parse_args()
work = Path(tempfile.mkdtemp(prefix="hera-csharp-s4s5-"))
print(f"Evidence: {work}", flush=True)
repo = Path(__file__).resolve().parents[3]
shutil.copytree(repo / "addons/hera_agent_godot", work / "addons/hera_agent_godot")
shutil.copyfile(Path(__file__).with_name("ProbePlugin.cs"), work / "addons/hera_agent_godot/ProbePlugin.cs")
shutil.copyfile(Path(__file__).with_name("theme_probe.gd"), work / "theme_probe.gd")
(work / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera S4S5"\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
    path = work / "home" / key.lower()
    path.mkdir(parents=True)
    env[key] = str(path)


def run(engine, name, extra, marker):
    result = subprocess.run([str(engine.resolve()), "--headless", "--editor", "--path", str(work)] + extra,
                            env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=60)
    output = result.stdout.decode("utf-8", errors="replace")
    (work / (name + ".log")).write_text(output, encoding="utf-8")
    assert result.returncode == 0 and marker in output and "SCRIPT ERROR" not in output, output
    print("PASS " + name, flush=True)


run(args.standard, "s4-standard", ["--quit-after", "240"], "[hera] Hera Agent Godot listening")
run(args.floor, "s5-check", ["--check-only", "--script", "res://theme_probe.gd"], "Godot Engine")
run(args.floor, "s5-theme", ["--script", "res://theme_probe.gd"], "HERA_THEME_PROBE_OK")
