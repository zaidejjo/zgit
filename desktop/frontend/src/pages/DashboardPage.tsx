import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  LayoutDashboard, Star, FolderOpen, RefreshCw, Trash2,
  ArrowUpWideNarrow, ArrowDownWideNarrow, GitBranch, AlertTriangle,
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
    fetchDashboard, toggleFavorite, fetchAllRepos, pruneDeadRepos,
  } = useAppStore();

  useEffect(() => {
    fetchDashboard();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const busy = !!loading.dashboard;
  const fetching = !!loading.dashboardFetch;
  const favorites = dashboard.filter((r) => r.favorite);
  const rest = dashboard.filter((r) => !r.favorite);

  return (
    <div className="h-full flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Dashboard</h2>
          {dashboard.length > 0 && (
            <Badge variant="outline" className="text-xs">{dashboard.length}</Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => pruneDeadRepos()} disabled={busy}>
            <Trash2 className="w-4 h-4 mr-1.5" />
            Prune dead
          </Button>
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

      {busy && dashboard.length === 0 ? (
        <div className="flex items-center justify-center h-64 text-muted-foreground">
          Loading repositories...
        </div>
      ) : dashboard.length === 0 ? (
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
                  <RepoCard key={r.path} repo={r} current={r.path === repoPath}
                    onOpen={() => openRepo(r.path)} onFavorite={() => toggleFavorite(r.path)} />
                ))}
              </div>
            </div>
          )}
          <div>
            {favorites.length > 0 && (
              <h3 className="text-sm font-medium text-muted-foreground mb-2">Recent</h3>
            )}
            <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {rest.map((r) => (
                <RepoCard key={r.path} repo={r} current={r.path === repoPath}
                  onOpen={() => openRepo(r.path)} onFavorite={() => toggleFavorite(r.path)} />
              ))}
            </div>
          </div>
        </ScrollArea>
      )}
    </div>
  );
}

function RepoCard({ repo: r, current, onOpen, onFavorite }: {
  repo: RepoOverview;
  current: boolean;
  onOpen: () => void;
  onFavorite: () => void;
}) {
  return (
    <div className={cn(
      "rounded-lg border p-3 transition-colors",
      current ? "border-primary/40 bg-primary/5" : "border-border/50 hover:bg-accent/30",
    )}>
      <div className="flex items-center gap-2 min-w-0">
        <span className={cn(
          "w-2.5 h-2.5 rounded-full shrink-0",
          r.error ? "bg-gray-500" : r.is_clean ? "bg-green-500" : "bg-yellow-500",
        )} title={r.error ? "unreadable" : r.is_clean ? "clean" : `${r.dirty_count} dirty files`} />
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
      </div>

      {r.error ? (
        <p className="text-xs text-muted-foreground mt-1.5 flex items-center gap-1.5">
          <AlertTriangle className="w-3 h-3 shrink-0" />
          <span className="truncate">{r.error}</span>
        </p>
      ) : (
        <>
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
        </>
      )}

      <div className="mt-2">
        {!current && !r.error && (
          <Button size="sm" variant="outline" className="h-7 text-xs w-full" onClick={onOpen}>
            <FolderOpen className="w-3.5 h-3.5 mr-1.5" />
            Open
          </Button>
        )}
      </div>
    </div>
  );
}
