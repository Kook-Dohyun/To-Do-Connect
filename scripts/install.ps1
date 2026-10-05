# Windows installation from GitHub Releases or a local release. No Go, Node or Docker.
[CmdletBinding()]
param(
    [string]$ReleaseDirectory,
    [Parameter(Mandatory = $true)][string]$Version,
    [string]$InstallDirectory,
    [switch]$RegisterCodex
)

$ErrorActionPreference = 'Stop'
# A launcher may inherit PowerShell 7's PSModulePath. Load this shell's own
# inbox utility module so Windows PowerShell 5.1 resolves Get-FileHash correctly.
Import-Module (Join-Path $PSHOME 'Modules/Microsoft.PowerShell.Utility/Microsoft.PowerShell.Utility.psd1')
if ($env:OS -ne 'Windows_NT') { throw 'This installer is for Windows.' }
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$') { throw 'Version must be a semantic version without path components.' }
$nativeArchitecture = $env:PROCESSOR_ARCHITEW6432
if (-not $nativeArchitecture) { $nativeArchitecture = $env:PROCESSOR_ARCHITECTURE }
$architecture = switch ($nativeArchitecture) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "Unsupported Windows architecture: $nativeArchitecture" }
}
$archiveName = "todo-connect-$Version-windows-$architecture.zip"
$downloadRoot = $null
try {
if (-not $ReleaseDirectory) {
    $downloadRoot = Join-Path ([IO.Path]::GetTempPath()) ('todo-connect-download-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $downloadRoot | Out-Null
    $releaseURL = "https://github.com/Kook-Dohyun/To-Do-Connect/releases/download/v$Version"
    $previousTLS = [Net.ServicePointManager]::SecurityProtocol
    try {
        [Net.ServicePointManager]::SecurityProtocol = $previousTLS -bor [Net.SecurityProtocolType]::Tls12
        foreach ($asset in @('SHA256SUMS', $archiveName)) {
            Invoke-WebRequest -UseBasicParsing -Uri "$releaseURL/$asset" -OutFile (Join-Path $downloadRoot $asset) -TimeoutSec 60
        }
    } finally { [Net.ServicePointManager]::SecurityProtocol = $previousTLS }
    $ReleaseDirectory = $downloadRoot
}
$releaseRoot = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$archivePath = Join-Path $releaseRoot $archiveName
$checksumLines = @(Get-Content -LiteralPath (Join-Path $releaseRoot 'SHA256SUMS') | Where-Object { $_ -match ('^[a-fA-F0-9]{64}  ' + [regex]::Escape($archiveName) + '$') })
if ($checksumLines.Count -ne 1) { throw 'Release checksum entry is missing or duplicated.' }
$expectedHash = $checksumLines[0].Substring(0, 64)
if ((Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash -ne $expectedHash) { throw 'Release checksum mismatch; nothing was installed.' }
if (-not $InstallDirectory) {
    $InstallDirectory = Join-Path $env:LOCALAPPDATA "Programs/ToDoConnect/$Version"
}
if (-not [IO.Path]::IsPathRooted($InstallDirectory)) { throw 'InstallDirectory must be absolute.' }
$destination = [IO.Path]::GetFullPath($InstallDirectory)
if (Test-Path -LiteralPath $destination) { throw "Install destination already exists; it was not modified: $destination. Use a new directory for updates." }
if ($RegisterCodex) { $codex = (Get-Command codex -ErrorAction Stop).Source }

# ExtractToDirectory rejects traversal outside the destination and does not
# overwrite existing files. A failed extraction is retained for inspection.
Add-Type -AssemblyName System.IO.Compression.FileSystem
[IO.Compression.ZipFile]::ExtractToDirectory($archivePath, $destination)
$build = Get-Content -Raw -LiteralPath (Join-Path $destination 'BUILD.json') | ConvertFrom-Json
$binaryRelative = 'plugins/todo-connect/bin/todo-connect.exe'
if ($build.version -ne $Version -or $build.os -ne 'windows' -or $build.arch -ne $architecture -or $build.binary -ne $binaryRelative) {
    throw "Package metadata does not match this host. Extracted files retained at $destination; no program was executed."
}
$program = Join-Path $destination $binaryRelative
if ((Get-FileHash -LiteralPath $program -Algorithm SHA256).Hash -ne $build.sha256) { throw 'Executable checksum mismatch; no program was executed.' }
$reportedVersion = & $program version
if ($LASTEXITCODE -ne 0 -or $reportedVersion -ne $Version) { throw 'Installed executable did not report the expected version.' }

if ($RegisterCodex) {
    & $codex plugin marketplace add $destination --json
    if ($LASTEXITCODE -ne 0) { throw "Program installed, but marketplace registration failed. Files retained at $destination." }
    & $codex plugin add todo-connect@todo-connect-local --json
    if ($LASTEXITCODE -ne 0) { throw "Marketplace registered, but plugin installation failed. Files retained at $destination." }
}
[pscustomobject]@{
    Version = $Version
    Architecture = $architecture
    Program = $program
    MarketplaceDirectory = $destination
    CodexRegistrationRequested = [bool]$RegisterCodex
}
} finally {
    if ($downloadRoot) {
        # Only the two files created by this invocation; never recurse into a user path.
        Remove-Item -LiteralPath (Join-Path $downloadRoot 'SHA256SUMS'), (Join-Path $downloadRoot $archiveName) -Force -ErrorAction SilentlyContinue
        [IO.Directory]::Delete($downloadRoot)
    }
}
