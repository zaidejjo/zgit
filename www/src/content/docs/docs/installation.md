---
title: Installation
description: Install Zgit on Linux, macOS, or Windows.
---

## Quick Install

```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

## Linux

### Arch Linux (AUR)

Desktop app bundle (recommended):

```bash
yay -S zgit-desktop-bin
```

CLI-only (no desktop frontend):

```bash
yay -S zgit-bin
```

### Homebrew (Linux)

```bash
brew install zaidejjo/tap/zgit
```

### Build from Source

```bash
git clone https://github.com/zaidejjo/zgit.git
cd zgit
make build
sudo mv zgit /usr/local/bin/
```

Desktop dependencies:

```bash
# Ubuntu/Debian
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

# Arch
sudo pacman -S gtk3 webkit2gtk

# Fedora
sudo dnf install gtk3-devel webkit2gtk3-devel
```

## macOS

### Homebrew

```bash
brew install zaidejjo/tap/zgit
```

### Go Install

```bash
go install github.com/zaidejjo/zgit/cmd/zgit@latest
```

## Windows

### Scoop

```bash
scoop bucket add zgit https://github.com/zaidejjo/scoop-zgit
scoop install zgit
```

### Winget

```bash
winget install zgit
```

### Go Install

```powershell
go install github.com/zaidejjo/zgit/cmd/zgit@latest
```

## Verify

```bash
zgit version
```

Expected output: `zgit v0.2.0 (commit abc1234)`
