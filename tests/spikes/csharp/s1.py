"""Disposable C# editor/ALC spike; never opens or modifies a user project."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
args = parser.parse_args()
engine = args.engine.resolve()
work = Path(tempfile.mkdtemp(prefix="hera-csharp-s1-"))
print(f"Evidence: {work}", flush=True)
project = work / "project"
plugin = project / "addons" / "probe"
plugin.mkdir(parents=True)
home = work / "home"
home.mkdir()
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "DOTNET_CLI_HOME"):
    path = home / key.lower()
    path.mkdir()
    env[key] = str(path)
env.update(HERA_SPIKE_DIRECTORY=str(work), DOTNET_SKIP_FIRST_TIME_EXPERIENCE="1", DOTNET_GENERATE_ASPNET_CERTIFICATE="false", DOTNET_CLI_TELEMETRY_OPTOUT="1", MSBuildEnableWorkloadResolver="false")


def run(command, name, cwd=project):
    with (work / (name + ".log")).open("w", encoding="utf-8") as log:
        result = subprocess.run([str(x) for x in command], cwd=cwd, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=120)
    if result.returncode:
        raise RuntimeError(f"{name}: exit {result.returncode}; see {work / (name + '.log')}")


def wait_for(name, process):
    deadline = time.monotonic() + 40
    while time.monotonic() < deadline:
        if (work / "failed").exists():
            raise RuntimeError((work / "failed").read_text())
        if (work / name).exists():
            print("PASS " + name, flush=True)
            return
        if process.poll() is not None:
            raise RuntimeError(f"Editor exited {process.returncode} before {name}")
        time.sleep(0.2)
    raise TimeoutError(name)


feed = engine.parent / "GodotSharp" / "Tools" / "nupkgs"
(project / "NuGet.Config").write_text(f'<configuration><packageSources><clear/><add key="godot" value="{feed}"/></packageSources></configuration>')
(project / "Spike.csproj").write_text('<Project Sdk="Godot.NET.Sdk/4.7.2"><PropertyGroup><TargetFramework>net10.0</TargetFramework><EnableDynamicLoading>true</EnableDynamicLoading><Nullable>enable</Nullable></PropertyGroup></Project>')
(project / "project.godot").write_text('config_version=5\n[application]\nconfig/name="Hera CSharp S1"\n[dotnet]\nproject/assembly_name="Spike"\n[rendering]\nrenderer/rendering_method="gl_compatibility"\n')
shutil.copyfile(Path(__file__).with_name("ProbePlugin.cs"), plugin / "ProbePlugin.cs")
(plugin / "plugin.cfg").write_text('[plugin]\nname="CSharp spike"\ndescription="Disposable probe"\nauthor="Hera"\nversion="0"\nscript="ProbePlugin.cs"\n')
stamp = project / "ProbeStamp.cs"
stamp.write_text('public static class ProbeStamp { public static int Value => 1; }')
run(["dotnet", "build", "Spike.csproj", "--nologo"], "build-1")
expression = work / "expression"
expression.mkdir()
api = engine.parent / "GodotSharp" / "Api" / "Debug"
refs = {"GodotSharp": api / "GodotSharp.dll", "GodotSharpEditor": api / "GodotSharpEditor.dll", "Spike": project / ".godot/mono/temp/bin/Debug/Spike.dll"}
items = ''.join(f'<Reference Include="{n}"><HintPath>{p}</HintPath><Private>false</Private></Reference>' for n, p in refs.items())
(expression / "Expression.csproj").write_text('<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup><ItemGroup>' + items + '</ItemGroup></Project>')
(expression / "Expression.cs").write_text('using Godot; public static class Expression { public static object Run(Node SceneRoot) => ProbeStamp.Value > 0 && EditorInterface.Singleton != null ? ((Node2D)SceneRoot).Position : Vector2.Zero; }')
shutil.copyfile(project / "NuGet.Config", expression / "NuGet.Config")
run(["dotnet", "build", "Expression.csproj", "--nologo"], "expression-build", expression)
shutil.copyfile(expression / "bin/Debug/net10.0/Expression.dll", work / "expression.dll")
# Import before enabling the C# plugin, so Godot knows its script resource.
run([engine, "--headless", "--editor", "--path", project, "--import"], "import")
with (project / "project.godot").open("a") as f:
    f.write('\n[editor_plugins]\nenabled=PackedStringArray("res://addons/probe/plugin.cfg")\n')
with (work / "editor.log").open("w", encoding="utf-8") as log:
    editor = subprocess.Popen([str(engine), "--headless", "--editor", "--path", str(project)], env=env, stdout=log, stderr=subprocess.STDOUT)
    try:
        wait_for("loaded-1", editor)
        (work / "eval-request").touch()
        wait_for("unloaded-1", editor)
        stamp.write_text('public static class ProbeStamp { public static int Value => 2; }')
        run(["dotnet", "build", "Spike.csproj", "--nologo"], "build-2")
        wait_for("loaded-2", editor)
        (work / "eval-request").touch()
        wait_for("unloaded-2", editor)
        (work / "quit").touch()
        assert editor.wait(timeout=20) == 0
        print("PASS S1: native C# loader, engine/scene API, ALC collection and same-editor rebuild/reload", flush=True)
    finally:
        if editor.poll() is None:
            editor.kill()
            editor.wait(timeout=10)
