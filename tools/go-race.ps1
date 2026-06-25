param(
    [string]$Pattern = "./...",
    [switch]$VerboseTests
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$gccDir = "C:\Users\PC\msys64\ucrt64\bin"
if (Test-Path -LiteralPath $gccDir) {
    $env:PATH = "$gccDir;$env:PATH"
}

$env:CGO_ENABLED = "1"
$env:CC = "gcc"
$env:CXX = "g++"

$tmp = Join-Path $repoRoot ".gotmp-race"
if (Test-Path -LiteralPath $tmp) {
    Remove-Item -LiteralPath $tmp -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

try {
    $packages = go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' $Pattern |
        Where-Object { $_ -ne "" }
    foreach ($pkg in $packages) {
        $name = ($pkg -replace '[^a-zA-Z0-9]+', '_').Trim('_')
        $out = Join-Path $tmp "$name-race.test.exe"
        Write-Host "race: $pkg"
        go test -race -c $pkg -o $out
        if ($VerboseTests) {
            & $out "-test.v"
        } else {
            $log = Join-Path $tmp "$name-race.log"
            & $out > $log 2>&1
            if ($LASTEXITCODE -ne 0) {
                Get-Content -LiteralPath $log
                exit $LASTEXITCODE
            }
            Write-Host "pass: $pkg"
        }
    }
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
