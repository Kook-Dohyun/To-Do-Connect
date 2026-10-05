[CmdletBinding()]
param([string]$Version = 'dev')

$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[a-zA-Z0-9][a-zA-Z0-9._-]*$') { throw 'Version must be a single safe directory component.' }
$projectRoot = Split-Path -Parent $PSScriptRoot
$previousGoOS = $env:GOOS
$previousGoArch = $env:GOARCH
$previousCgo = $env:CGO_ENABLED
Push-Location -LiteralPath (Join-Path $projectRoot 'server')
try {
    $env:CGO_ENABLED = '0'
    foreach ($targetOS in @('windows', 'linux', 'darwin')) {
        foreach ($targetArch in @('amd64', 'arm64')) {
            $env:GOOS = $targetOS
            $env:GOARCH = $targetArch
            $targetDir = Join-Path $projectRoot "dist/$Version/$targetOS-$targetArch"
            New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
            $name = if ($targetOS -eq 'windows') { 'todo-connect.exe' } else { 'todo-connect' }
            $output = Join-Path $targetDir $name
            go build -trimpath -ldflags="-s -w -X github.com/Kook-Dohyun/To-Do-Connect/internal/app.buildVersion=$Version" -o $output ./cmd/todo-connect
            if ($LASTEXITCODE -ne 0) { throw "Build failed: $targetOS/$targetArch" }
            [pscustomobject]@{ Target = "$targetOS/$targetArch"; Path = $output; SHA256 = (Get-FileHash -LiteralPath $output -Algorithm SHA256).Hash }
        }
    }
}
finally {
    $env:GOOS = $previousGoOS
    $env:GOARCH = $previousGoArch
    $env:CGO_ENABLED = $previousCgo
    Pop-Location
}
