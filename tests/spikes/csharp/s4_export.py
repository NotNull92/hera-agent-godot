"""Export the disposable S1 C# project and inspect exported type metadata."""
import argparse
import os
from pathlib import Path
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
parser.add_argument("--evidence", type=Path, required=True)
args = parser.parse_args()
work = args.evidence.resolve()
is_w1 = work.name.startswith("hera-csharp-w1-") and (work / "calls.jsonl").exists()
assert is_w1 or (work.name.startswith("hera-csharp-s1-") and (work / "unloaded-2").exists())
assembly_name = "Probe" if is_w1 else "Spike"
project = work / "project"
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "DOTNET_CLI_HOME"):
    env[key] = str(work / "home" / key.lower())
env.update(DOTNET_GENERATE_ASPNET_CERTIFICATE="false", MSBuildEnableWorkloadResolver="false", HERA_SPIKE_DIRECTORY=str(work))


def run(command, name, cwd=project):
    result = subprocess.run([str(x) for x in command], cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180)
    text = result.stdout.decode("utf-8", errors="replace")
    (work / (name + ".log")).write_text(text, encoding="utf-8")
    assert result.returncode == 0, text
    return text


# Only the disposable fixture gets build files; production must never create them.
if not (project / (assembly_name + ".sln")).exists():
    run(["dotnet", "new", "sln", "--format", "sln", "--name", assembly_name], "s4-solution")
    run(["dotnet", "sln", assembly_name + ".sln", "add", assembly_name + ".csproj"], "s4-solution-add")
(project / "Main.tscn").write_text('[gd_scene format=3]\n[node name="Main" type="Node"]\n')
settings = (project / "project.godot").read_text()
if 'run/main_scene=' not in settings:
    settings = settings.replace('[application]', '[application]\nrun/main_scene="res://Main.tscn"')
(project / "project.godot").write_text(settings)
(project / "export_presets.cfg").write_text('[preset.0]\nname="Windows"\nplatform="Windows Desktop"\nrunnable=true\nexport_filter="all_resources"\ninclude_filter=""\nexclude_filter=""\nexport_path=""\nscript_export_mode=2\n[preset.0.options]\nbinary_format/architecture="x86_64"\ndotnet/include_scripts_content=false\n')
config = project / "NuGet.Config"
text = config.read_text()
if "api.nuget.org" not in text:
    config.write_text(text.replace('</packageSources>', '<add key="nuget" value="https://api.nuget.org/v3/index.json"/></packageSources>'))
output = run([args.engine.resolve(), "--headless", "--path", project, "--export-pack", "Windows", work / "probe-final.pck"], "s4-export-final")
assert "ERROR:" not in output, output
assemblies = list((project / ".godot/mono/temp/bin/ExportRelease").rglob(assembly_name + ".dll"))
assert assemblies, "No export-compiled assembly"
inspector = work / "inspect"
inspector.mkdir(exist_ok=True)
(inspector / "Inspect.csproj").write_text('<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net10.0</TargetFramework><OutputType>Exe</OutputType></PropertyGroup></Project>')
(inspector / "Program.cs").write_text('''using System;
using System.IO;
using System.Reflection.Metadata;
using System.Reflection.PortableExecutable;
using var stream = File.OpenRead(args[0]);
using var pe = new PEReader(stream);
var metadata = pe.GetMetadataReader();
bool control = false;
foreach (var handle in metadata.TypeDefinitions) {
    var name = metadata.GetString(metadata.GetTypeDefinition(handle).Name);
    if (name.Contains("ProbePlugin") || name.Contains("HeraCSharpEval") || name.Contains("EvaluationContext")) throw new Exception("TOOLS loader leaked into export");
    if (name == "ProbeStamp" || name == "Stamp") control = true;
}
if (!control) throw new Exception("Wrong assembly: missing control type");
Console.WriteLine("HERA_EXPORT_TYPES_OK");
''')
for assembly in assemblies:
    output = run(["dotnet", "run", "--project", inspector / "Inspect.csproj", "--", assembly], "s4-types", inspector)
    assert "HERA_EXPORT_TYPES_OK" in output, output
print("PASS S4 export: C# PCK export, ExportRelease assembly excludes TOOLS loader types", flush=True)
