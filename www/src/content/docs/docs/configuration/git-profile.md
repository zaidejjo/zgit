---
title: Git Profile
description: Git identity, GitHub integration, and profile settings.
---

## Identity

The **Git** tab in Settings manages:

- **User Name** — Sets `git config --global user.name`
- **User Email** — Sets `git config --global user.email`
- **GPG Signing** — Enable commit signing with GPG key
- **Default Branch** — Set default branch name (e.g., `main`)

## GitHub Integration

Connect your GitHub account via OAuth in **Settings → Profile**:

1. Click **Connect GitHub**
2. Authorize in the browser
3. Your avatar and stats appear in the Profile tab

## Profile Tab

The **Profile** tab (accessible from sidebar or Settings) shows:

- **Avatar** and display name
- **Stats grid**: Followers, Following, Public Repos
- **Identity card**: Display Name syncs to git `user.name`
- **Plan info**: Free/Pro badge
- **Quick actions**: Open profile, manage repos, view PRs
- **Disconnect**: Remove GitHub connection

## Profile Card Structure

```
┌─────────────────────────────┐
│  [Avatar]  Display Name     │
│            @username        │
│                             │
│  Followers  Following  Repos│
│    128       47        32   │
│                             │
│  [Identity Card]            │
│  Display Name: Zaid         │
│  (syncs to git user.name)   │
│                             │
│  [Disconnect]               │
└─────────────────────────────┘
```
