[CmdletBinding()]
param(
    [string]$BundleRoot = $PSScriptRoot
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2

$metadataPath = Join-Path $BundleRoot "version.json"
$binarySource = Join-Path $BundleRoot "tabcli.exe"
$extensionArchive = Join-Path $BundleRoot "tabcli-extension.zip"
foreach ($required in @($metadataPath, $binarySource, $extensionArchive)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) {
        throw "Required bundle file is missing: $required"
    }
}

$metadata = Get-Content -LiteralPath $metadataPath -Raw | ConvertFrom-Json
if (-not $metadata.version) {
    throw "version.json does not contain a version"
}
if (-not $env:LOCALAPPDATA) {
    throw "LOCALAPPDATA is not set"
}

$programDirectory = Join-Path $env:LOCALAPPDATA "Programs\tabcli"
$binaryDestination = Join-Path $programDirectory "tabcli.exe"
$productDirectory = Join-Path $env:LOCALAPPDATA "tabcli"
$releaseDirectory = Join-Path $productDirectory ("releases\" + $metadata.version)
$extensionDirectory = Join-Path $releaseDirectory "tabcli-extension-unpacked"
$stagingRoot = Join-Path $productDirectory (".install-" + [Guid]::NewGuid().ToString("N"))

New-Item -ItemType Directory -Force -Path $programDirectory, $releaseDirectory, $stagingRoot | Out-Null
try {
    $stagedBinary = Join-Path $stagingRoot "tabcli.exe"
    $stagedExtension = Join-Path $stagingRoot "tabcli-extension-unpacked"
    Copy-Item -LiteralPath $binarySource -Destination $stagedBinary
    Expand-Archive -LiteralPath $extensionArchive -DestinationPath $stagedExtension
    $binaryVersion = (& $stagedBinary --json version | ConvertFrom-Json).version
    $extensionVersion = (Get-Content -LiteralPath (Join-Path $stagedExtension "manifest.json") -Raw | ConvertFrom-Json).version
    if ($binaryVersion -ne $metadata.version -or $extensionVersion -ne $metadata.version) {
        throw "Bundle version mismatch: metadata=$($metadata.version), binary=$binaryVersion, extension=$extensionVersion"
    }

    $pendingBinary = "$binaryDestination.new"
    Copy-Item -LiteralPath $stagedBinary -Destination $pendingBinary -Force
    try {
        Move-Item -LiteralPath $pendingBinary -Destination $binaryDestination -Force
    } catch {
        Remove-Item -LiteralPath $pendingBinary -Force -ErrorAction SilentlyContinue
        throw "Unable to replace tabcli.exe. Completely exit Google Chrome and rerun the installer. No process was terminated. $($_.Exception.Message)"
    }

    if (Test-Path -LiteralPath $extensionDirectory) {
        Remove-Item -LiteralPath $extensionDirectory -Recurse -Force
    }
    Move-Item -LiteralPath $stagedExtension -Destination $extensionDirectory
    Copy-Item -LiteralPath $metadataPath -Destination (Join-Path $releaseDirectory "version.json") -Force

    & $binaryDestination install
    if ($LASTEXITCODE -ne 0) {
        throw "tabcli install failed with exit code $LASTEXITCODE"
    }
} finally {
    if (Test-Path -LiteralPath $stagingRoot) {
        Remove-Item -LiteralPath $stagingRoot -Recurse -Force
    }
}

Write-Host "Installed tabcli: $binaryDestination"
Write-Host "Chrome extension directory: $extensionDirectory"
Write-Host "Open chrome://extensions, enable Developer mode, and choose Load unpacked."
Write-Host "This unsigned build may show a Windows SmartScreen warning."
