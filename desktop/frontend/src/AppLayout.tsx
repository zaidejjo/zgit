import { useEffect, useRef, useState } from "react";
import { Outlet, useNavigate, useLocation } from "@tanstack/react-router";
import {
  GitCommitHorizontal, History, GitBranch, GitPullRequest,
  CircleDot, Play, Globe, Settings, FolderOpen, Folder, Sun, Moon, X,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import LoginDialog from "@/components/LoginDialog";
import PushConfirmDialog from "@/components/PushConfirmDialog";
import { EventsOn } from "../wailsjs/runtime";

const tabs = [
  { id: "status",   label: "Status",   Icon: GitCommitHorizontal },
  { id: "log",      label: "Log",      Icon: History },
  { id: "branches", label: "Branches", Icon: GitBranch },
  { id: "pull-requests", label: "PRs", Icon: GitPullRequest },
  { id: "issues",   label: "Issues",   Icon: CircleDot },
  { id: "actions",  label: "Actions",  Icon: Play },
  { id: "remotes",  label: "Remotes",  Icon: Globe },
  { id: "settings", label: "Settings", Icon: Settings },
] as const;

export default function AppLayout() {
  const {
    activeTab, setActiveTab, error, setError, darkMode, toggleDarkMode,
    fetchStatus, checkGitHubAuth, ghAuthenticated, ghUser, setLoginDialogOpen,
    repoPath, recentRepos, openRepo, selectAndOpenRepo, fetchRecentRepos, refreshAll,
  } = useAppStore();
  const navigate = useNavigate();
  const location = useLocation();

  const [repoMenuOpen, setRepoMenuOpen] = useState(false);
  const repoMenuRef = useRef<HTMLDivElement>(null);

  // Initialize
  useEffect(() => {
    document.documentElement.classList.toggle("dark", darkMode);
  }, [darkMode]);

  useEffect(() => {
    fetchStatus();
    checkGitHubAuth();
    fetchRecentRepos();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // File system watcher — auto-refresh status on file changes
  const unwatcherRef = useRef<(() => void) | null>(null);
  useEffect(() => {
    const unwatch = EventsOn("fs:status-changed", () => {
      get().fetchStatus();
    });
    unwatcherRef.current = unwatch;
    return () => {
      if (unwatcherRef.current) unwatcherRef.current();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Repo switch event — refresh all data when backend opens a new repo
  useEffect(() => {
    const unwatch = EventsOn("repo:switched", () => {
      get().refreshAll();
      get().fetchRecentRepos();
    });
    return () => unwatch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  // Helper to access store inside event callback
  const get = useAppStore.getState;

  // Derive repo name for display
  const repoName = repoPath ? repoPath.split("/").pop() || repoPath : "";

  const handleTabChange = (tabId: string) => {
    setActiveTab(tabId);
    navigate({ to: `/${tabId === "status" ? "" : tabId}` });
  };

  // Auto-dismiss error after 5s
  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error, setError]);

  const currentTab = tabs.find((t) =>
    location.pathname === "/" ? t.id === "status" : location.pathname === `/${t.id}`
  )?.id || "status";

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="flex items-center justify-between px-4 h-12">
          <div className="flex items-center gap-1">
            {/* Repository switcher — top-left before tabs */}
            <div className="relative mr-2" ref={repoMenuRef}>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs font-mono max-w-[160px] truncate"
                onClick={() => setRepoMenuOpen(!repoMenuOpen)}
                title={repoPath || "No repository"}
              >
                <FolderOpen className="w-3.5 h-3.5 mr-1.5 shrink-0" />
                {repoName || "Open Repo"}
              </Button>

              {repoMenuOpen && (
                <div className="absolute left-0 top-full mt-1 z-50 w-72 rounded-lg border bg-popover shadow-lg">
                  <div className="p-1">
                    {/* Open folder button */}
                    <button
                      className="flex items-center gap-2 w-full px-3 py-2 text-sm rounded-md hover:bg-accent transition-colors"
                      onClick={() => { setRepoMenuOpen(false); selectAndOpenRepo(); }}
                    >
                      <FolderOpen className="w-4 h-4 text-muted-foreground" />
                      <span>Open Folder...</span>
                    </button>

                    {/* Recent repos */}
                    {recentRepos.length > 0 && (
                      <>
                        <div className="h-px bg-border mx-2 my-1" />
                        <div className="px-3 py-1 text-xs text-muted-foreground font-medium">
                          Recent
                        </div>
                        {recentRepos.slice(0, 10).map((r) => {
                          const name = r.split("/").pop() || r;
                          return (
                            <button
                              key={r}
                              className={cn(
                                "flex items-center gap-2 w-full px-3 py-1.5 text-sm rounded-md hover:bg-accent transition-colors",
                                r === repoPath && "bg-accent/50 text-primary font-medium"
                              )}
                              onClick={() => { setRepoMenuOpen(false); openRepo(r); }}
                              title={r}
                            >
                              <Folder className="w-4 h-4 text-muted-foreground shrink-0" />
                              <div className="min-w-0 text-left">
                                <div className="truncate">{name}</div>
                                <div className="text-xs text-muted-foreground truncate">{r}</div>
                              </div>
                            </button>
                          );
                        })}
                      </>
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* Nav tabs */}
            {tabs.map((tab) => {
              const TabIcon = tab.Icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => handleTabChange(tab.id)}
                  className={cn(
                    "flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors",
                    currentTab === tab.id
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:text-foreground hover:bg-accent"
                  )}
                >
                  <TabIcon className="w-4 h-4" />
                  {tab.label}
                </button>
              );
            })}
          </div>

          <div className="flex items-center gap-2">
            {/* GitHub profile */}
            {ghAuthenticated && ghUser ? (
              <div className="flex items-center gap-2 text-sm">
                <img
                  src={ghUser.avatar_url}
                  alt={ghUser.login}
                  className="w-6 h-6 rounded-full"
                />
                <span className="text-muted-foreground hidden sm:inline">
                  {ghUser.login}
                </span>
              </div>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={() => setLoginDialogOpen(true)}
              >
                Sign In
              </Button>
            )}

            <div className="w-px h-5 bg-border" />

            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2"
              onClick={toggleDarkMode}
            >
              {darkMode ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
            </Button>
          </div>
        </div>
      </header>

      {/* Error toast */}
      {error && (
        <div className="fixed top-4 right-4 z-50 max-w-md bg-destructive text-destructive-foreground px-4 py-3 rounded-lg shadow-lg text-sm flex items-center gap-2">
          <span className="flex-1">{error}</span>
          <button
            className="text-destructive-foreground/70 hover:text-destructive-foreground shrink-0"
            onClick={() => setError(null)}
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {/* Main content */}
      <main className="p-4">
        <Outlet />
      </main>

      {/* Login dialog */}
      <LoginDialog />
      {/* Push confirmation dialog */}
      <PushConfirmDialog />
    </div>
  );
}
