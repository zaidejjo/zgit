---
title: Branch, Stash & Log
description: Daily Git operations — commits, branches, stashes, and history.
---

## Commits

### Desktop

1. Stage files with `Ctrl+S` (file) or `Ctrl+Shift+S` (all)
2. Write a commit message (or use AI Generate)
3. Commit with `Ctrl+Enter` or `Ctrl+Shift+Enter` (commit & push)

### TUI

| Key | Action |
|-----|--------|
| `s` | Stage/unstage focused file |
| `S` | Stage all |
| `c` | Commit with message prompt |
| `C` | Commit & push |

## Branches

### Desktop

- **Switch**: `Ctrl+B` opens branch switcher with search
- **Create**: `Ctrl+Shift+B` or click **+** in branch list
- **Delete**: Right-click branch → Delete (with confirmation)
- **Rebase**: Interactive rebase from log context menu

### TUI

| Key | Action |
|-----|--------|
| `b` | Open branch switcher |
| `B` | Create new branch |
| `Ctrl+D` | Delete branch |

## Stashes

### Desktop

Stash management is available from the sidebar **Stashes** section:

- **Stash**: Save working directory changes
- **Pop**: Apply and drop the latest stash
- **Apply**: Apply without dropping
- **Drop**: Discard a stash
- **List**: View all stashes with timestamps

### TUI

| Key | Action |
|-----|--------|
| `g` | Stash changes |
| `G` | List/pop stashes |

## Log & History

### Desktop Log View

- **Filter**: Search by message, author, branch, date
- **Graph**: Visual commit graph with drag-and-drop reorder
- **Details**: Click a commit to see diff, metadata, branches/tags
- **Revert**: Right-click → Revert commit
- **Reset**: Right-click → Reset (soft/mixed/hard)

### TUI Log

| Key | Action |
|-----|--------|
| `l` | Log view |
| `Enter` | Show commit diff |
| `/` | Search commits |
| `r` | Revert selected commit |

## Merge Conflicts

When conflicts arise during merge/rebase/pull:

1. Conflict files are listed in the **Conflicts** panel
2. Click a file to open the **Merge Editor**
3. The editor shows three-pane layout: **Base** | **Ours** | **Theirs**
4. Click conflict markers to resolve, save, and mark as resolved
5. Continue the merge/rebase when all conflicts are resolved

## Undo

Zgit tracks undo state for most operations:

- **Undo Banner** appears after destructive actions
- Supports undo for: commits (amend/reset), stashes, branch operations
- Each undo step shows a description of what will be undone
