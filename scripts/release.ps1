# release.ps1
# Builds release archives for all target platforms and places them in dist/.
#
# Usage (via Makefile):
#   make release VERSION=x.y.z
#
# Usage (direct):
#   pwsh -NoProfile -File scripts/release.ps1 -Version 1.2.3

param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root   = Split-Path $PSScriptRoot -Parent
$binary = "mack"
$dist   = Join-Path $root "dist"

# Resolve build metadata
$commit    = (git -C $root rev-parse --short HEAD 2>$null).Trim()
if (-not $commit) { $commit = "unknown" }
$buildDate = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
$ldflags   = "-s -w -X main.Version=$Version -X main.Commit=$commit -X main.BuildDate=$buildDate"

# ── Update Windows resource metadata (.syso) with this release version ────────
$winresJson = Join-Path $root "winres\winres.json"
$winresOrig = [System.IO.File]::ReadAllText($winresJson)

# Patch all version strings in the JSON
$winresNew = $winresOrig `
    -replace '"file_version":\s*"[^"]*"',    ('"file_version": "{0}.0"' -f $Version) `
    -replace '"product_version":\s*"[^"]*"', ('"product_version": "{0}.0"' -f $Version) `
    -replace '"FileVersion":\s*"[^"]*"',     ('"FileVersion": "{0}.0"' -f $Version) `
    -replace '"ProductVersion":\s*"[^"]*"',  ('"ProductVersion": "{0}"' -f $Version)

[System.IO.File]::WriteAllText($winresJson, $winresNew)
Write-Host "Updated winres/winres.json to v$Version"

# Regenerate .syso files with new version
Push-Location $root
& go-winres make --product-version "$Version.0" --file-version "$Version.0"
if ($LASTEXITCODE -ne 0) {
    [System.IO.File]::WriteAllText($winresJson, $winresOrig)
    Write-Error "go-winres failed"
    exit 1
}
Pop-Location

$platforms = @(
    @{ GOOS = "linux";   GOARCH = "amd64"; ext = "" },
    @{ GOOS = "linux";   GOARCH = "arm64"; ext = "" },
    @{ GOOS = "windows"; GOARCH = "amd64"; ext = ".exe" },
    @{ GOOS = "darwin";  GOARCH = "amd64"; ext = "" },
    @{ GOOS = "darwin";  GOARCH = "arm64"; ext = "" }
)

# Clean and recreate dist/
if (Test-Path $dist) { Remove-Item $dist -Recurse -Force }
New-Item $dist -ItemType Directory | Out-Null

Write-Host "Building mack v$Version  (commit: $commit)"
Write-Host ""

foreach ($p in $platforms) {
    $os   = $p.GOOS
    $arch = $p.GOARCH
    $exe  = "$binary$($p.ext)"
    $name = "${binary}_${Version}_${os}_${arch}"
    $out  = Join-Path $dist $exe

    Write-Host -NoNewline "  $os/$arch ... "

    $env:GOOS   = $os
    $env:GOARCH = $arch

    $result = & go build -ldflags $ldflags -trimpath -o $out ./cmd/mack-bot/ 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAILED"
        Write-Error "Build failed for ${os}/${arch}:`n$result"
        exit 1
    }

    # Package into archive
    if ($os -eq "windows") {
        $archive = Join-Path $dist "$name.zip"
        Compress-Archive -Path $out -DestinationPath $archive -Force
    } else {
        $archive = Join-Path $dist "$name.tar.gz"
        & tar -czf $archive -C $dist $exe
        if ($LASTEXITCODE -ne 0) {
            Write-Error "tar failed for ${os}/${arch}"
            exit 1
        }
    }

    Remove-Item $out -Force
    Write-Host (Split-Path $archive -Leaf)
}

Remove-Item Env:GOOS   -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Release archives written to dist/"
Get-ChildItem $dist | Select-Object Name, @{N="Size";E={ "{0:N0} KB" -f ($_.Length / 1KB) }} | Format-Table -AutoSize
