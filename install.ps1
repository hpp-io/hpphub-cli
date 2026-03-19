# HPP Hub CLI installer for Windows PowerShell
# Usage: irm https://raw.githubusercontent.com/hpp-io/hpphub-cli/main/install.ps1 | iex

$ErrorActionPreference = "Stop"
$repo = "hpp-io/hpphub-cli"
$installDir = "$env:LOCALAPPDATA\hpphub"
$binaryName = "hpphub.exe"

Write-Host "Fetching latest release..." -ForegroundColor Cyan

try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $tag = $release.tag_name
} catch {
    Write-Host "Error: Could not determine latest release" -ForegroundColor Red
    exit 1
}

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$asset = "hpphub-windows-$arch.exe"
$downloadUrl = "https://github.com/$repo/releases/download/$tag/$asset"

Write-Host "Downloading hpphub $tag for windows/$arch..." -ForegroundColor Cyan

try {
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $outPath = Join-Path $installDir $binaryName
    Invoke-WebRequest -Uri $downloadUrl -OutFile $outPath
} catch {
    Write-Host "Error: Download failed - $_" -ForegroundColor Red
    exit 1
}

# Add to PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "  Added $installDir to PATH (restart terminal to apply)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "  hpphub $tag installed to $outPath" -ForegroundColor Green
Write-Host ""
Write-Host "  Get started (restart terminal first):" -ForegroundColor Cyan
Write-Host "    hpphub launch openclaw"
Write-Host ""
