import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  LayoutDashboard, Star, FolderOpen, RefreshCw, Trash2,
  ArrowUpWideNarrow, ArrowDownWideNarrow, GitBranch, AlertTriangle, X,
  ChevronDown, EyeOff,
} from "lucide-react";
import { useAppStore, type RepoOverview } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

function timeAgo(unix?: number): string {
  if (!unix) return "";
  const s = Math.floor(Date.now() / 1000 - unix);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const {
    dashboard, loading, repoPath, openRepo,
    fetchDashboard, toggleFavorite, fetchAllRepos, pruneDeadRepos, removeRepo,
  } = useAppStore();
  const [showDead, setShowDead] = useState(false);

  useEffect(() => {
    fetchDashboard();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const busy = !!loading.dashboard;
  const fetching = !!loading.dashboardFetch;

  // Defensive dedupe by path (favorites win); backend already dedupes.
  const unique = useMemo(() => {
    const seen = new Set<string>();
    return dashboard.filter((r) => {
      if (seen.has(r.path)) return false;
      seen.add(r.path);
      return true;
    });
  }, [dashboard]);

  const live = useMemo(() => unique.filter((r) => !r.error), [unique]);
  const dead = useMemo(() => unique.filter((r) => !!r.error), [unique]);
  const favorites = live.filter((r) => r.favorite);
  const rest = live.filter((r) => !r.favorite);

  const cardProps = (r: RepoOverview) => ({
    repo: r,
    current: r.path === repoPath,
    onOpen: () => openRepo(r.path),
    onFavorite: () => toggleFavorite(r.path),
    onRemove: () => removeRepo(r.path),
  });

  return (
    <div className="h-full flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Dashboard</h2>
          {live.length > 0 && (
            <Badge variant="outline" className="text-xs">{live.length}</Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => fetchDashboard()} disabled={busy}>
            <RefreshCw className={cn("w-4 h-4 mr-1.5", busy && "animate-spin")} />
            Refresh
          </Button>
          <Button size="sm" onClick={() => fetchAllRepos()} disabled={busy || fetching}>
            <RefreshCw className={cn("w-4 h-4 mr-1.5", fetching && "animate-spin")} />
            {fetching ? "Fetching..." : "Fetch all"}
          </Button>
        </div>
      </div>

      {busy && live.length === 0 && dead.length === 0 ? (
        <div className="flex items-center justify-center h-64 text-muted-foreground">
          Loading repositories...
        </div>
      ) : live.length === 0 && dead.length === 0 ? (
        <EmptyState
          icon={<LayoutDashboard className="w-16 h-16" />}
          title="No repositories yet"
          description="Open a repository to pin it here. Star favorites to keep them on top."
          action={{ label: "Open Repo", onClick: () => navigate({ to: "/" }) }}
          className="flex-1"
        />
      ) : (
        <ScrollArea className="flex-1 min-h-0">
          {favorites.length > 0 && (
            <div className="mb-5">
              <h3 className="text-sm font-medium text-muted-foreground mb-2 flex items-center gap-1.5">
                <Star className="w-3.5 h-3.5" /> Favorites
              </h3>
              <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                {favorites.map((r) => (
                  <RepoCard key={r.path} {...cardProps(r)} />
                ))}
              </div>
            </div>
          )}
          {rest.length > 0 && (
            <div className={cn(favorites.length > 0 && "mb-5")}>
              {favorites.length > 0 && (
                <h3 className="text-sm font-medium text-muted-foreground mb-2">Recent</h3>
              )}
              <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                {rest.map((r) => (
                  <RepoCard key={r.path} {...cardProps(r)} />
                ))}
              </div>
            </div>
          )}

          {/* Unreachable entries live here, out of the way */}
          {dead.length > 0 && (
            <div className="rounded-lg border border-border/50 overflow-hidden">
              <button
                onClick={() => setShowDead(!showDead)}
                className="flex items-center gap-2 w-full px-3 py-2 text-xs text-muted-foreground hover:bg-accent/30 transition-colors"
              >
                <EyeOff className="w-3.5 h-3.5" />
                <span>{dead.length} unreachable {dead.length === 1 ? "repo" : "repos"}</span>
                <ChevronDown className={cn("w-3.5 h-3.5 ml-auto transition-transform", showDead && "rotate-180")} />
              </button>
              {showDead && (
                <div className="border-t border-border/30 px-2 py-1.5 space-y-0.5">
                  {dead.map((r) => (
                    <div
                      key={r.path}
                      className="flex items-center gap-2 px-2 py-1.5 rounded text-xs"
                      title={r.error}
                    >
                      <AlertTriangle className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                      <span className="font-medium truncate" title={r.path}>{r.name}</span>
                      <span className="text-muted-foreground truncate font-mono text-[11px]">{r.path}</span>
                      <button
                        onClick={() => removeRepo(r.path)}
                        className="ml-auto p-1 rounded text-muted-foreground/60 hover:text-destructive shrink-0"
                        title="Remove from app (files on disk untouched)"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ))}
                  <Button
                    size="sm"
                    variant="ghost"
                    className="w-full h-7 text-xs text-muted-foreground"
                    onClick={() => pruneDeadRepos()}
                    disabled={busy}
                  >
                    <Trash2 className="w-3.5 h-3.5 mr-1.5" />
                    Prune all unreachable
                  </Button>
                </div>
              )}
            </div>
          )}
        </ScrollArea>
      )}
    </div>
  );
}

function RepoCard({ repo: r, current, onOpen, onFavorite, onRemove }: {
  repo: RepoOverview;
  current: boolean;
  onOpen: () => void;
  onFavorite: () => void;
  onRemove: () => void;
}) {
  const clickable = !current;
  return (
    <div
      onClick={clickable ? onOpen : undefined}
      className={cn(
        "rounded-lg border p-3 transition-colors",
        current
          ? "border-primary/40 bg-primary/5"
          : "border-border/50 hover:bg-accent/30 hover:border-border cursor-pointer",
      )}
    >
      <div className="flex items-center gap-2 min-w-0">
        <span className={cn(
          "w-2.5 h-2.5 rounded-full shrink-0",
          r.is_clean ? "bg-green-500" : "bg-yellow-500",
        )} title={r.is_clean ? "clean" : `${r.dirty_count} dirty files`} />
        <span className="font-medium text-sm truncate flex-1" title={r.path}>{r.name}</span>
        {current && (
          <Badge variant="secondary" className="text-[10px] h-4 shrink-0">open</Badge>
        )}
        <button
          onClick={(e) => { e.stopPropagation(); onFavorite(); }}
          className={cn(
            "p-1 rounded transition-colors shrink-0",
            r.favorite ? "text-yellow-500" : "text-muted-foreground/40 hover:text-yellow-500",
          )}
          title={r.favorite ? "Unpin" : "Pin to favorites"}
        >
          <Star className={cn("w-3.5 h-3.5", r.favorite && "fill-current")} />
        </button>
        {!current && (
          <button
            onClick={(e) => { e.stopPropagation(); onRemove(); }}
            className="p-1 rounded transition-colors shrink-0 text-muted-foreground/40 hover:text-destructive"
            title="Remove from app (files on disk untouched)"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      <div className="flex items-center gap-2 mt-1.5 text-xs text-muted-foreground">
        <span className="font-mono flex items-center gap-1 truncate">
          <GitBranch className="w-3 h-3 shrink-0" />
          {r.branch || "HEAD"}
        </span>
        {!r.is_clean && (
          <span className="text-yellow-500 shrink-0">{r.dirty_count} dirty</span>
        )}
        {(r.ahead > 0 || r.behind > 0) && (
          <span className="flex items-center gap-1 shrink-0">
            {r.ahead > 0 && (
              <span className="flex items-center gap-0.5 text-green-500">
                <ArrowUpWideNarrow className="w-3 h-3" />{r.ahead}
              </span>
            )}
            {r.behind > 0 && (
              <span className="flex items-center gap-0.5 text-red-500">
                <ArrowDownWideNarrow className="w-3 h-3" />{r.behind}
              </span>
            )}
          </span>
        )}
      </div>
      {r.last_message && (
        <p className="text-xs text-muted-foreground truncate mt-1" title={r.last_message}>
          {r.last_message}
          {r.last_time ? ` · ${timeAgo(r.last_time)}` : ""}
        </p>
      )}

      {!current && (
        <div className="mt-2">
          <Button size="sm" variant="outline" className="h-7 text-xs w-full" onClick={onOpen}>
            <FolderOpen className="w-3.5 h-3.5 mr-1.5" />
            Open
          </Button>
        </div>
      )}
    </div>
  );
}
