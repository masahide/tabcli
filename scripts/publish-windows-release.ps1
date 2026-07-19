[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^\d+\.\d+\.\d+([-.][0-9A-Za-z.-]+)?$')]
    [string]$Version,

    [string]$Repository = "masahide/tabcli",
    [string]$Remote = "origin",
    [string]$OutDirectory = "dist",
    [switch]$Publish
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE"
    }
}

function Get-NativeOutput {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )
    $output = & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE"
    }
    return ($output | Out-String).Trim()
}

function Test-NativeSuccess {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )
    $previousErrorActionPreference = $ErrorActionPreference
    try {
        # Windows PowerShell 5.1 converts expected native stderr (for example,
        # "release not found") into NativeCommandError when Stop is active.
        $ErrorActionPreference = "Continue"
        & $Command @Arguments *> $null
        return $LASTEXITCODE -eq 0
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
}

function Assert-Checksums {
    param([Parameter(Mandatory = $true)][string]$Root)

    $checksumPath = Join-Path $Root "SHA256SUMS"
    if (-not (Test-Path -LiteralPath $checksumPath -PathType Leaf)) {
        throw "SHA256SUMS was not generated"
    }
    foreach ($line in Get-Content -LiteralPath $checksumPath) {
        if ($line -notmatch '^([0-9a-fA-F]{64})\s{2}(.+)$') {
            throw "Invalid SHA256SUMS line: $line"
        }
        $expected = $Matches[1].ToLowerInvariant()
        $relativePath = $Matches[2].Replace('/', [IO.Path]::DirectorySeparatorChar)
        $artifactPath = Join-Path $Root $relativePath
        if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) {
            throw "Checksummed artifact is missing: $relativePath"
        }
        $actual = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "Checksum mismatch: $relativePath"
        }
    }
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Push-Location $repositoryRoot
try {
    foreach ($command in @("git", "go", "npm")) {
        if (-not (Get-Command $command -ErrorAction SilentlyContinue)) {
            throw "Required command is unavailable: $command"
        }
    }
    if ($Publish -and -not (Get-Command "gh" -ErrorAction SilentlyContinue)) {
        throw "GitHub CLI (gh) is required when -Publish is specified"
    }

    $status = Get-NativeOutput "git" @("status", "--porcelain")
    if ($status) {
        throw "Release requires a clean worktree. Commit or stash all changes first."
    }
    $commit = Get-NativeOutput "git" @("rev-parse", "HEAD")
    $tag = "v$Version"

    foreach ($manifestPath in @("extension\manifest.json", "extension\package.json")) {
        $document = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
        if ($document.version -ne $Version) {
            throw "$manifestPath version $($document.version) does not match release version $Version"
        }
    }

    Write-Host "Building tabcli $Version for Windows amd64 from $commit"
    Invoke-Native "go" @(
        "run", ".\cmd\release",
        "--target", "windows-amd64",
        "--out", $OutDirectory,
        "--version", $Version,
        "--commit", $commit
    )

    $outRoot = (Resolve-Path $OutDirectory).Path
    $bundleName = "tabcli-$Version-windows-amd64.zip"
    $requiredArtifacts = @(
        $bundleName,
        "tabcli.exe",
        "tabcli-extension.zip",
        "install.ps1",
        "install-with-gh.ps1",
        "INSTALL.txt",
        "version.json",
        "SHA256SUMS"
    )
    foreach ($artifact in $requiredArtifacts) {
        if (-not (Test-Path -LiteralPath (Join-Path $outRoot $artifact) -PathType Leaf)) {
            throw "Required release artifact is missing: $artifact"
        }
    }
    Assert-Checksums $outRoot

    $metadata = Get-Content -LiteralPath (Join-Path $outRoot "version.json") -Raw | ConvertFrom-Json
    if ($metadata.version -ne $Version -or $metadata.commit -ne $commit) {
        throw "version.json does not match version $Version and commit $commit"
    }
    if ($metadata.targets -notcontains "windows/amd64") {
        throw "version.json does not contain the windows/amd64 target"
    }

    Write-Host "Build and artifact verification succeeded: $outRoot"
    if (-not $Publish) {
        Write-Host "Dry run only. Re-run with -Publish to create and push $tag and publish the GitHub Release."
        return
    }

    Invoke-Native "gh" @("auth", "status")
    if (Test-NativeSuccess "gh" @("release", "view", $tag, "--repo", $Repository)) {
        throw "GitHub Release $tag already exists in $Repository"
    }

    if (Test-NativeSuccess "git" @("show-ref", "--verify", "--quiet", "refs/tags/$tag")) {
        $localTagCommit = Get-NativeOutput "git" @("rev-list", "-n", "1", $tag)
        if ($localTagCommit -ne $commit) {
            throw "Local tag $tag points to $localTagCommit instead of $commit"
        }
        $localTagType = Get-NativeOutput "git" @("cat-file", "-t", "refs/tags/$tag")
        if ($localTagType -ne "tag") {
            throw "Local tag $tag is not an annotated tag"
        }
    } else {
        Invoke-Native "git" @("tag", "-a", $tag, $commit, "-m", "tabcli $Version")
    }

    $remoteTagLine = Get-NativeOutput "git" @("ls-remote", "--tags", $Remote, "refs/tags/$tag^{}")
    if (-not $remoteTagLine) {
        $remoteUndereferencedTag = Get-NativeOutput "git" @("ls-remote", "--tags", $Remote, "refs/tags/$tag")
        if ($remoteUndereferencedTag) {
            throw "Remote tag $tag exists but is not an annotated tag"
        }
    }
    if ($remoteTagLine) {
        $remoteTagCommit = ($remoteTagLine -split '\s+')[0]
        if ($remoteTagCommit -ne $commit) {
            throw "Remote tag $tag points to $remoteTagCommit instead of $commit"
        }
    } else {
        Invoke-Native "git" @("push", $Remote, "refs/tags/$tag")
    }

    $assetNames = @(
        $bundleName,
        "tabcli.exe",
        "tabcli-extension.zip",
        "install.ps1",
        "install-with-gh.ps1",
        "INSTALL.txt",
        "version.json",
        "SHA256SUMS"
    )
    $releaseArguments = @(
        "release", "create", $tag,
        "--repo", $Repository,
        "--verify-tag",
        "--target", $commit,
        "--title", "tabcli $Version",
        "--generate-notes"
    )
    foreach ($assetName in $assetNames) {
        $releaseArguments += (Join-Path $outRoot $assetName)
    }
    Invoke-Native "gh" $releaseArguments
    Write-Host "Published GitHub Release: https://github.com/$Repository/releases/tag/$tag"
} finally {
    Pop-Location
}
