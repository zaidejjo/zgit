import { useEffect, useRef, useState, useCallback } from "react";
import { Outlet, useNavigate, useLocation } from "@tanstack/react-router";
import {
  GitCommitHorizontal, History, GitBranch, GitPullRequest,
  CircleDot, Play, Globe, Tag, Settings, FolderOpen, Folder,
  X, Sparkles, Command, Search, ChevronDown, GitFork, Palette,
} from "lucide-react";
import { useAppStore, type Theme, THEMES } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import LoginDialog from "@/components/LoginDialog";
import PushConfirmDialog from "@/components/PushConfirmDialog";
import UndoBanner from "@/components/UndoBanner";
import AIAssistantPanel from "@/components/AIAssistantPanel";
import { EventsOn } from "../wailsjs/runtime";
import CommandPalette from "@/components/CommandPalette";

const navItems = [
  { id: "status",   label: "Status",   Icon: GitCommitHorizontal },
  { id: "log",      label: "Log",      Icon: History },
  { id: "branches", label: "Branches", Icon: GitBranch },
  { id: "pull-requests", label: "PRs", Icon: GitPullRequest },
  { id: "issues",   label: "Issues",   Icon: CircleDot },
  { id: "actions",  label: "Actions",  Icon: Play },
] as const;

const bottomNavItems = [
  { id: "remotes",  label: "Remotes",  Icon: Globe },
  { id: "tags",     label: "Tags",     Icon: Tag },
  { id: "settings", label: "Settings", Icon: Settings },
] as const;

export default function AppLayout() {
  const {
    activeTab, setActiveTab, error, setError,
    fetchStatus, checkGitHubAuth, ghAuthenticated, ghUser, setLoginDialogOpen,
    repoPath, recentRepos, openRepo, selectAndOpenRepo, fetchRecentRepos, refreshAll,
    toggleAIPanel, aiPanelOpen, currentBranch, status,
    theme, setTheme,
  } = useAppStore();
  const navigate = useNavigate();
  const location = useLocation();

  const [repoMenuOpen, setRepoMenuOpen] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [themeMenuOpen, setThemeMenuOpen] = useState(false);
  const repoMenuRef = useRef<HTMLDivElement>(null);
  const themeMenuRef = useRef<HTMLDivElement>(null);

  // Initialize
  useEffect(() => {
    fetchStatus();
    checkGitHubAuth();
    fetchRecentRepos();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // File system watcher
  useEffect(() => {
    const unwatch = EventsOn("fs:status-changed", () => {
      useAppStore.getState().fetchStatus();
    });
    return () => unwatch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Repo switch event
  useEffect(() => {
    const unwatch = EventsOn("repo:switched", () => {
      useAppStore.getState().refreshAll();
      useAppStore.getState().fetchRecentRepos();
    });
    return () => unwatch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Cmd+K / Ctrl+K — Command Palette
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setCommandPaletteOpen(true);
      }
      if (e.ctrlKey && e.shiftKey && e.key === "A") {
        e.preventDefault();
        if (location.pathname === "/ai") {
          navigate({ to: "/" });
          return;
        }
        toggleAIPanel();
      }
      if (e.ctrlKey && e.shiftKey && e.key === "F") {
        e.preventDefault();
        navigate({ to: "/ai" });
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [toggleAIPanel, navigate, location.pathname]);

  // Close repo dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (repoMenuRef.current && !repoMenuRef.current.contains(e.target as Node)) {
        setRepoMenuOpen(false);
      }
    };
    if (repoMenuOpen) {
      document.addEventListener("mousedown", handler);
      return () => document.removeEventListener("mousedown", handler);
    }
  }, [repoMenuOpen]);

  // Close theme menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (themeMenuRef.current && !themeMenuRef.current.contains(e.target as Node)) {
        setThemeMenuOpen(false);
      }
    };
    if (themeMenuOpen) {
      document.addEventListener("mousedown", handler);
      return () => document.removeEventListener("mousedown", handler);
    }
  }, [themeMenuOpen]);

  // Auto-dismiss error
  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error, setError]);

  // Page title lookup
  const pageTitleMap: Record<string, string> = {
    status: "Status", log: "Log", branches: "Branches",
    "pull-requests": "PRs", issues: "Issues", actions: "Actions",
    remotes: "Remotes", tags: "Tags", settings: "Settings",
  };

  // Derive repo name for display
  const repoName = repoPath ? repoPath.split("/").pop() || repoPath : "";

  const handleNav = useCallback((tabId: string) => {
    setActiveTab(tabId);
    navigate({ to: `/${tabId === "status" ? "" : tabId}` });
  }, [setActiveTab, navigate]);

  const getNavId = (): string => {
    if (location.pathname === "/") return "status";
    const p = location.pathname.slice(1);
    return navItems.find((n) => n.id === p)?.id
      || bottomNavItems.find((n) => n.id === p)?.id
      || "status";
  };

  const currentNavId = getNavId();
  const isBottomNav = bottomNavItems.some((n) => n.id === currentNavId);

  // Dirty file count for sidebar badge
  const dirtyCount = !status?.is_clean
    ? (status?.files || []).filter((f) =>
        f.staged !== 7 || f.unstaged !== 7
      ).length
    : 0;

  return (
    <div className="h-screen flex overflow-hidden bg-background">
      {/* ─── Sidebar ─── */}
      <aside className="w-56 shrink-0 flex flex-col border-r border-border/50 bg-card/50">
        {/* Logo + Search */}
        <div className="px-4 py-3 flex items-center justify-between border-b border-border/30">
          <div className="flex items-center gap-2">
            <img src="/images/logo.svg" alt="zgit" className="w-5 h-5" />
            <span className="text-sm font-semibold tracking-tight">zgit</span>
          </div>
          <button
            onClick={() => setCommandPaletteOpen(true)}
            className="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
            title="Command Palette (Ctrl+K)"
          >
            <Search className="w-3.5 h-3.5" />
          </button>
        </div>

        {/* Repo switcher */}
        <div className="px-3 py-2" ref={repoMenuRef}>
          <button
            onClick={() => setRepoMenuOpen(!repoMenuOpen)}
            className={cn(
              "sidebar-item w-full",
              repoMenuOpen && "bg-accent/50"
            )}
            title={repoPath || "No repository"}
          >
            <FolderOpen className="w-4 h-4 shrink-0" />
            <span className="flex-1 truncate text-left text-xs font-medium">
              {repoName || "Open Repo"}
            </span>
            <ChevronDown className={cn(
              "w-3 h-3 transition-transform duration-150",
              repoMenuOpen && "rotate-180"
            )} />
          </button>

          {repoMenuOpen && (
            <div className="mt-1 rounded-lg border border-border/50 bg-popover shadow-xl py-1">
              <button
                className="flex items-center gap-2.5 w-full px-3 py-2 text-sm hover:bg-accent/50 transition-colors"
                onClick={() => { setRepoMenuOpen(false); selectAndOpenRepo(); }}
              >
                <FolderOpen className="w-4 h-4 text-muted-foreground" />
                <span>Open Folder...</span>
              </button>

              {recentRepos.length > 0 && (
                <>
                  <div className="h-px bg-border/50 mx-3 my-1" />
                  <div className="px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground font-medium">
                    Recent
                  </div>
                  {recentRepos.slice(0, 8).map((r) => {
                    const name = r.split("/").pop() || r;
                    return (
                      <button
                        key={r}
                        className={cn(
                          "flex items-center gap-2.5 w-full px-3 py-1.5 text-sm hover:bg-accent/50 transition-colors",
                          r === repoPath && "bg-accent/30 text-primary font-medium"
                        )}
                        onClick={() => { setRepoMenuOpen(false); openRepo(r); }}
                        title={r}
                      >
                        <Folder className="w-4 h-4 text-muted-foreground shrink-0" />
                        <div className="min-w-0 text-left">
                          <div className="truncate text-xs">{name}</div>
                          <div className="text-[10px] text-muted-foreground truncate">{r}</div>
                        </div>
                      </button>
                    );
                  })}
                </>
              )}
            </div>
          )}
        </div>

        {/* Branch indicator */}
        {status?.branch && (
          <div className="px-3 pb-2">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-primary/5 border border-primary/10 text-xs">
              <GitFork className="w-3 h-3 text-primary" />
              <span className="font-mono text-primary font-medium truncate">{status.branch}</span>
            </div>
          </div>
        )}

        {/* Main nav */}
        <nav className="flex-1 px-3 space-y-0.5 overflow-y-auto">
          {navItems.map((item) => {
            const NavIcon = item.Icon;
            const isActive = currentNavId === item.id && !isBottomNav;
            const isStatusWithDirty = item.id === "status" && dirtyCount > 0;
            return (
              <button
                key={item.id}
                onClick={() => handleNav(item.id)}
                className={cn(
                  "sidebar-item group",
                  isActive && "active"
                )}
              >
                <NavIcon className="w-4 h-4 shrink-0" />
                <span className="flex-1 text-left">{item.label}</span>
                {isStatusWithDirty && (
                  <span className="w-5 h-5 flex items-center justify-center rounded-full bg-primary/15 text-[10px] font-medium text-primary">
                    {dirtyCount > 9 ? "9+" : dirtyCount}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Bottom nav */}
        <div className="px-3 py-2 border-t border-border/30 space-y-0.5">
          {bottomNavItems.map((item) => {
            const NavIcon = item.Icon;
            const isActive = currentNavId === item.id && isBottomNav;
            return (
              <button
                key={item.id}
                onClick={() => handleNav(item.id)}
                className={cn("sidebar-item", isActive && "active")}
              >
                <NavIcon className="w-4 h-4 shrink-0" />
                <span className="flex-1 text-left">{item.label}</span>
              </button>
            );
          })}
        </div>
      </aside>

      {/* ─── Main Area ─── */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Glass header */}
        <header className="shrink-0 glass glass-hover rounded-none border-x-0 border-t-0 z-10">
          <div className="flex items-center justify-between h-11 px-5">
            {/* Page title */}
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-foreground">
                {pageTitleMap[currentNavId] || "Status"}
              </span>
              {!status?.is_clean && dirtyCount > 0 && (
                <span className="text-[11px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                  {dirtyCount} change{dirtyCount !== 1 ? "s" : ""}
                </span>
              )}
            </div>

            <div className="flex items-center gap-2">
              {/* GitHub profile */}
              {ghAuthenticated && ghUser ? (
                <div className="flex items-center gap-2 text-xs">
                  <img
                    src={ghUser.avatar_url}
                    alt={ghUser.login}
                    className="w-5 h-5 rounded-full ring-1 ring-border"
                  />
                  <span className="text-muted-foreground hidden sm:inline">
                    {ghUser.login}
                  </span>
                </div>
              ) : (
                <button
                  className="text-xs px-2.5 py-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale"
                  onClick={() => setLoginDialogOpen(true)}
                >
                  Sign In
                </button>
              )}

              {/* AI Assistant trigger */}
              <button
                onClick={toggleAIPanel}
                className={cn(
                  "p-1.5 rounded-md transition-all duration-150 press-scale",
                  aiPanelOpen
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
                )}
                title="AI Assistant (Ctrl+Shift+A)"
              >
                <Sparkles className="w-4 h-4" />
              </button>

              {/* Theme selector */}
              <div className="relative" ref={themeMenuRef}>
                <button
                  onClick={() => setThemeMenuOpen(!themeMenuOpen)}
                  className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all duration-150 press-scale"
                  title="Change theme"
                >
                  <Palette className="w-4 h-4" />
                </button>

                {themeMenuOpen && (
                  <div className="absolute right-0 top-full mt-1 z-50 w-48 rounded-xl glass shadow-2xl py-1.5">
                    {THEMES.map((t) => (
                      <button
                        key={t.id}
                        onClick={() => { setTheme(t.id); setThemeMenuOpen(false); }}
                        className={cn(
                          "flex items-center gap-3 w-full px-3 py-2 text-xs transition-colors hover:bg-accent/30",
                          theme === t.id ? "text-primary font-medium" : "text-muted-foreground"
                        )}
                      >
                        <span className="text-base">{t.icon}</span>
                        <span>{t.label}</span>
                        {theme === t.id && (
                          <span className="w-1.5 h-1.5 rounded-full bg-primary ml-auto" />
                        )}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Command palette hint */}
              <div className="hidden sm:flex items-center gap-1 px-2 py-1 rounded-md bg-muted/30 text-[10px] text-muted-foreground font-mono">
                <Command className="w-2.5 h-2.5" />
                <span>K</span>
              </div>
            </div>
          </div>
        </header>

        {/* Error toast */}
        {error && (
          <div className="fixed top-4 right-4 z-50 max-w-md glass text-foreground px-4 py-3 rounded-xl shadow-2xl text-sm flex items-center gap-2 animate-in slide-in-from-right-2">
            <span className="flex-1">{error}</span>
            <button
              className="text-muted-foreground hover:text-foreground shrink-0 press-scale"
              onClick={() => setError(null)}
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Undo banner */}
        <UndoBanner />

        {/* Main content */}
        <main className="flex-1 overflow-auto">
          <div className="p-5">
            <Outlet />
          </div>
        </main>
      </div>

      {/* Dialogs */}
      <LoginDialog />
      <PushConfirmDialog />
      {location.pathname !== "/ai" && <AIAssistantPanel />}

      {/* Command Palette */}
      <CommandPalette
        open={commandPaletteOpen}
        onClose={() => setCommandPaletteOpen(false)}
        onNavigate={handleNav}
      />
    </div>
  );
}
