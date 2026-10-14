# zgit installer for Windows (amd64).
#
# Installs BOTH the TUI (zgit.exe) and the desktop app (zgit-desktop.exe).
# Re-running it updates both to the latest release.
#
# Run in PowerShell:
#   irm https://zgit.pages.dev/install.ps1 | iex
#
# Environment variables (optional):
#   $env:ZGIT_VERSION     pinned version tag, e.g. v0.1.1 (default: latest release)
#   $env:ZGIT_NO_DESKTOP  set to '1' to install the TUI only
#
# Installs to %LocalAppData%\zgit and adds it to the user PATH.
# Every archive checksum is verified before installing anything.
$ErrorActionPreference = 'Stop'

$Repo = 'zaidejjo/zgit'
$Pinned = -not [string]::IsNullOrEmpty($env:ZGIT_VERSION)
$Version = $env:ZGIT_VERSION
if (-not $Version) {
  $Version = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
  if (-not $Version) { throw 'could not resolve latest release (set $env:ZGIT_VERSION to pin one)' }
}

switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { $Arch = 'amd64' }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE (windows/amd64 only)" }
}

$Base = "https://github.com/$Repo/releases/download/$Version"
$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) 'zgit-install'
$Dest = Join-Path $env:LocalAppData 'zgit'

function Get-InstalledVersion {
  $bin = Join-Path $Dest 'zgit.exe'
  if (-not (Test-Path $bin)) {
    $cmd = Get-Command zgit -ErrorAction SilentlyContinue
    if (-not $cmd) { return $null }
    $bin = $cmd.Source
  }
  $out = & $bin --version 2>$null
  # 'zgit v0.1.1' or 'zgit 0.1.1' (release builds strip the 'v').
  if ($out -match '^zgit v?([\d.]+)') { return $Matches[1] }
  return $null
}

function Test-DesktopPresent {
  if (Test-Path (Join-Path $Dest 'zgit-desktop.exe')) { return $true }
  return $null -ne (Get-Command zgit-desktop -ErrorAction SilentlyContinue)
}

function Install-Archive {
  # TUI archives are listed in the release checksums.txt (GoReleaser);
  # desktop bundles ship a "<asset>.sha256" sidecar instead.
  param([string]$Asset, [string]$BinName, [string]$OutName, [string]$ChecksumSource = '')
  Write-Host "install: downloading $Asset ($Version)..."
  Invoke-WebRequest "$Base/$Asset" -OutFile (Join-Path $Tmp $Asset)

  Write-Host 'install: verifying checksum...'
  if ($ChecksumSource -eq '') {
    Invoke-WebRequest "$Base/checksums.txt" -OutFile (Join-Path $Tmp 'checksums.txt')
    $line = Select-String -Path (Join-Path $Tmp 'checksums.txt') -Pattern " $Asset$" | Select-Object -First 1
    if (-not $line) { throw "$Asset not listed in checksums.txt" }
    $want = ($line.Line -split ' ')[0]
  } else {
    try {
      Invoke-WebRequest "$Base/$ChecksumSource" -OutFile (Join-Path $Tmp $ChecksumSource)
    } catch {
      throw "no checksum published for $Asset (use a newer release, or set `$env:ZGIT_NO_DESKTOP='1')"
    }
    $want = ((Get-Content (Join-Path $Tmp $ChecksumSource) -First 1) -split ' ')[0]
    if (-not $want) { throw "empty checksum file for $Asset" }
  }
  $got = (Get-FileHash (Join-Path $Tmp $Asset) -Algorithm SHA256).Hash.ToLower()
  if ($want -ne $got) { throw "checksum mismatch for $Asset" }

  $outDir = Join-Path $Tmp ([System.IO.Path]::GetFileNameWithoutExtension($Asset) + '-out')
  Expand-Archive -Path (Join-Path $Tmp $Asset) -DestinationPath $outDir -Force
  $bin = Join-Path $outDir $BinName
  if (-not (Test-Path $bin)) { throw "archive $Asset did not contain $BinName" }

  New-Item -ItemType Directory -Force -Path $Dest | Out-Null
  Copy-Item $bin (Join-Path $Dest $OutName) -Force
}

function Add-StartMenuShortcut {
  $startMenu = Join-Path ([Environment]::GetFolderPath('Programs')) 'Zgit.lnk'
  if (Test-Path $startMenu) { return }
  $shell = New-Object -ComObject WScript.Shell
  $link = $shell.CreateShortcut($startMenu)
  $link.TargetPath = Join-Path $Dest 'zgit-desktop.exe'
  $link.WorkingDirectory = $Dest
  $link.Description = 'Zgit desktop'
  $link.Save()
  Write-Host 'install: Start Menu shortcut created'
}

New-Item -ItemType Directory -Force -Path $Tmp | Out-Null
try {
  # Update mode: tracking latest and TUI already current.
  $current = Get-InstalledVersion
  if ((-not $Pinned) -and $current -and ($current -eq $Version.TrimStart('v'))) {
    if (($env:ZGIT_NO_DESKTOP -eq '1') -or (Test-DesktopPresent)) {
      Write-Host "install: zgit $Version already up to date"
      return
    }
    Write-Host "install: TUI $Version current, installing missing desktop app..."
    Install-Archive "zgit-desktop-$Version-windows-$Arch.zip" 'zgit.exe' 'zgit-desktop.exe' "zgit-desktop-$Version-windows-$Arch.zip.sha256"
    Add-StartMenuShortcut
    return
  }

  Install-Archive "zgit-$Version-windows-$Arch.zip" 'zgit.exe' 'zgit.exe'

  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $Dest) {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$Dest", 'User')
    Write-Host "install: added $Dest to user PATH (restart the terminal to pick it up)"
  }

  & (Join-Path $Dest 'zgit.exe') --help | Out-Null
  Write-Host "install: zgit $Version installed to $Dest\zgit.exe"

  if ($env:ZGIT_NO_DESKTOP -ne '1') {
    try {
      Install-Archive "zgit-desktop-$Version-windows-$Arch.zip" 'zgit.exe' 'zgit-desktop.exe' "zgit-desktop-$Version-windows-$Arch.zip.sha256"
      Add-StartMenuShortcut
      Write-Host "install: zgit-desktop $Version installed to $Dest\zgit-desktop.exe"
    } catch {
      Write-Host "install: warning: desktop app skipped ($($_.Exception.Message))"
    }
  }
}
finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
