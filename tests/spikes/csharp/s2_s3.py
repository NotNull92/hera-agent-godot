"""Continue only in a disposable S1 project: isolated build and direct C# run."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("--engine", type=Path, required=True)
parser.add_argument("--evidence", type=Path, required=True)
args = parser.parse_args()
work = args.evidence.resolve()
assert work.name.startswith("hera-csharp-s1-") and (work / "unloaded-2").is_file()
project = work / "project"
env = os.environ.copy()
for key in ("HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "DOTNET_CLI_HOME"):
    env[key] = str(work / "home" / key.lower())
env.update(DOTNET_GENERATE_ASPNET_CERTIFICATE="false", DOTNET_CLI_TELEMETRY_OPTOUT="1", MSBuildEnableWorkloadResolver="false")


def run(command, name):
    with (work / (name + ".log")).open("w", encoding="utf-8") as log:
        result = subprocess.run([str(x) for x in command], cwd=project, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=120)
    return result.returncode


def snapshot():
    return {str(p.relative_to(project)): (hashlib.sha256(p.read_bytes()).hexdigest(), p.stat().st_mtime_ns)
            for p in (project / ".godot/mono/temp").rglob("*") if p.is_file()}


before = snapshot()
staging = work / "isolated-build"
staging.mkdir(exist_ok=True)
result = run(["dotnet", "build", "Spike.csproj", "--nologo",
              f"-p:BaseOutputPath={staging / 'bin'}/",
              f"-p:OutputPath={staging / 'bin' / 'Debug'}/",
              f"-p:BaseIntermediateOutputPath={staging / 'obj'}/",
              f"-p:IntermediateOutputPath={staging / 'obj' / 'Debug'}/",
              f"-p:MSBuildProjectExtensionsPath={staging / 'obj'}/"], "s2-build")
after = snapshot()
s2 = {"exit_code": result, "editor_outputs_unchanged": before == after,
      "staged_assembly": [str(p.relative_to(staging)) for p in staging.rglob("Spike.dll")]}
(work / "s2.json").write_text(json.dumps(s2, indent=2))
assert result == 0 and before == after and s2["staged_assembly"], s2
print("PASS S2: build in redirected bin/obj; editor output hashes and mtimes unchanged", flush=True)

(project / "Tool.cs").write_text('''using Godot;
public partial class Tool : SceneTree
{
    public override void _Initialize()
    {
        GD.Print("HERA_CSHARP_DIRECT_RUN_OK:", string.Join(",", OS.GetCmdlineUserArgs()));
        Quit(0);
    }
}
''')
assert run(["dotnet", "build", "Spike.csproj", "--nologo"], "s3-build") == 0
result = run([args.engine.resolve(), "--headless", "--path", project,
              "--script", "res://Tool.cs", "--", "one", "two"], "s3-run")
output = (work / "s3-run.log").read_text(encoding="utf-8", errors="replace")
s3 = {"exit_code": result, "marker_present": "HERA_CSHARP_DIRECT_RUN_OK:one,two" in output}
(work / "s3.json").write_text(json.dumps(s3, indent=2))
assert result == 0 and s3["marker_present"], s3
print("PASS S3: direct SceneTree C# --script and user args", flush=True)
