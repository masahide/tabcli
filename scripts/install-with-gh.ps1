[CmdletBinding()]
param(
    [string]$Repository = "masahide/tabcli",
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    throw "GitHub CLI (gh) is required"
}

$temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("tabcli-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $temporary | Out-Null
try {
    $tagArguments = @()
    if ($Version) {
        $tagArguments = @("--pattern", "tabcli-$Version-windows-amd64.zip")
    } else {
        $tagArguments = @("--pattern", "tabcli-*-windows-amd64.zip")
    }
    & gh release download @tagArguments --pattern "SHA256SUMS" --repo $Repository --dir $temporary --clobber
    if ($LASTEXITCODE -ne 0) {
        throw "GitHub Release download failed"
    }
    $bundle = Get-ChildItem -LiteralPath $temporary -Filter "tabcli-*-windows-amd64.zip" | Select-Object -First 1
    if (-not $bundle) {
        throw "Windows amd64 bundle was not downloaded"
    }
    $checksumLine = Get-Content -LiteralPath (Join-Path $temporary "SHA256SUMS") |
        Where-Object { $_ -match ([Regex]::Escape($bundle.Name) + '$') } |
        Select-Object -First 1
    if (-not $checksumLine) {
        throw "SHA256SUMS does not contain $($bundle.Name)"
    }
    $expected = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -LiteralPath $bundle.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Checksum mismatch for $($bundle.Name)"
    }
    $expanded = Join-Path $temporary "bundle"
    Expand-Archive -LiteralPath $bundle.FullName -DestinationPath $expanded
    & (Join-Path $expanded "install.ps1") -BundleRoot $expanded
    if ($LASTEXITCODE -ne 0) {
        throw "Installer failed with exit code $LASTEXITCODE"
    }
} finally {
    if (Test-Path -LiteralPath $temporary) {
        Remove-Item -LiteralPath $temporary -Recurse -Force
    }
}
