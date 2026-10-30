# Zgit Docs

## Installation

### Quick install script (recommended)

Installs both `zgit` (terminal) and `zgit-desktop`, with sha256 verification.
Re-run anytime to update both to the latest release.

```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

```powershell
irm https://zgit.pages.dev/install.ps1 | iex
```

| Env | Effect |
|-----|--------|
| `ZGIT_VERSION=vX.Y.Z` | Pin a version instead of latest |
| `ZGIT_NO_DESKTOP=1` | TUI only, skip the desktop app |
| `ZGIT_INSTALL_DIR=/path` | Custom install dir (bash script) |

### Pre-built binaries

Download from [GitHub Releases](https://github.com/zaidejjo/zgit/releases).

| Package | File |
|---------|------|
| Linux (TUI) | `zgit-v{VERSION}-linux-{arch}.tar.gz` |
| Linux (Desktop) | `zgit-desktop-v{VERSION}-linux-{arch}.tar.gz` |
| macOS (TUI) | `zgit-v{VERSION}-darwin-{arch}.tar.gz` |
| macOS (Desktop) | `zgit-desktop-v{VERSION}-darwin-{arch}.tar.gz` |
| Windows (TUI) | `zgit-v{VERSION}-windows-amd64.zip` |
| Windows (Desktop) | `zgit-desktop-v{VERSION}-windows-amd64.zip` |

### Debian / Ubuntu (APT)

```bash
curl -fsSL https://zaidejjo.github.io/apt/key.asc | sudo gpg --dearmor -o /usr/share/keyrings/zgit.gpg
echo "deb [signed-by=/usr/share/keyrings/zgit.gpg] https://zaidejjo.github.io/apt stable main" | sudo tee /etc/apt/sources.list.d/zgit.list
sudo apt update && sudo apt install zgit
```

Uninstall: `sudo apt remove zgit && sudo rm /etc/apt/sources.list.d/zgit.list /usr/share/keyrings/zgit.gpg`

### macOS (Homebrew)

```bash
brew tap zaidejjo/tap
brew trust zaidejjo/tap   # one time: third-party taps are untrusted by default
brew install --cask zaidejjo/tap/zgit
```

Uninstall: `brew uninstall --cask zgit`

### Arch Linux (AUR)

```bash
yay -S zgit-bin          # TUI
yay -S zgit-desktop-bin  # Desktop
```

### Flatpak (build from manifest)

Not on Flathub yet — build locally:

```bash
flatpak install flathub org.gnome.Platform//48 org.gnome.Sdk//48 \
  org.freedesktop.Sdk.Extension.golang//24.08 org.freedesktop.Sdk.Extension.node20//24.08
flatpak-builder --install --user build dist/flatpak/com.zaidejjo.zgit.yml
flatpak run com.zaidejjo.zgit
```

### Build from source

```bash
go install golang.org/dl/go1.26@latest
git clone https://github.com/zaidejjo/zgit.git
cd zgit
make build          # TUI binary → ./zgit
make install        # go install → $GOPATH/bin/zgit

# Desktop app (requires Wails CLI + webkit2gtk-4.0)
cd desktop
wails build
```

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

## Usage

```bash
zgit                 # open current directory as a repo
zgit -C /path/to/repo
zgit tui             # launch the terminal UI
zgit status          # working tree status
zgit log --follow -f src/main.go
zgit blame src/main.go
zgit worktree list   # linked working trees
zgit-desktop         # launch the desktop app
```

Desktop pages: Status, Log, Branches, Worktrees, History (commits + blame),
PRs, Issues, Actions, Remotes, Tags, AI chat. `Ctrl+K` opens the command palette.

## Configuration

Config file: `~/.config/zgit/config.toml` (a legacy `config.yaml` is migrated
once and kept as `config.yaml.bak`).

```toml
[ai]
provider = "openrouter"

[ai.providers.openrouter]
api_key = "${OPENROUTER_API_KEY}"
model = "openai/gpt-4o"

[tui]
theme = "dark"    # dark | light
glyphs = "auto"   # auto | nerd | unicode | ascii
```

### Terminal UI

The TUI is a keyboard-first cockpit: panes are separated by rules rather than
boxes, rows are aligned tables (graph, hash, age, subject, refs), and the
status bar is a prioritised list that drops the least useful segment first.

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | cycle panes |
| `0`–`9` | open screen (Refs, Stash, Worktrees, Notes, Actions, Repos, Issues, PRs) |
| `/` | filter the focused list |
| `L` | toggle light/dark |
| `Ctrl+K` | command palette |
| `?` | full keymap |

### Icons

`glyphs = "auto"` picks a symbol set per terminal: Nerd Font icons only where
the terminal ships a symbol fallback (Ghostty, WezTerm) or where `NERD_FONT` is
set, and safe Unicode everywhere else — a missing icon beats a box drawn in the
middle of the UI. Override it with `glyphs = "nerd" | "unicode" | "ascii"`, the
`ZGIT_TUI_GLYPHS` environment variable, or the **Cycle Icon Set** palette
command (the choice is persisted).
