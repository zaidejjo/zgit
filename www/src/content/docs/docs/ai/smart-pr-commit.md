---
title: Smart PR & AI Commits
description: AI-powered commit messages and PR generation.
---

## AI Commit Messages

Zgit can generate meaningful commit messages from your staged changes.

1. Stage your changes (`Ctrl+S`)
2. Click **Generate** in the commit panel (or `Ctrl+Shift+G`)
3. The AI analyzes the diff and produces a Conventional Commits message
4. Edit or accept

Example output:

```
feat: Add encrypted API key storage

- Implements AES-256-GCM encryption for provider API keys
- Auto-generates encryption key on first use
- Adds ProviderStatus struct with masked key display
```

The prompt uses a **Senior Dev** persona to ensure production-quality output.

## Smart PR Generation

### Three-Dot Diff

The PR generator uses a **three-dot diff** (`base...head`) to show only commits unique to the head branch:

```
git merge-base base head   # find merge base
git diff base...head       # three-dot diff
```

This avoids including commits already on the base branch, producing a focused PR description.

### PR Dialog

1. Navigate to the **Pull Requests** view
2. Click **New PR** (`Ctrl+N`)
3. Auto-detected branches:
   - **Base**: defaults to `main`/`master`
   - **Head**: defaults to `currentBranch`
4. Click **Generate with AI** (`Ctrl+Shift+G`) in the title row
5. Review auto-filled title and description
6. Toggle **Draft PR** if needed
7. Click **Create PR**

### Prompt Structure

The AI receives:

```
You are a Senior Developer reviewing a pull request.

Base branch: {base}
Head branch: {head}

Git diff (base...head):
{diff}

Generate a PR title and description following Conventional Commits...
```

### PR View

The Pull Requests page displays:

- **Open PRs** — Active with CI status indicators
- **Merged PRs** — Successfully merged
- **Closed PRs** — Declined or abandoned
- Each PR card shows title, description, branch labels, CI status, reviews

### CLI

```bash
zgit pr create --base main --head feature-branch --generate
```
