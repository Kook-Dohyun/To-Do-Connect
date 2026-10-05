[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot

Push-Location -LiteralPath (Join-Path $projectRoot 'server')
try {
    $targetOS = go env GOOS
    if ($LASTEXITCODE -ne 0) { throw 'Could not determine the Go build target.' }
    $binaryName = if ($targetOS -eq 'windows') { 'todo-connect.exe' } else { 'todo-connect' }
    $binaryPath = Join-Path $projectRoot ('bin/' + $binaryName)
    New-Item -ItemType Directory -Path (Join-Path $projectRoot 'bin') -Force | Out-Null
    go build -o $binaryPath ./cmd/todo-connect
    if ($LASTEXITCODE -ne 0) { throw 'To Do Connect build failed.' }
    Write-Output $binaryPath
}
finally {
    Pop-Location
}
