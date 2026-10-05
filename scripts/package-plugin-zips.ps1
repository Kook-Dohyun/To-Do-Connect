# Repackage verified release bundles as single-root plugin ZIPs.
# Does not install, authenticate, upload, or publish anything.
[CmdletBinding()]
param([Parameter(Mandatory = $true)][string]$ReleaseDirectory)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression.FileSystem
$release = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$version = Split-Path -Leaf $release
$output = Join-Path $release 'plugin-zips'
if (Test-Path -LiteralPath $output) { throw "Plugin ZIP output already exists: $output" }
$sums = Get-Content -LiteralPath (Join-Path $release 'SHA256SUMS')
$archives = @(Get-ChildItem -LiteralPath $release -Filter "todo-connect-$version-*.zip" -File)
if ($archives.Count -ne 6) { throw 'Expected six platform release archives.' }
foreach ($archive in $archives) {
    $checksum = @( $sums | Where-Object { $_ -match ('^[a-fA-F0-9]{64}  ' + [regex]::Escape($archive.Name) + '$') } )
    if ($checksum.Count -ne 1 -or (Get-FileHash -LiteralPath $archive.FullName -Algorithm SHA256).Hash -ne $checksum[0].Substring(0, 64)) {
        throw "Release checksum mismatch: $($archive.Name)"
    }
}
New-Item -ItemType Directory -Path $output | Out-Null
$pluginSums = @()
$prefix = 'plugins/todo-connect/'
foreach ($archive in $archives) {
    $name = $archive.Name.Replace("todo-connect-$version-", "todo-connect-plugin-$version-")
    $destination = Join-Path $output $name
    $sourceZip = [IO.Compression.ZipFile]::OpenRead($archive.FullName)
    try {
        $entries = @($sourceZip.Entries | Where-Object { $_.FullName.StartsWith($prefix, [StringComparison]::Ordinal) -and $_.Name })
        if (-not ($entries.FullName -contains ($prefix + 'plugin.json')) -or -not ($entries.FullName -contains ($prefix + 'mcp.json'))) {
            throw 'Release is missing plugin manifests.'
        }
        $targetZip = [IO.Compression.ZipFile]::Open($destination, [IO.Compression.ZipArchiveMode]::Create)
        try {
            foreach ($entry in $entries) {
                $relative = $entry.FullName.Substring($prefix.Length)
                $copy = $targetZip.CreateEntry($relative, [IO.Compression.CompressionLevel]::Optimal)
                $copy.ExternalAttributes = $entry.ExternalAttributes
                $copy.LastWriteTime = $entry.LastWriteTime
                $inputStream = $entry.Open()
                $outputStream = $copy.Open()
                try { $inputStream.CopyTo($outputStream) }
                finally { $outputStream.Dispose(); $inputStream.Dispose() }
            }
        } finally { $targetZip.Dispose() }
    } finally { $sourceZip.Dispose() }
    $pluginSums += (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $name
    Write-Output $destination
}
[IO.File]::WriteAllLines((Join-Path $output 'SHA256SUMS'), [string[]]$pluginSums, [Text.UTF8Encoding]::new($false))
