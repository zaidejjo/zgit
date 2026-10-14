---
title: Getting Started
description: Install Zgit, launch the desktop or TUI, and make your first commit.
---

Welcome to **Zgit** — the Git client that combines a desktop app, a terminal TUI, and AI-assisted workflows.

## Quick Install

```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

Or use your package manager:

| Platform | Command |
|----------|---------|
| Arch Linux (AUR) | `yay -S zgit-desktop-bin` |
| Arch Linux (CLI) | `yay -S zgit-bin` |
| macOS (Homebrew) | `brew install zaidejjo/tap/zgit` |
| Windows (Scoop) | `scoop bucket add zgit https://github.com/zaidejjo/scoop-zgit && scoop install zgit` |
| Any (Go) | `go install github.com/zaidejjo/zgit/cmd/zgit@latest` |

## Launch

```bash
zgit desktop    # Launch the desktop app
zgit            # Launch the terminal TUI
```

## First Launch

When you run Zgit for the first time:

1. Config is created at `~/.config/zgit/config.yaml`
2. The dashboard shows your current repo status
3. If no repo is open, use **File > Open Repository** or `Ctrl+O`

## Desktop App

The desktop app (React + TypeScript + Wails v2) features:

- Real-time Git status with file watcher (`fsnotify`)
- Inline diff viewer with syntax highlighting
- AI commit message generation
- PR creation and management
- GitHub OAuth integration
- Custom themes with accent sliders
- Resizable panels and merge conflict editor

## Terminal TUI

The TUI (Bubble Tea) is for terminal-first workflows:

```bash
zgit
```

| Key | Action |
|-----|--------|
| `s` | Stage/unstage |
| `S` | Stage all |
| `c` | Commit |
| `p` | Push |
| `b` | Branch switcher |
| `l` | Log view |
| `d` | Diff view |
| `?` | Help |

## Next Steps

- [Set up AI providers](/docs/ai/provider-setup) for smart commit messages
- [Configure themes](/docs/configuration/theme-keybindings) and keybindings
- [Connect GitHub](/docs/configuration/git-profile) for PR workflows
- [Learn about AI PR generation](/docs/ai/smart-pr-commit)
