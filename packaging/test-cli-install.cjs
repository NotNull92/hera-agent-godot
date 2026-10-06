"use strict";
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const root = path.resolve(__dirname, "..");
const temp = fs.mkdtempSync(path.join(os.tmpdir(), "hera-command-test-"));
const run = (command, args, cwd = temp) => {
  const result = spawnSync(command, args, { cwd, encoding: "utf8", windowsHide: true });
  assert.equal(result.status, 0, `${command}: ${result.error || result.stderr || result.stdout}`);
  return result.stdout;
};
try {
  assert.ok(process.env.npm_execpath, "Run through npm test in packaging/npm");
  const npm = args => run(process.execPath, [process.env.npm_execpath, ...args]);
  const prefix = path.join(temp, "prefix 한글 space");
  const bins = path.join(prefix, "node_modules", ".bin");
  fs.mkdirSync(bins, { recursive: true });
  const sentinel = path.join(bins, process.platform === "win32" ? "hera.cmd" : "hera");
  fs.writeFileSync(sentinel, "unrelated hera command\n");
  const packed = JSON.parse(npm(["pack", path.join(root, "packaging", "npm"), "--json", "--ignore-scripts", "--pack-destination", temp]))[0];
  npm(["install", "--prefix", prefix, "--ignore-scripts", "--no-audit", "--no-fund", path.join(temp, packed.filename)]);
  assert.equal(fs.readFileSync(sentinel, "utf8"), "unrelated hera command\n");
  const installed = path.join(prefix, "node_modules", "hera-godot");
  const manifest = JSON.parse(fs.readFileSync(path.join(installed, "package.json"), "utf8"));
  assert.deepEqual(Object.keys(manifest.bin).sort(), ["hera-agent-godot", "hera-godot"]);
  const vendor = path.join(installed, "vendor");
  fs.mkdirSync(vendor);
  run("go", ["build", "-o", path.join(vendor, process.platform === "win32" ? "hera.exe" : "hera"), "."], root);
  for (const name of Object.keys(manifest.bin)) {
    const shim = path.join(bins, name + (process.platform === "win32" ? ".cmd" : ""));
    let output;
    if (process.platform === "win32") {
      const script = path.join(temp, `${name}-check.ps1`);
      fs.writeFileSync(script, "\ufeff& '" + shim.replaceAll("'", "''") + "' --help\nexit $LASTEXITCODE\n");
      output = run("powershell.exe", ["-NoProfile", "-File", script]);
    } else output = run(shim, ["--help"]);
    assert.match(output, /^hera-godot —/);
    assert.match(output, /usage: hera-godot /);
  }
  console.log(JSON.stringify({platform:process.platform,pack:"pass",install:"pass",aliases:"pass",existingHera:"preserved",liveEditor:"not_run"}));
} finally {
  assert.equal(path.dirname(path.resolve(temp)), path.resolve(os.tmpdir()));
  assert.ok(path.basename(temp).startsWith("hera-command-test-"));
  fs.rmSync(temp, { recursive: true, force: true });
}
