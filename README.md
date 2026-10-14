<p align="center">
  <img src="images/logo.svg" width="120" alt="zgit logo" />
</p>

<h1 align="center">zgit</h1>

<p align="center">
  <a href="https://github.com/zaidejjo/zgit/actions/workflows/ci.yml"><img src="https://github.com/zaidejjo/zgit/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg" alt="License: Apache-2.0" /></a>
  <a href="https://github.com/zaidejjo/zgit/releases"><img src="https://img.shields.io/github/v/release/zaidejjo/zgit?color=orange" alt="Latest release" /></a>
  <img src="https://img.shields.io/github/go-mod/go-version/zaidejjo/zgit" alt="Go version" />
</p>

<p align="center">
  A modern, fast Git &amp; GitHub client combining local Git operations with GitHub CLI/API features (PRs, Issues, Actions, Reviews) into a clean, non-cluttered interface.
</p>

<p align="center">
  Ships as <b>both</b> a terminal UI (TUI) and a desktop application.
</p>

## Features

- **Full Git integration** — status, log, branches, tags, remotes
- **Worktrees** — manage linked working trees across CLI and desktop
- **File history & blame** — per-line attribution, rename following, pickaxe (`-S`/`-G`) search
- **Commit management** — conventional commit tags, file checkboxes, force push options
- **Interactive rebase** — drag-and-drop commit reordering, cherry-pick, squash
- **3-way merge editor** — resolve conflicts inline with a visual editor
- **GitHub PRs** — create, merge, view, list with status badges
- **GitHub Issues** — create, close, detail view
- **AI assistant** — commit message generation, PR descriptions, chat-based git operations
- **Dual UI** — terminal (Bubble Tea) for devs, desktop (Wails + React + shadcn/ui) for rich interaction
- **Diff viewer** — unified/split toggle, hunk staging, syntax highlighting
- **Remote management** — add, rename, remove, set URL
- **Global undo** — revert the last git action
- **Multi-platform** — Linux, macOS and Windows via tarballs, APT, Homebrew, AUR and Flatpak

## Installation

### Quick install script (TUI only)

Linux / macOS (verifies sha256 checksums, TUI binary only):

```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

Windows PowerShell (verifies sha256 checksums, TUI binary only):

```powershell
irm https://zgit.pages.dev/install.ps1 | iex
```

Pin a version: `ZGIT_VERSION=v0.1.1 curl -fsSL https://zgit.pages.dev/install | bash`
(`$env:ZGIT_VERSION='v0.1.1'` on Windows).

### Pre-built binaries

Download the latest release from [GitHub Releases](https://github.com/zaidejjo/zgit/releases).

| Package | Command |
|---------|---------|
| Linux (TUI) | `zgit-v{VERSION}-linux-{arch}.tar.gz` |
| Linux (Desktop) | `zgit-desktop-v{VERSION}-linux-{arch}.tar.gz` |
| macOS (TUI) | `zgit-v{VERSION}-darwin-{arch}.tar.gz` |
| macOS (Desktop) | `zgit-desktop-v{VERSION}-darwin-{arch}.tar.gz` |
| Windows (TUI) | `zgit-v{VERSION}-windows-{arch}.exe` |
| Windows (Desktop) | `zgit-desktop-v{VERSION}-windows-{arch}.exe` (in .zip) |

### Build from source

```bash
# Prerequisites
go install golang.org/dl/go1.26@latest

# Clone and build
git clone https://github.com/zaidejjo/zgit.git
cd zgit
make build          # TUI binary → ./zgit
make install        # go install → $GOPATH/bin/zgit

# Desktop app (requires Wails CLI + webkit2gtk-4.0)
cd desktop
wails build
```

### Arch Linux (AUR)

```bash
# TUI
yay -S zgit-bin

# Desktop
yay -S zgit-desktop-bin
```

### Debian / Ubuntu (APT)

```bash
# Add the signing key and repository
curl -fsSL https://zaidejjo.github.io/apt/key.asc | sudo gpg --dearmor -o /usr/share/keyrings/zgit.gpg
echo "deb [signed-by=/usr/share/keyrings/zgit.gpg] https://zaidejjo.github.io/apt stable main" | sudo tee /etc/apt/sources.list.d/zgit.list

# Install (amd64 and arm64 supported)
sudo apt update && sudo apt install zgit

# Verify
zgit --help

# Uninstall
sudo apt remove zgit
sudo rm /etc/apt/sources.list.d/zgit.list /usr/share/keyrings/zgit.gpg
```

### macOS (Homebrew)

```bash
brew install --cask zaidejjo/tap/zgit

# Verify
zgit --help

# Uninstall
brew uninstall --cask zgit
```

### Flatpak (build from manifest)

The app is not on Flathub yet — build it locally from the manifest:

```bash
flatpak install flathub org.gnome.Platform//48 org.gnome.Sdk//48 \
  org.freedesktop.Sdk.Extension.golang//24.08 org.freedesktop.Sdk.Extension.node20//24.08
flatpak-builder --install --user build dist/flatpak/com.zaidejjo.zgit.yml
flatpak run com.zaidejjo.zgit
```

## Quick start

```bash
# Open current directory as a git repo
zgit

# Open a specific repo
zgit -C /path/to/repo

# Launch the TUI
zgit tui

# Desktop builds launch the GUI by default
./zgit-desktop
```

## Configuration

Config file: `~/.config/zgit/config.yaml`

```yaml
ai:
  default_provider: openrouter
  providers:
    openrouter:
      api_key: ${OPENROUTER_API_KEY}
      model: openai/gpt-4o
```

## Development

```bash
make dev    # fmt → vet → test → build
make test   # run all tests
make lint   # golangci-lint
```

## License

[Apache License 2.0](LICENSE) © 2026 zaidejjo
