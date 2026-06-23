# Install the hera-agent-godot CLI on Windows.
#
#   irm https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.ps1 | iex
#
# Environment overrides:
#   HERA_VERSION   release tag to install (default: latest)
#   HERA_BIN_DIR   install directory     (default: %LOCALAPPDATA%\hera\bin)
#
# This installs the CLI only. The Godot addon is a separate drop-in folder —
# grab hera-agent-godot-addon.zip from the release and unzip it into your
# project root (creating <project>\addons\hera_agent_godot).
$ErrorActionPreference = 'Stop'

$Repo = 'NotNull92/hera-agent-godot'
$Version = if ($env:HERA_VERSION) { $env:HERA_VERSION } else { 'latest' }
$BinDir = if ($env:HERA_BIN_DIR) { $env:HERA_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'hera\bin' }
$Architecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }

switch ($Architecture) {
  'AMD64' { $arch = 'amd64' }
  'ARM64' { $arch = 'arm64' }
  default { throw "hera: unsupported architecture: $Architecture" }
}

$asset = "hera-windows-$arch.zip"
$url = if ($Version -eq 'latest') {
  "https://github.com/$Repo/releases/latest/download/$asset"
} else {
  "https://github.com/$Repo/releases/download/$Version/$asset"
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("hera-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
  $zip = Join-Path $tmp 'hera.zip'
  Write-Host "hera: downloading $url"
  Invoke-WebRequest -Uri $url -OutFile $zip
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
  Expand-Archive -Path $zip -DestinationPath $BinDir -Force
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$exe = Join-Path $BinDir 'hera.exe'
Write-Host "hera: installed to $exe"
try { Write-Host "hera: version $(& $exe version)" } catch {}

# Add BinDir to the user PATH if it is not already there.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $BinDir) {
  $newPath = if ($userPath) { "$userPath;$BinDir" } else { $BinDir }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
  Write-Host "hera: added $BinDir to your user PATH (restart the terminal to pick it up)"
}
