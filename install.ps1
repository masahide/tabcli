$previousErrorActionPreference = $ErrorActionPreference
$previousProgressPreference = $ProgressPreference
$previousSecurityProtocol = [Net.ServicePointManager]::SecurityProtocol

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repository = "masahide/tabcli"
$apiRoot = "https://api.github.com/repos/$repository"
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("tabcli-install-" + [Guid]::NewGuid().ToString("N"))

function Get-ReleaseAsset {
    param(
        [Parameter(Mandatory = $true)]$Release,
        [Parameter(Mandatory = $true)][string]$Name
    )
    $asset = @($Release.assets) | Where-Object { $_.name -eq $Name } | Select-Object -First 1
    if (-not $asset) {
        throw "GitHub Release does not contain required asset: $Name"
    }
    $uri = [Uri]$asset.browser_download_url
    if ($uri.Scheme -ne "https" -or $uri.Host -ne "github.com") {
        throw "Refusing unexpected release asset URL: $uri"
    }
    return $asset
}

try {
    $architecture = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) {
        $architecture = $env:PROCESSOR_ARCHITEW6432
    }
    if ($architecture -ne "AMD64") {
        throw "tabcli currently supports Windows x64 only. Detected architecture: $architecture"
    }
    if (-not $env:LOCALAPPDATA) {
        throw "LOCALAPPDATA is not set"
    }

    New-Item -ItemType Directory -Path $temporary | Out-Null
    $headers = @{
        "Accept"     = "application/vnd.github+json"
        "User-Agent" = "tabcli-installer"
    }
    Write-Host "Resolving the latest tabcli release..."
    $release = Invoke-RestMethod -UseBasicParsing -Uri "$apiRoot/releases/latest" -Headers $headers
    if (-not $release.tag_name -or $release.draft -or $release.prerelease) {
        throw "GitHub did not return a stable tabcli release"
    }

    $version = $release.tag_name -replace '^v', ''
    $bundleName = "tabcli-$version-windows-amd64.zip"
    $bundleAsset = Get-ReleaseAsset -Release $release -Name $bundleName
    $checksumAsset = Get-ReleaseAsset -Release $release -Name "SHA256SUMS"
    $bundlePath = Join-Path $temporary $bundleName
    $checksumPath = Join-Path $temporary "SHA256SUMS"

    Write-Host "Downloading tabcli $version for Windows x64..."
    Invoke-WebRequest -UseBasicParsing -Uri $bundleAsset.browser_download_url -Headers $headers -OutFile $bundlePath
    Invoke-WebRequest -UseBasicParsing -Uri $checksumAsset.browser_download_url -Headers $headers -OutFile $checksumPath

    $checksumPattern = '^(?<hash>[0-9a-fA-F]{64})\s{2}' + [Regex]::Escape($bundleName) + '$'
    $checksumMatch = Get-Content -LiteralPath $checksumPath |
        Where-Object { $_ -match $checksumPattern } |
        Select-Object -First 1
    if (-not $checksumMatch) {
        throw "SHA256SUMS does not contain $bundleName"
    }
    $checksumMatch -match $checksumPattern | Out-Null
    $expectedHash = $Matches["hash"].ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $bundlePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "Checksum mismatch for $bundleName"
    }

    $bundleRoot = Join-Path $temporary "bundle"
    Expand-Archive -LiteralPath $bundlePath -DestinationPath $bundleRoot
    $bundleInstallerPath = Join-Path $bundleRoot "install.ps1"
    if (-not (Test-Path -LiteralPath $bundleInstallerPath -PathType Leaf)) {
        throw "Verified bundle does not contain install.ps1"
    }

    # Invoke the verified installer as a ScriptBlock so irm | iex also works
    # under execution policies that block direct execution of downloaded files.
    $bundleInstaller = [ScriptBlock]::Create(
        [IO.File]::ReadAllText($bundleInstallerPath, [Text.Encoding]::UTF8)
    )
    & $bundleInstaller -BundleRoot $bundleRoot
    Write-Host "tabcli $version installation completed."
    $installedBinary = Join-Path $env:LOCALAPPDATA "Programs\tabcli\tabcli.exe"
    $installedExtension = Join-Path $env:LOCALAPPDATA "tabcli\releases\$version\tabcli-extension-unpacked"
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "1. Open chrome://extensions in Google Chrome."
    Write-Host "2. Enable Developer mode, choose Load unpacked, and select:"
    Write-Host "   $installedExtension"
    Write-Host "3. Confirm extension ID: ddgfmgclndpdobieomcjaklboinbaoel"
    Write-Host "4. Reload the extension or restart Chrome, then verify the connection:"
    Write-Host "   & `"$installedBinary`" --json doctor"
    Write-Host "   & `"$installedBinary`" --json list"
    Write-Host "Guide: https://github.com/masahide/tabcli/blob/main/docs/getting-started-windows.md"
} finally {
    if (Test-Path -LiteralPath $temporary) {
        Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
    }
    [Net.ServicePointManager]::SecurityProtocol = $previousSecurityProtocol
    $ProgressPreference = $previousProgressPreference
    $ErrorActionPreference = $previousErrorActionPreference
}
