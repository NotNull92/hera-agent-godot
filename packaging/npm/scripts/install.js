#!/usr/bin/env node
// Postinstall: download the pinned hera release binary for this platform,
// verify its SHA-256 against manifest.json, and unpack it into vendor/.
// Pure Node (>=18), no dependencies. Extraction uses the system tar, which
// ships with macOS, Linux, and Windows 10+ (bsdtar handles .zip too).
"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const pkgRoot = path.join(__dirname, "..");
const manifest = require(path.join(pkgRoot, "manifest.json"));
const vendorDir = path.join(pkgRoot, "vendor");
const binName = process.platform === "win32" ? "hera.exe" : "hera";
const binPath = path.join(vendorDir, binName);

function fail(msg) {
  console.error(`hera-godot: ${msg}`);
  console.error(
    "hera-godot: you can install the CLI without npm instead: " +
      "https://github.com/NotNull92/hera-agent-godot#installation"
  );
  process.exit(1);
}

function download(url, dest, redirectsLeft = 5) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "hera-godot-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirectsLeft === 0) return reject(new Error("too many redirects"));
          return resolve(download(res.headers.location, dest, redirectsLeft - 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const out = fs.createWriteStream(dest);
        res.pipe(out);
        out.on("finish", () => out.close(resolve));
        out.on("error", reject);
      })
      .on("error", reject);
  });
}

function sha256(file) {
  return crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");
}

function extract(archive, destDir) {
  // bsdtar on Windows 10+ extracts .zip; GNU/BSD tar handles .tar.gz.
  const args = archive.endsWith(".zip") ? ["-xf", archive, "-C", destDir] : ["-xzf", archive, "-C", destDir];
  const tar = spawnSync("tar", args, { stdio: "pipe" });
  if (tar.status === 0) return;
  if (process.platform === "win32") {
    const ps = spawnSync(
      "powershell.exe",
      ["-NoProfile", "-Command", `Expand-Archive -LiteralPath '${archive}' -DestinationPath '${destDir}' -Force`],
      { stdio: "pipe" }
    );
    if (ps.status === 0) return;
  }
  throw new Error(`could not extract ${path.basename(archive)} (is tar available?)`);
}

async function main() {
  const key = `${process.platform}-${process.arch}`;
  const asset = manifest.assets[key];
  if (!asset) fail(`unsupported platform: ${key}`);

  if (fs.existsSync(binPath)) {
    const probe = spawnSync(binPath, ["version"], { stdio: "pipe" });
    if (probe.status === 0) return; // already installed and working
  }

  const url = `https://github.com/${manifest.repo}/releases/download/v${manifest.version}/${asset.name}`;
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "hera-godot-"));
  const archive = path.join(tmpDir, asset.name);
  try {
    console.log(`hera-godot: downloading ${url}`);
    await download(url, archive);
    const actual = sha256(archive);
    if (actual !== asset.sha256) {
      fail(`SHA-256 mismatch for ${asset.name}: expected ${asset.sha256}, got ${actual}`);
    }
    fs.mkdirSync(vendorDir, { recursive: true });
    extract(archive, vendorDir);
    if (!fs.existsSync(binPath)) fail(`archive did not contain ${binName}`);
    if (process.platform !== "win32") fs.chmodSync(binPath, 0o755);
    console.log(`hera-godot: installed hera v${manifest.version}`);
    console.log(
      "hera-godot: the CLI drives the Hera Agent Godot editor addon. Install the addon " +
        "from the Godot Asset Store (https://store.godotengine.org/asset/notnull92/hera-agent-godot/) " +
        "and enable it under Project Settings > Plugins."
    );
  } catch (err) {
    fail(err && err.message ? err.message : String(err));
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

main();
