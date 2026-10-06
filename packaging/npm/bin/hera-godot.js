#!/usr/bin/env node
"use strict";
const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const root = path.join(__dirname, "..");
// Published archives retain their existing internal filename and checksum.
const binary = path.join(root, "vendor", process.platform === "win32" ? "hera.exe" : "hera");
if (!fs.existsSync(binary)) {
  const setup = spawnSync(process.execPath, [path.join(root, "scripts", "install.js")], { stdio: "inherit" });
  if (setup.status !== 0 || !fs.existsSync(binary)) process.exit(setup.status || 1);
}
const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) console.error(`hera-godot: ${result.error.message}`);
process.exit(result.status === null ? 1 : result.status);
