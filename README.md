<div align="center">

<img src="./images/logo.png" alt="zgit logo" width="150" />

# Zgit

[![Build](https://github.com/zaidejjo/zgit/actions/workflows/build.yml/badge.svg)](https://github.com/zaidejjo/zgit/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
![Go Version](https://img.shields.io/github/go-mod/go-version/zaidejjo/zgit)
![Latest Release](https://img.shields.io/github/v/release/zaidejjo/zgit)

A modern, fast Git & GitHub client combining local Git operations with GitHub CLI/API features (PRs, Issues, Actions, Reviews) into a clean, non-cluttered interface.

Ships as **both** a terminal UI (TUI) and a desktop application.

</div>

---

## Installation

### macos/Linux
```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

### Arch Linux (AUR)

```bash
# TUI
yay -S zgit-bin

# Desktop
yay -S zgit-desktop-bin
```

### Flatpak

```bash
flatpak install com.zaidejjo.zgit
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

## Features

- **Full Git integration** — status, log, branches, tags, remotes
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


## Development

```bash
make dev    # fmt → vet → test → build
make test   # run all tests
make lint   # golangci-lint
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue.

## License

[Apache License 2.0](LICENSE) © 2026 zaidejjo
