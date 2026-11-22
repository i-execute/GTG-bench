param(
    [switch]$y,
    [switch]$clean
)

$ErrorActionPreference = "Stop"
$Version = "0.1.0"
$Repository = "i-execute/GTG-bench"
$ReleaseBase = "https://github.com/$Repository/releases/download/v$Version"
$CacheRoot = if ($env:GTGBENCH_CACHE) { $env:GTGBENCH_CACHE } else { Join-Path $env:LOCALAPPDATA "gtg-bench" }

if ($clean) {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $CacheRoot
    Write-Output "Removed $CacheRoot"
    exit 0
}

$architecture = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") { "arm64" } elseif ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "32-bit Windows is unsupported" }
$asset = "gtg-bench_$Version`_windows_$architecture.zip"
$assetUrl = "$ReleaseBase/$asset"
$binDir = Join-Path $CacheRoot "bin\$Version\windows_$architecture"
$binary = Join-Path $binDir "gtg-bench.exe"

function Show-Preview([string]$Url) {
    $size = "unknown"
    try {
        $response = Invoke-WebRequest -Method Head -Uri $Url -UseBasicParsing
        if ($response.Headers["Content-Length"]) { $size = $response.Headers["Content-Length"] }
    } catch {}
    Write-Output "Download preview: $Url"
    Write-Output "Declared size: $size bytes"
}

function Confirm-Download {
    if ($y) { return }
    $answer = Read-Host "Continue? [y/N]"
    if ($answer -notin @("y", "Y", "yes", "YES")) {
        Write-Output "Cancelled."
        exit 0
    }
}

function Download([string]$Url, [string]$Destination) {
    Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
}

function Verify([string]$Path, [string]$Name, [string]$Sums) {
    $expected = ((Get-Content $Sums) | Where-Object { $_ -match "\s\*?$([regex]::Escape($Name))$" } | Select-Object -First 1).Split()[0]
    if (-not $expected) { return $false }
    return ((Get-FileHash -Algorithm SHA256 $Path).Hash.ToLowerInvariant() -eq $expected.ToLowerInvariant())
}

if (Test-Path $binary) {
    & $binary
    exit $LASTEXITCODE
}

Show-Preview $assetUrl
Confirm-Download
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("gtg-bench-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $work | Out-Null
try {
    $sums = Join-Path $work "SHA256SUMS"
    Download "$ReleaseBase/SHA256SUMS" $sums
    $archive = Join-Path $work $asset
    Download $assetUrl $archive
    if (-not (Verify $archive $asset $sums)) { throw "SHA256 verification failed for $asset" }
    Expand-Archive -Force -Path $archive -DestinationPath $binDir
    & $binary
    exit $LASTEXITCODE
} catch {
    Write-Output "Verified binary asset unavailable. Trying pinned source build."
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $work
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw "Go is required for the pinned source fallback and was not found." }
$sourceAsset = "gtg-bench_$Version`_src.tar.gz"
$gtgAsset = "gtg-545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13_src.tar.gz"
Show-Preview "$ReleaseBase/$sourceAsset"
Show-Preview "$ReleaseBase/$gtgAsset"
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("gtg-bench-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $work | Out-Null
try {
    $sums = Join-Path $work "SHA256SUMS"
    Download "$ReleaseBase/SHA256SUMS" $sums
    $sourceArchive = Join-Path $work $sourceAsset
    $gtgArchive = Join-Path $work $gtgAsset
    Download "$ReleaseBase/$sourceAsset" $sourceArchive
    Download "$ReleaseBase/$gtgAsset" $gtgArchive
    if (-not (Verify $sourceArchive $sourceAsset $sums)) { throw "SHA256 verification failed for $sourceAsset" }
    if (-not (Verify $gtgArchive $gtgAsset $sums)) { throw "SHA256 verification failed for $gtgAsset" }
    tar -xzf $sourceArchive -C $work
    tar -xzf $gtgArchive -C $work
    $sourceDir = Get-ChildItem -Directory -Path $work | Where-Object { $_.Name -like "GTG-bench-*" } | Select-Object -First 1
    $gtgDir = Get-ChildItem -Directory -Path $work | Where-Object { $_.Name -eq "GTG" } | Select-Object -First 1
    if (-not $sourceDir -or -not $gtgDir) { throw "Invalid verified source layout." }
    Push-Location $sourceDir.FullName
    try { $env:CGO_ENABLED = "0"; go build -trimpath -ldflags="-s -w" -o $binary ./cmd/gtg-bench } finally { Pop-Location }
    & $binary
    exit $LASTEXITCODE
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $work
}
