---
title: Theme & Keybindings
description: Customize appearance and keyboard shortcuts.
---

## Theme Customization

The **Theme** tab in Settings provides:

- **Accent Color Slider**: Pick any hue. The UI updates immediately with glass morphism effects, semantic CSS variables, and selection highlight colors.
- **Brightness Slider**: Adjust overall UI brightness. Supports opacity-safe theme baselines.
- **Glass Opacity**: Control backdrop-blur intensity for panels and cards.
- **Border Radius**: Toggle between sharp and rounded UI.

### Theme Engine

Themes are powered by CSS custom properties. Each component uses semantic tokens like `--git-added`, `--git-modified`, `--git-deleted`, `--pr-open`, `--pr-closed`, `--pr-merged`.

Theme files are loaded from `~/.config/zgit/themes/`. Built-in presets:

- **Default Dark** (OLED)
- **Glass Midnight**
- **Monochrome Slate**

## Keybinding Editor

The **Keybindings** tab provides a visual editor for all shortcuts.

### Desktop App

| Shortcut | Action |
|----------|--------|
| `Ctrl+K` | Command palette |
| `Ctrl+,` | Open settings |
| `Ctrl+S` | Stage file |
| `Ctrl+Shift+S` | Stage all |
| `Ctrl+Enter` | Commit |
| `Ctrl+Shift+Enter` | Commit & push |
| `Ctrl+P` | Push |
| `Ctrl+Shift+P` | Pull |
| `Ctrl+B` | Branch switcher |
| `Ctrl+N` | New PR |
| `Ctrl+Shift+G` | Generate PR with AI |
| `Ctrl+1-5` | Switch tabs |

### Terminal TUI

| Key | Action |
|-----|--------|
| `s` | Stage/unstage |
| `S` | Stage all |
| `c` | Commit |
| `p` | Push |
| `b` | Switch branch |
| `l` | View log |
| `d` | Show diff |
| `?` | Help |

### Customization

All keybindings are editable in **Settings → Keybindings**. Changes take effect immediately. Use **Reset to Defaults** to restore.
