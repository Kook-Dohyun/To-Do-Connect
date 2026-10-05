[CmdletBinding()]
param([string]$Version = '0.2.0-dev')

$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$') { throw 'Version must be a semantic version without path components.' }
$projectRoot = Split-Path -Parent $PSScriptRoot
$releaseDir = Join-Path $projectRoot "dist/releases/$Version"
if (Test-Path -LiteralPath $releaseDir) { throw "Release output already exists: $releaseDir. Use a new version; existing artifacts are never overwritten." }

# Build from this source tree; never package a stale binary from a previous version.
& (Join-Path $PSScriptRoot 'build-targets.ps1') -Version $Version
New-Item -ItemType Directory -Path $releaseDir | Out-Null
$utf8 = [System.Text.UTF8Encoding]::new($false)
$checksums = [System.Collections.Generic.List[string]]::new()

foreach ($targetOS in @('windows', 'linux', 'darwin')) {
    foreach ($targetArch in @('amd64', 'arm64')) {
        $target = "$targetOS-$targetArch"
        $bundle = Join-Path $releaseDir "todo-connect-$Version-$target"
        $pluginRoot = Join-Path $bundle 'plugins/todo-connect'
        New-Item -ItemType Directory -Path (Join-Path $pluginRoot 'bin') -Force | Out-Null
        $binaryName = if ($targetOS -eq 'windows') { 'todo-connect.exe' } else { 'todo-connect' }
        Copy-Item -LiteralPath (Join-Path $projectRoot "dist/$Version/$target/$binaryName") -Destination (Join-Path $pluginRoot "bin/$binaryName")
        Copy-Item -LiteralPath (Join-Path $projectRoot 'LICENSE') -Destination $pluginRoot
        Push-Location -LiteralPath (Join-Path $projectRoot 'server')
        try {
            go run ../tools/license-notices/main.go (Join-Path $pluginRoot "bin/$binaryName") (Join-Path $pluginRoot 'THIRD_PARTY_NOTICES.txt')
            if ($LASTEXITCODE -ne 0) { throw "License notice collection failed for $target" }
        } finally { Pop-Location }
        Copy-Item -LiteralPath (Join-Path $projectRoot 'packaging/plugin/skills') -Destination $pluginRoot -Recurse
        Copy-Item -LiteralPath (Join-Path $projectRoot 'packaging/plugin/assets') -Destination $pluginRoot -Recurse
        $manifest = Get-Content -Raw -LiteralPath (Join-Path $projectRoot 'packaging/plugin/plugin.json') | ConvertFrom-Json
        $manifest.version = $Version
        [IO.File]::WriteAllText((Join-Path $pluginRoot 'plugin.json'), ($manifest | ConvertTo-Json -Depth 12), $utf8)
        $mcp = Get-Content -Raw -LiteralPath (Join-Path $projectRoot 'packaging/plugin/mcp.json') | ConvertFrom-Json
        $mcp.mcpServers.'todo-connect'.command = './bin/' + $binaryName
        [IO.File]::WriteAllText((Join-Path $pluginRoot 'mcp.json'), ($mcp | ConvertTo-Json -Depth 12), $utf8)
        $marketDir = Join-Path $bundle '.agents/plugins'
        New-Item -ItemType Directory -Path $marketDir -Force | Out-Null
        $market = @{
            name = 'todo-connect-local'
            plugins = @(@{
                name = 'todo-connect'
                source = @{ source = 'local'; path = './plugins/todo-connect' }
                policy = @{ installation = 'AVAILABLE'; authentication = 'ON_INSTALL' }
                category = 'Productivity'
            })
        }
        [IO.File]::WriteAllText((Join-Path $marketDir 'marketplace.json'), ($market | ConvertTo-Json -Depth 12), $utf8)
        Copy-Item -LiteralPath (Join-Path $projectRoot 'packaging/GETTING_STARTED.md') -Destination $bundle
        Copy-Item -LiteralPath (Join-Path $projectRoot 'docs/runtime.md') -Destination (Join-Path $bundle 'RUNTIME.md')
        $buildInfo = @{ version = $Version; os = $targetOS; arch = $targetArch; binary = "plugins/todo-connect/bin/$binaryName"; sha256 = (Get-FileHash -LiteralPath (Join-Path $pluginRoot "bin/$binaryName") -Algorithm SHA256).Hash.ToLowerInvariant() }
        [IO.File]::WriteAllText((Join-Path $bundle 'BUILD.json'), ($buildInfo | ConvertTo-Json), $utf8)

        $zipPath = "$bundle.zip"
        Push-Location -LiteralPath (Join-Path $projectRoot 'tools')
        try {
            go run ./package-archive $bundle $zipPath
            if ($LASTEXITCODE -ne 0) { throw "ZIP assembly failed for $target" }
        } finally { Pop-Location }
        $checksums.Add((Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + [IO.Path]::GetFileName($zipPath))
        Write-Output $zipPath
    }
}
foreach ($installer in @('install.ps1', 'install.sh')) {
    $installerPath = Join-Path $releaseDir $installer
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot $installer) -Destination $installerPath
    $checksums.Add((Get-FileHash -LiteralPath $installerPath -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $installer)
}
[IO.File]::WriteAllLines((Join-Path $releaseDir 'SHA256SUMS'), $checksums, $utf8)
& (Join-Path $PSScriptRoot 'package-plugin-zips.ps1') -ReleaseDirectory $releaseDir
Write-Output 'Local release artifacts assembled. This does not publish or install the plugin.'
