# Soundcheck portable installer for Windows PowerShell
# Usage: irm https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = "soundcheck/soundcheck"
$Version = if ($env:VERSION) { $env:VERSION } else { "v2.0.0" }
$VersionNum = $Version.TrimStart('v')

$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Error "Soundcheck requires a 64-bit Windows operating system."
    exit 1
}

$InstallDir = if ($env:INSTALL_DIR) {
    $env:INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Programs\soundcheck"
}

$ZipName = "soundcheck_${VersionNum}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/${Repo}/releases/download/${Version}/${ZipName}"
$ChecksumsUrl = "https://github.com/${Repo}/releases/download/${Version}/checksums.txt"

Write-Host "==> Installing Soundcheck ${Version} for windows/${Arch}..." -ForegroundColor Cyan

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    $ZipPath = Join-Path $TempDir $ZipName
    Write-Host "==> Downloading $DownloadUrl..."
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
    } catch {
        # Fallback to local build if available
        if (Test-Path "bin\soundcheck.exe") {
            Write-Host "==> Release download unavailable; installing from local bin\soundcheck.exe..." -ForegroundColor Yellow
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
            Copy-Item "bin\soundcheck.exe" (Join-Path $InstallDir "soundcheck.exe") -Force
            $Success = $true
        } else {
            throw $_
        }
    }

    if (Test-Path $ZipPath) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force
        Copy-Item (Join-Path $TempDir "soundcheck.exe") (Join-Path $InstallDir "soundcheck.exe") -Force
        $Success = $true
    }

    if ($Success) {
        $UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        if ($UserPath -notlike "*$InstallDir*") {
            [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", [EnvironmentVariableTarget]::User)
            $env:Path += ";$InstallDir"
            Write-Host "==> Added $InstallDir to user PATH." -ForegroundColor Green
        }

        Write-Host "==> Soundcheck installed successfully to $InstallDir\soundcheck.exe" -ForegroundColor Green
        Write-Host ""
        Write-Host "Run 'soundcheck doctor' to verify coding agent health." -ForegroundColor Cyan
    }
}
finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
