"""Capture an owned GUI editor after applying its real theme and display scale."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
parser.add_argument("--out", type=Path, required=True)
parser.add_argument("--theme", choices=["Default", "Light"], default="Default")
parser.add_argument("--scale", choices=["1", "1.5"], default="1")
args = parser.parse_args()
repo = Path(__file__).resolve().parents[2]
work = Path(tempfile.mkdtemp(prefix="hera-w2-visual-"))
print(f"Evidence: {work}", flush=True)
project = work / "project"
shutil.copytree(repo / "addons", project / "addons", ignore=shutil.ignore_patterns("*.import"))
(project / "docs/assets").mkdir(parents=True)
shutil.copyfile(repo / "docs/assets/hera-pointing.png", project / "docs/assets/hera-pointing.png")
(project / "addons/capture").mkdir()
shutil.copyfile(Path(__file__).with_suffix(".gd"), project / "addons/capture/capture.gd")
(project / "addons/capture/plugin.cfg").write_text('[plugin]\nname="Capture"\ndescription="Owned UI QA"\nauthor="Hera"\nversion="1"\nscript="capture.gd"\n')
shutil.copyfile(Path(__file__).with_name("editor_import_probe.gd"), project / "import_probe.gd")
(project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera UI review"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"):
    target = work / "home" / key.lower()
    target.mkdir(parents=True)
    env[key] = str(target)
env.update(HERA_AGENT_GODOT_TOKEN="", HERA_UI_THEME=args.theme, HERA_UI_SCALE=args.scale, HERA_UI_SCREENSHOT=str(args.out.resolve()))
args.out.parent.mkdir(parents=True, exist_ok=True)

def run(name, extra, timeout=90):
    with (work / (name + ".log")).open("wb") as log:
        result = subprocess.run([str(args.engine.resolve()), "--editor", "--path", str(project)] + extra, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=timeout)
    output = (work / (name + ".log")).read_text(encoding="utf-8", errors="replace")
    assert result.returncode == 0 and "SCRIPT ERROR" not in output, output
    print(name + ": " + output[-350:], flush=True)

help_text = subprocess.check_output([str(args.engine.resolve()), "--help"], env=env).decode("utf-8", errors="replace")
run("import", ["--headless"] + (["--import"] if "--import" in help_text else ["--script", "res://import_probe.gd"]), 300)
with (project / "project.godot").open("a") as settings:
    settings.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg", "res://addons/capture/plugin.cfg")\n')
env["HERA_UI_PHASE"] = "configure"
run("configure", ["--headless"])
env["HERA_UI_PHASE"] = "capture"
run("capture", ["--resolution", "1600x1200"])
assert args.out.exists()
