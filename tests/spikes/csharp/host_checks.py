"""Compiler-host assertions without loading or executing the emitted assembly."""
import argparse
import base64
import json
from pathlib import Path
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument("--host", type=Path, required=True)
parser.add_argument("--engine", type=Path, required=True)
args = parser.parse_args()
api = args.engine.resolve().parent / "GodotSharp/Api/Debug"
refs = [str(api / "GodotSharp.dll"), str(api / "GodotSharpEditor.dll")]
with tempfile.TemporaryDirectory(prefix="hera-host-check-") as work:
    request = Path(work) / "request.json"
    for expression, valid in [("1+1", True), ("SceneRoot.Name", True), ('SceneRoot.SetMeta("test", 1)', True), ("1+", False), ("1; System.Console.WriteLine(2)", False), ("MissingSymbol", False), ("", False), ("x" * 65537, False)]:
        request.write_text(json.dumps({"expression": expression, "references": refs}))
        r = subprocess.run(["dotnet", str(args.host.resolve()), "--request", str(request)], capture_output=True, timeout=30)
        result = json.loads(r.stdout)
        assert result["ok"] == valid and r.returncode == (0 if valid else 1), result
        if valid:
            assert base64.b64decode(result["assembly"])[:2] == b"MZ", result
        else:
            assert result["diagnostics"] and "assembly" not in result, result
    request.write_text(json.dumps({"expression": "1", "references": refs, "project_reference": refs[0], "project_mvid": "wrong"}))
    r = subprocess.run(["dotnet", str(args.host.resolve()), "--request", str(request)], capture_output=True, timeout=30)
    assert r.returncode == 1 and "differs from the editor" in json.loads(r.stdout)["diagnostics"][0]
print("PASS compiler host: expression/void compilation, syntax rejection, bounds and MVID mismatch")
