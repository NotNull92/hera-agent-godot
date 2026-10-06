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
param([switch]$NoPathUpdate)
$ErrorActionPreference = 'Stop'

$Repo = 'NotNull92/hera-agent-godot'
$Version = if ($env:HERA_VERSION) { $env:HERA_VERSION } else { 'latest' }
$BinDir = if ($env:HERA_BIN_DIR) { $env:HERA_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'hera\bin' }
$Architecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } elseif ($env:PROCESSOR_ARCHITECTURE) { $env:PROCESSOR_ARCHITECTURE } else { [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() }

switch ($Architecture) {
  'AMD64' { $arch = 'amd64' }
  'X64' { $arch = 'amd64' }
  'ARM64' { $arch = 'arm64' }
  default { throw "hera-godot: unsupported architecture: $Architecture" }
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
  Write-Host "hera-godot: downloading $url"
  Invoke-WebRequest -Uri $url -OutFile $zip
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
  $extracted = Join-Path $tmp 'extracted'
  Expand-Archive -LiteralPath $zip -DestinationPath $extracted
  Copy-Item -LiteralPath (Join-Path $extracted 'hera.exe') -Destination (Join-Path $BinDir 'hera-godot.exe') -Force
} finally {
  $resolvedTemp = [IO.Path]::GetFullPath($tmp)
  if ([IO.Path]::GetDirectoryName($resolvedTemp) -ne [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') -or [IO.Path]::GetFileName($resolvedTemp) -notlike 'hera-*') { throw 'Unexpected temporary directory' }
  Remove-Item -LiteralPath $resolvedTemp -Recurse -Force -ErrorAction SilentlyContinue
}

$exe = Join-Path $BinDir 'hera-godot.exe'
# Transitional alias for scripts that still call the long name.
Set-Content -Path (Join-Path $BinDir 'hera-agent-godot.cmd') -Value "@`"%~dp0hera-godot.exe`" %*"
Write-Host "hera-godot: installed to $exe (alias: hera-agent-godot); existing hera commands were preserved"
try { Write-Host "hera-godot: version $(& $exe version)" } catch {}

# Add BinDir to the user PATH if it is not already there.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (!$NoPathUpdate -and ($userPath -split ';') -notcontains $BinDir) {
  $newPath = if ($userPath) { "$userPath;$BinDir" } else { $BinDir }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
  Write-Host "hera-godot: added $BinDir to your user PATH (restart the terminal to pick it up)"
}
