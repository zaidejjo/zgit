<p align="center">
  <img src="images/logo.svg" width="120" alt="Zgit logo" />
</p>

<h1 align="center">Zgit</h1>

<p align="center">
  <a href="https://github.com/zaidejjo/zgit/actions/workflows/ci.yml"><img src="https://github.com/zaidejjo/zgit/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg" alt="License: Apache-2.0" /></a>
  <a href="https://github.com/zaidejjo/zgit/releases"><img src="https://img.shields.io/github/v/release/zaidejjo/zgit?color=orange" alt="Latest release" /></a>
  <img src="https://img.shields.io/github/go-mod/go-version/zaidejjo/zgit" alt="Go version" />
</p>

<p align="center">
  <b>Git without the headaches.</b>
</p>

<p align="center">
  Terminal UI and desktop app in one fast package.
</p>

## Installation

One script installs <b>both</b> the terminal client (`zgit`) and the desktop app (`zgit-desktop`). Re-run it anytime to update both. Everything is sha256-verified.

```bash
curl -fsSL https://zgit.pages.dev/install | bash
```

```powershell
irm https://zgit.pages.dev/install.ps1 | iex
```

TUI only: `ZGIT_NO_DESKTOP=1 curl -fsSL https://zgit.pages.dev/install | bash`
Pin a version: `ZGIT_VERSION=v0.1.1 curl -fsSL https://zgit.pages.dev/install | bash`

Prefer a package manager? [APT, Homebrew, AUR, Flatpak and tarballs →](DOCS.md#installation)

## Quick start

```bash
zgit                 # open current directory
zgit -C /path/to/repo
zgit tui             # terminal UI
zgit-desktop         # desktop app
```

New here? Start with [DOCS.md](DOCS.md). Want to hack on it? Read [CONTRIBUTING.md](CONTRIBUTING.md).


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue.

## License

[Apache License 2.0](LICENSE) © 2026 zaidejjo
