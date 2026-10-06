$ErrorActionPreference = 'Stop'
$taskRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$taskTemp = Join-Path ([IO.Path]::GetTempPath()) ('hera-installer-test-' + [guid]::NewGuid().ToString('N'))
$taskOldBin = $env:HERA_BIN_DIR
$taskOldVersion = $env:HERA_VERSION
$taskOldPath = [Environment]::GetEnvironmentVariable('Path','User')
New-Item -ItemType Directory -Path $taskTemp | Out-Null
try {
    $taskBuild = Join-Path $taskTemp 'hera.exe'
    Push-Location $taskRoot
    try {
        go build -o $taskBuild .
        if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    } finally { Pop-Location }
    $taskArchive = Join-Path $taskTemp 'fixture.zip'
    Compress-Archive -LiteralPath $taskBuild -DestinationPath $taskArchive
    function Invoke-WebRequest { param($Uri,$OutFile) Copy-Item -LiteralPath $taskArchive -Destination $OutFile }
    $env:HERA_BIN_DIR = Join-Path $taskTemp 'installed space'
    $env:HERA_VERSION = 'test-fixture'
    New-Item -ItemType Directory -Path $env:HERA_BIN_DIR | Out-Null
    $taskSentinel = Join-Path $env:HERA_BIN_DIR 'hera.exe'
    Set-Content -LiteralPath $taskSentinel -Value 'unrelated executable placeholder'
    $taskBefore = (Get-FileHash -LiteralPath $taskSentinel).Hash
    & (Join-Path $taskRoot 'install.ps1') -NoPathUpdate
    if ((Get-FileHash -LiteralPath $taskSentinel).Hash -cne $taskBefore) { throw 'Existing hera was overwritten' }
    if ([Environment]::GetEnvironmentVariable('Path','User') -cne $taskOldPath) { throw 'User PATH was changed' }
    foreach ($taskName in @('hera-godot.exe','hera-agent-godot.cmd')) {
        $taskOutput = & (Join-Path $env:HERA_BIN_DIR $taskName) --help
        if ($LASTEXITCODE -ne 0 -or ($taskOutput -join "`n") -notmatch '^hera-godot ') { throw "Launcher failed: $taskName" }
    }
    'Windows installer: PASS (local fixture archive; existing hera and user PATH preserved)'
} finally {
    $env:HERA_BIN_DIR = $taskOldBin
    $env:HERA_VERSION = $taskOldVersion
    $taskResolved = [IO.Path]::GetFullPath($taskTemp)
    if ([IO.Path]::GetDirectoryName($taskResolved) -ne [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([char[]]'\/') -or [IO.Path]::GetFileName($taskResolved) -notlike 'hera-installer-test-*') { throw 'Unexpected test cleanup path' }
    Remove-Item -LiteralPath $taskResolved -Recurse -Force
}
