import { useEffect, useRef, useState, useCallback, useMemo } from "react";
import {
  GitCommitHorizontal, History, GitBranch, GitPullRequest,
  CircleDot, Play, Globe, Tag, Settings,
  Upload, Download, RotateCcw, Archive, Undo2,
  Sparkles, Search, ArrowRight, GitFork,
  FileText, Star, Plus, Trash2, Merge, Palette,
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

export default function CommandPalette({ open, onClose, onNavigate }: CommandPaletteProps) {
  const {
    gitFetch, gitPull, gitPush,
    stageAll, unstageAll,
    stashPush,
    fetchReflog, undoLastAction, undoDescription,
    toggleAIPanel,
    fetchStashes, stashPop, stashApply, stashDrop,
    stashes, theme, setTheme,
  } = useAppStore();

  const [query, setQuery] = useState("");
  const [selectedIdx, setSelectedIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Reset state on open
  useEffect(() => {
    if (open) {
      setQuery("");
      setSelectedIdx(0);
      setTimeout(() => inputRef.current?.focus(), 50);
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
    // Navigation
    { id: "nav-status",   label: "Go to Status",     icon: <GitCommitHorizontal className="w-4 h-4" />, action: () => { onNavigate("status"); onClose(); }, category: "Navigation" },
    { id: "nav-log",      label: "Go to Log",        icon: <History className="w-4 h-4" />, action: () => { onNavigate("log"); onClose(); }, category: "Navigation" },
    { id: "nav-branches", label: "Go to Branches",   icon: <GitBranch className="w-4 h-4" />, action: () => { onNavigate("branches"); onClose(); }, category: "Navigation" },
    { id: "nav-prs",      label: "Go to Pull Requests", icon: <GitPullRequest className="w-4 h-4" />, action: () => { onNavigate("pull-requests"); onClose(); }, category: "Navigation" },
    { id: "nav-issues",   label: "Go to Issues",     icon: <CircleDot className="w-4 h-4" />, action: () => { onNavigate("issues"); onClose(); }, category: "Navigation" },
    { id: "nav-actions",  label: "Go to Actions",    icon: <Play className="w-4 h-4" />, action: () => { onNavigate("actions"); onClose(); }, category: "Navigation" },
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
    { id: "ai-panel",    label: "Toggle AI Assistant", description: "Open or close AI panel", icon: <Sparkles className="w-4 h-4" />, action: () => { toggleAIPanel(); onClose(); }, category: "AI", shortcut: "Ctrl+Shift+A" },

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
  ], [onNavigate, onClose, gitFetch, gitPull, gitPush, stageAll, unstageAll, stashPush, undoLastAction, undoDescription, toggleAIPanel, theme, setTheme]);

  // Filter commands by query
  const filtered = useMemo(() => {
    if (!query.trim()) return commands;
    const q = query.toLowerCase();
    return commands.filter(
      (c) =>
        c.label.toLowerCase().includes(q) ||
        c.description?.toLowerCase().includes(q) ||
        c.category.toLowerCase().includes(q)
    );
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
            placeholder="Search commands, pages, git actions..."
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
      </div>
    </>
  );
}
