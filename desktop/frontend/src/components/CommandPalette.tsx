import { useEffect, useRef, useState, useCallback, useMemo } from "react";
import {
  GitCommitHorizontal, History, GitBranch, GitPullRequest,
  CircleDot, Play, Globe, Tag, Settings, Bell,
  Upload, Download, RotateCcw, Archive, Undo2,
  Sparkles, Search, ArrowRight, GitFork, Folder,
  FileText, Star, Plus, Trash2, Merge, Palette, LayoutDashboard, FolderKanban,
} from "lucide-react";
import { useAppStore, type Theme, THEMES } from "@/store/app";
import { cn } from "@/lib/utils";

interface Command {
  id: string;
  label: string;
  description?: string;
  icon: React.ReactNode;
  action: () => void;
  category: string;
  shortcut?: string;
}

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  onNavigate: (tabId: string) => void;
}

/** Subsequence fuzzy score (higher = better), null when no match.
 *  Rewards prefixes, word starts and consecutive runs; prefers short targets. */
function fuzzyScore(rawQuery: string, rawText: string): number | null {
  const q = rawQuery.toLowerCase();
  const text = rawText.toLowerCase();
  if (!q) return 0;
  let score = 0;
  let ti = 0;
  let consec = 0;
  for (let qi = 0; qi < q.length; qi++) {
    const found = text.indexOf(q[qi], ti);
    if (found === -1) return null;
    const prev = found > 0 ? text[found - 1] : "";
    if (found === ti) {
      consec++;
      score += 10 + consec * 5;
    } else {
      consec = 0;
      score += 5;
      if (found === 0 || prev === "/" || prev === "-" || prev === "_" || prev === " " || prev === ".") {
        score += 8;
      }
    }
    if (qi === 0 && found === 0) score += 15;
    ti = found + 1;
  }
  return score - text.length * 0.1;
}

export default function CommandPalette({ open, onClose, onNavigate }: CommandPaletteProps) {
  const {
    gitFetch, gitPull, gitPush,
    stageAll, unstageAll,
    stashPush,
    fetchReflog, undoLastAction, undoDescription,
    toggleAIPanel,
    fetchStashes, stashPop, stashApply, stashDrop,
    stashes, theme, setTheme,
    userPreferences,
    recentRepos, fetchRecentRepos, openRepo, repoPath,
    branches, currentBranch, fetchBranches, checkoutBranch,
    projects, fetchProjects,
  } = useAppStore();
  const aiShortcut = userPreferences?.keybindings?.toggle_ai_panel || "Ctrl+Shift+A";

  const [query, setQuery] = useState("");
  const [selectedIdx, setSelectedIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Reset state on open + make sure dynamic sources are loaded
  useEffect(() => {
    if (open) {
      setQuery("");
      setSelectedIdx(0);
      setTimeout(() => inputRef.current?.focus(), 50);
      const s = useAppStore.getState();
      if (s.recentRepos.length === 0) s.fetchRecentRepos().catch(() => {});
      if (s.branches.length === 0) s.fetchBranches().catch(() => {});
      if (s.projects.length === 0) s.fetchProjects().catch(() => {});
    }
  }, [open]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  const commands: Command[] = useMemo(() => [
    // Repositories — most useful first
    ...recentRepos.slice(0, 6).map((r) => {
      const name = r.split("/").pop() || r;
      const current = r === repoPath;
      return {
        id: `repo-${r}`,
        label: current ? `${name} (current)` : `Open ${name}`,
        description: r,
        icon: <Folder className="w-4 h-4" />,
        action: () => { if (!current) openRepo(r); onClose(); },
        category: "Repositories",
      };
    }),

    // Branches of the current repo
    ...branches.slice(0, 6).map((b) => {
      const current = b.name === currentBranch;
      return {
        id: `branch-${b.name}`,
        label: current ? `${b.name} (current)` : `Switch to ${b.name}`,
        description: [b.upstream, b.latest_msg].filter(Boolean).join(" · ") || undefined,
        icon: <GitBranch className="w-4 h-4" />,
        action: () => { if (!current) checkoutBranch(b.name); onClose(); },
        category: "Branches",
      };
    }),

    // Projects
    ...projects.map((p) => ({
      id: `project-${p.id}`,
      label: `Project: ${p.name}`,
      description: `${p.repos.length} ${p.repos.length === 1 ? "repo" : "repos"}`,
      icon: <FolderKanban className="w-4 h-4" />,
      action: () => {
        useAppStore.setState({ activeProjectId: p.id });
        onNavigate("projects");
        onClose();
      },
      category: "Projects",
    })),

    // Navigation
    { id: "nav-status",   label: "Go to Status",     icon: <GitCommitHorizontal className="w-4 h-4" />, action: () => { onNavigate("status"); onClose(); }, category: "Navigation" },
    { id: "nav-log",      label: "Go to Log",        icon: <History className="w-4 h-4" />, action: () => { onNavigate("log"); onClose(); }, category: "Navigation" },
    { id: "nav-branches", label: "Go to Branches",   icon: <GitBranch className="w-4 h-4" />, action: () => { onNavigate("branches"); onClose(); }, category: "Navigation" },
    { id: "nav-worktrees", label: "Go to Worktrees", icon: <GitFork className="w-4 h-4" />, action: () => { onNavigate("worktrees"); onClose(); }, category: "Navigation" },
    { id: "nav-dashboard", label: "Go to Dashboard", icon: <LayoutDashboard className="w-4 h-4" />, action: () => { onNavigate("dashboard"); onClose(); }, category: "Navigation" },
    { id: "nav-projects", label: "Go to Projects", icon: <FolderKanban className="w-4 h-4" />, action: () => { onNavigate("projects"); onClose(); }, category: "Navigation" },
    { id: "nav-history", label: "Go to File History", icon: <History className="w-4 h-4" />, action: () => { onNavigate("history"); onClose(); }, category: "Navigation" },
    { id: "nav-prs",      label: "Go to Pull Requests", icon: <GitPullRequest className="w-4 h-4" />, action: () => { onNavigate("pull-requests"); onClose(); }, category: "Navigation" },
    { id: "nav-issues",   label: "Go to Issues",     icon: <CircleDot className="w-4 h-4" />, action: () => { onNavigate("issues"); onClose(); }, category: "Navigation" },
    { id: "nav-actions",  label: "Go to Actions",    icon: <Play className="w-4 h-4" />, action: () => { onNavigate("actions"); onClose(); }, category: "Navigation" },
    { id: "nav-notifications", label: "Go to Notifications", icon: <Bell className="w-4 h-4" />, action: () => { onNavigate("notifications"); onClose(); }, category: "Navigation" },
    { id: "nav-remotes",  label: "Go to Remotes",    icon: <Globe className="w-4 h-4" />, action: () => { onNavigate("remotes"); onClose(); }, category: "Navigation" },
    { id: "nav-tags",     label: "Go to Tags",       icon: <Tag className="w-4 h-4" />, action: () => { onNavigate("tags"); onClose(); }, category: "Navigation" },
    { id: "nav-settings", label: "Go to Settings",   icon: <Settings className="w-4 h-4" />, action: () => { onNavigate("settings"); onClose(); }, category: "Navigation" },

    // Git actions
    { id: "git-fetch",  label: "Fetch",  description: "Fetch from remote",  icon: <Download className="w-4 h-4" />, action: () => { gitFetch(); onClose(); }, category: "Git", shortcut: "F" },
    { id: "git-pull",   label: "Pull",   description: "Pull from upstream", icon: <Download className="w-4 h-4" />, action: () => { gitPull(false); onClose(); }, category: "Git" },
    { id: "git-push",   label: "Push",   description: "Push to remote",     icon: <Upload className="w-4 h-4" />, action: () => { gitPush(); onClose(); }, category: "Git" },
    { id: "stage-all",  label: "Stage All",   description: "Stage all changes",  icon: <Plus className="w-4 h-4" />, action: () => { stageAll(); onClose(); }, category: "Git" },
    { id: "unstage-all", label: "Unstage All", description: "Unstage all changes", icon: <Trash2 className="w-4 h-4" />, action: () => { unstageAll(); onClose(); }, category: "Git" },
    { id: "stash",       label: "Stash",    description: "Stash working directory", icon: <Archive className="w-4 h-4" />, action: () => { stashPush(); onClose(); }, category: "Git" },
    { id: "undo",        label: "Undo",     description: undoDescription || "Undo last git action", icon: <Undo2 className="w-4 h-4" />, action: () => { undoLastAction(); onClose(); }, category: "Git" },

    // AI
    { id: "ai-panel",    label: "Toggle AI Assistant", description: "Open or close AI panel", icon: <Sparkles className="w-4 h-4" />, action: () => { toggleAIPanel(); onClose(); }, category: "AI", shortcut: aiShortcut },

    // Theme switching
    ...THEMES.map((t) => ({
      id: `theme-${t.id}`,
      label: `Theme: ${t.label}`,
      description: `Switch to ${t.label} theme`,
      icon: <Palette className="w-4 h-4" />,
      action: () => { setTheme(t.id); onClose(); },
      category: "Appearance",
      shortcut: theme === t.id ? "active" : undefined,
    })),
  ], [onNavigate, onClose, gitFetch, gitPull, gitPush, stageAll, unstageAll, stashPush, undoLastAction, undoDescription, toggleAIPanel, theme, setTheme, aiShortcut, recentRepos, repoPath, openRepo, branches, currentBranch, checkoutBranch, projects]);

  // Fuzzy filter with #branch / @repo prefix modes, ranked by score.
  // Empty query shows everything grouped (dynamic sources first).
  const filtered = useMemo(() => {
    const raw = query.trim();
    if (!raw) return commands;
    let q = raw;
    let onlyCategory: string | null = null;
    if (q.startsWith("#")) { onlyCategory = "Branches"; q = q.slice(1).trim(); }
    else if (q.startsWith("@")) { onlyCategory = "Repositories"; q = q.slice(1).trim(); }
    else if (q.startsWith(">")) { q = q.slice(1).trim(); }
    if (!q) {
      return onlyCategory ? commands.filter((c) => c.category === onlyCategory) : commands;
    }
    const scored: { cmd: Command; score: number; idx: number }[] = [];
    commands.forEach((c, idx) => {
      if (onlyCategory && c.category !== onlyCategory) return;
      const text = c.description ? `${c.label} ${c.description}` : c.label;
      const s = fuzzyScore(q, text);
      if (s !== null) {
        scored.push({ cmd: c, score: s, idx });
      } else if (!onlyCategory && c.category.toLowerCase().includes(q.toLowerCase())) {
        scored.push({ cmd: c, score: 0.5, idx });
      }
    });
    scored.sort((a, b) => b.score - a.score || a.idx - b.idx);
    return scored.map((s) => s.cmd);
  }, [commands, query]);

  // Keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIdx((prev) => Math.min(prev + 1, filtered.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIdx((prev) => Math.max(prev - 1, 0));
    } else if (e.key === "Enter" && filtered[selectedIdx]) {
      e.preventDefault();
      filtered[selectedIdx].action();
    }
  }, [filtered, selectedIdx]);

  // Scroll selected item into view
  useEffect(() => {
    const el = listRef.current?.children[selectedIdx] as HTMLElement | undefined;
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIdx]);

  if (!open) return null;

  // Group filtered commands by category
  const grouped = filtered.reduce<Record<string, Command[]>>((acc, cmd) => {
    if (!acc[cmd.category]) acc[cmd.category] = [];
    acc[cmd.category].push(cmd);
    return acc;
  }, {});

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 command-overlay animate-in fade-in duration-150"
        onClick={onClose}
      />

      {/* Palette */}
      <div className="fixed left-1/2 top-[15%] -translate-x-1/2 z-50 w-[520px] max-w-[90vw] rounded-xl glass shadow-2xl border-border/60 animate-in fade-in zoom-in-95 duration-200">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-border/30">
          <Search className="w-4 h-4 text-muted-foreground shrink-0" />
          <input
            ref={inputRef}
            type="text"
            placeholder="Search actions, #branches, @repos..."
            value={query}
            onChange={(e) => { setQuery(e.target.value); setSelectedIdx(0); }}
            onKeyDown={handleKeyDown}
            className="flex-1 bg-transparent text-sm placeholder:text-muted-foreground outline-none"
          />
          <kbd className="hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono bg-muted/50 text-muted-foreground">
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-80 overflow-y-auto p-2 space-y-1">
          {Object.entries(grouped).map(([category, cmds]) => (
            <div key={category}>
              <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-muted-foreground font-medium">
                {category}
              </div>
              {cmds.map((cmd, idx) => {
                const globalIdx = filtered.indexOf(cmd);
                return (
                  <button
                    key={cmd.id}
                    className={cn(
                      "flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm transition-colors text-left",
                      globalIdx === selectedIdx
                        ? "bg-accent/60 text-foreground"
                        : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
                    )}
                    onClick={() => cmd.action()}
                    onMouseEnter={() => setSelectedIdx(globalIdx)}
                  >
                    <span className="shrink-0 text-muted-foreground">{cmd.icon}</span>
                    <div className="flex-1 min-w-0">
                      <div className="truncate font-medium">{cmd.label}</div>
                      {cmd.description && (
                        <div className="text-[11px] text-muted-foreground/70 truncate">
                          {cmd.description}
                        </div>
                      )}
                    </div>
                    {cmd.shortcut && (
                      <kbd className="shrink-0 text-[10px] font-mono px-1.5 py-0.5 rounded bg-muted/40 text-muted-foreground">
                        {cmd.shortcut}
                      </kbd>
                    )}
                  </button>
                );
              })}
            </div>
          ))}

          {filtered.length === 0 && (
            <div className="px-3 py-8 text-center text-sm text-muted-foreground">
              No commands match "{query}"
            </div>
          )}
        </div>

        {/* Hints */}
        <div className="flex items-center gap-3 px-4 py-2 border-t border-border/30 text-[10px] text-muted-foreground">
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted/50 font-mono">↑↓</kbd> navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted/50 font-mono">↵</kbd> run
          </span>
          <span className="hidden sm:flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted/50 font-mono">#</kbd> branches
          </span>
          <span className="hidden sm:flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded bg-muted/50 font-mono">@</kbd> repos
          </span>
        </div>
      </div>
    </>
  );
}
