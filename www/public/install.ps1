# zgit installer for Windows (amd64).
#
# Run in PowerShell:
#   irm https://zgit.pages.dev/install.ps1 | iex
#
# Environment variables (optional):
#   $env:ZGIT_VERSION  pinned version tag, e.g. v0.1.1 (default: latest release)
#
# Installs to %LocalAppData%\zgit and adds it to the user PATH.
# The zip checksum is verified before installing anything.
$ErrorActionPreference = 'Stop'

$Repo = 'zaidejjo/zgit'
$Version = $env:ZGIT_VERSION
if (-not $Version) {
  $Version = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
  if (-not $Version) { throw 'could not resolve latest release (set $env:ZGIT_VERSION to pin one)' }
}

switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { $Arch = 'amd64' }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE (windows/amd64 only)" }
}

$Asset = "zgit-$Version-windows-$Arch.zip"
$Base = "https://github.com/$Repo/releases/download/$Version"
$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) "zgit-install"
New-Item -ItemType Directory -Force -Path $Tmp | Out-Null
try {
  Write-Host "install: downloading $Asset ($Version)..."
  Invoke-WebRequest "$Base/$Asset" -OutFile (Join-Path $Tmp $Asset)

  Write-Host 'install: verifying checksum...'
  Invoke-WebRequest "$Base/checksums.txt" -OutFile (Join-Path $Tmp 'checksums.txt')
  $line = Select-String -Path (Join-Path $Tmp 'checksums.txt') -Pattern " $Asset$" | Select-Object -First 1
  if (-not $line) { throw "$Asset not listed in checksums.txt" }
  $want = ($line.Line -split ' ')[0]
  $got = (Get-FileHash (Join-Path $Tmp $Asset) -Algorithm SHA256).Hash.ToLower()
  if ($want -ne $got) { throw "checksum mismatch for $Asset" }

  Expand-Archive -Path (Join-Path $Tmp $Asset) -DestinationPath (Join-Path $Tmp 'out') -Force
  $bin = Join-Path $Tmp 'out\zgit.exe'
  if (-not (Test-Path $bin)) { throw 'archive did not contain zgit.exe' }

  $dest = Join-Path $env:LocalAppData 'zgit'
  New-Item -ItemType Directory -Force -Path $dest | Out-Null
  Copy-Item $bin (Join-Path $dest 'zgit.exe') -Force

  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $dest) {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$dest", 'User')
    Write-Host "install: added $dest to user PATH (restart the terminal to pick it up)"
  }

  & (Join-Path $dest 'zgit.exe') --help | Out-Null
  Write-Host "install: zgit $Version installed to $dest\zgit.exe"
}
finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
