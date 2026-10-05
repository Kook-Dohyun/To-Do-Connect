[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot

Push-Location -LiteralPath (Join-Path $projectRoot 'server')
try {
    $targetOS = go env GOOS
    if ($LASTEXITCODE -ne 0) { throw 'Could not determine the Go build target.' }
    $targetArch = go env GOARCH
    if ($LASTEXITCODE -ne 0) { throw 'Could not determine the Go build architecture.' }
    $binaryName = if ($targetOS -eq 'windows') { 'todo-connect.exe' } else { 'todo-connect' }
    $targetDir = Join-Path $projectRoot "dist/dev/$targetOS-$targetArch"
    $binaryPath = Join-Path $targetDir $binaryName
    New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    go build -o $binaryPath ./cmd/todo-connect
    if ($LASTEXITCODE -ne 0) { throw 'To Do Connect build failed.' }
    Write-Output $binaryPath
}
finally {
    Pop-Location
}
