param([string]$HeraCli = "hera-godot")
$ErrorActionPreference = "Stop"
Push-Location $PSScriptRoot
try {
    New-Item -ItemType Directory -Force addons | Out-Null
    Copy-Item ../../addons/hera_agent_godot addons -Recurse -Force
    $versionLine = Select-String -LiteralPath addons/hera_agent_godot/plugin.cfg -Pattern '^version="([^"]+)"$'
    if (-not $versionLine) { throw "Copied addon manifest has no version." }
    $addonVersion = $versionLine.Matches[0].Groups[1].Value
    $cliVersion = (& $HeraCli --version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $cliVersion -notmatch "^(hera-godot )?v?$([regex]::Escape($addonVersion))$") {
        throw "Pass -HeraCli pointing to released hera-godot $addonVersion from npm/Scoop; got '$cliVersion'."
    }
    dotnet build
    if ($LASTEXITCODE) { throw "dotnet build failed" }
    & $HeraCli csharp-host install
    if ($LASTEXITCODE) { throw "compiler helper install failed" }
} finally {
    Pop-Location
}
