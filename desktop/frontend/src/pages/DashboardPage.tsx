import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  LayoutDashboard, Star, FolderOpen, FolderSearch, RefreshCw, Trash2,
  ArrowUpWideNarrow, ArrowDownWideNarrow, GitBranch, AlertTriangle, X,
  ChevronDown, EyeOff, History,
} from "lucide-react";
import { useAppStore, type RepoOverview, type ScannedRepo } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

/** 1234 -> "1.2k" */
function fmtCount(n?: number): string {
  if (!n) return "0";
  if (n < 1000) return String(n);
  const v = n / 1000;
  return `${v >= 100 ? Math.round(v) : v.toFixed(1).replace(/\.0$/, "")}k`;
}

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
    pickDirectory, scanRepos, trackRepos,
  } = useAppStore();
  const [showDead, setShowDead] = useState(false);
  const [scanOpen, setScanOpen] = useState(false);
  const [scanRoot, setScanRoot] = useState("");
  const [scanResults, setScanResults] = useState<ScannedRepo[]>([]);
  const [scanPicked, setScanPicked] = useState<Set<string>>(new Set());

  const scanning = !!loading.repoScan;

  const handleScanFolder = async () => {
    const dir = await pickDirectory();
    if (!dir) return;
    setScanRoot(dir);
    setScanResults([]);
    setScanPicked(new Set());
    setScanOpen(true);
    const found = await scanRepos(dir);
    setScanResults(found);
    setScanPicked(new Set(found.filter((r) => !r.tracked).map((r) => r.path)));
  };

  const toggleScanPick = (path: string) => {
    setScanPicked((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const handleAddScanned = async () => {
    if (scanPicked.size === 0) return;
    await trackRepos([...scanPicked]);
    setScanOpen(false);
  };

  useEffect(() => {
    fetchDashboard();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const busy = !!loading.dashboard;
  const refreshing = !!loading.dashboardRefreshing;
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
    <div className="h-full flex flex-col gap-3 min-h-0">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2 mr-auto">
          <h2 className="text-xl font-bold">Dashboard</h2>
          {live.length > 0 && (
            <Badge variant="outline" className="text-xs tabular-nums">{live.length}</Badge>
          )}
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <Button size="sm" variant="outline" className="h-8" onClick={handleScanFolder} disabled={busy || refreshing || scanning} title="Find git repositories in a folder and add them">
            <FolderSearch className={cn("w-3.5 h-3.5 mr-1.5", scanning && "animate-spin")} />
            {scanning ? "Scanning..." : "Scan folder"}
          </Button>
          <Button size="sm" variant="outline" className="h-8" onClick={() => fetchDashboard(true)} disabled={busy || refreshing}>
            <RefreshCw className={cn("w-3.5 h-3.5 mr-1.5", (busy || refreshing) && "animate-spin")} />
            {refreshing ? "Refreshing..." : "Refresh"}
          </Button>
          <Button size="sm" className="h-8" onClick={() => fetchAllRepos()} disabled={busy || refreshing || fetching}>
            <RefreshCw className={cn("w-3.5 h-3.5 mr-1.5", fetching && "animate-spin")} />
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
              <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
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
              <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
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

      {/* Scan dialog */}
      <Dialog open={scanOpen} onOpenChange={setScanOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FolderSearch className="w-5 h-5" />
              Add repositories
            </DialogTitle>
            <DialogDescription className="font-mono text-[11px] truncate" title={scanRoot}>
              {scanRoot}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            {scanning && scanResults.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-6">Scanning for git repositories...</p>
            ) : scanResults.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-6">
                No git repositories found here. Try a parent folder.
              </p>
            ) : (
              <>
                <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
                  <Checkbox
                    checked={scanResults.filter((r) => !r.tracked).every((r) => scanPicked.has(r.path))}
                    onChange={() => {
                      const untracked = scanResults.filter((r) => !r.tracked);
                      if (untracked.every((r) => scanPicked.has(r.path))) setScanPicked(new Set());
                      else setScanPicked(new Set(untracked.map((r) => r.path)));
                    }}
                  />
                  Select all new ({scanResults.filter((r) => !r.tracked).length})
                </label>
                <ScrollArea className="max-h-64">
                  <div className="space-y-0.5 pr-2">
                    {scanResults.map((r) => (
                      <label
                        key={r.path}
                        className={cn(
                          "flex items-start gap-2.5 px-2.5 py-2 rounded-lg transition-colors",
                          r.tracked ? "opacity-50" : "hover:bg-accent/40 cursor-pointer",
                        )}
                        title={r.path}
                      >
                        <Checkbox
                          checked={r.tracked || scanPicked.has(r.path)}
                          disabled={r.tracked}
                          onChange={() => toggleScanPick(r.path)}
                        />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">
                            {r.name}
                            {r.tracked && (
                              <span className="ml-1.5 text-[10px] text-muted-foreground font-normal">already added</span>
                            )}
                          </p>
                          <p className="text-[11px] text-muted-foreground truncate font-mono">{r.path}</p>
                        </div>
                      </label>
                    ))}
                  </div>
                </ScrollArea>
              </>
            )}
            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">Close</Button>
              </DialogClose>
              <Button onClick={handleAddScanned} disabled={scanPicked.size === 0}>
                <FolderOpen className="w-4 h-4 mr-1.5" />
                Add {scanPicked.size > 0 ? `(${scanPicked.size})` : ""}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
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

      <div className="flex items-center gap-2 mt-1.5 text-xs text-muted-foreground flex-wrap">
        <span className="font-mono flex items-center gap-1 truncate">
          <GitBranch className="w-3 h-3 shrink-0" />
          {r.branch || "HEAD"}
        </span>
        {typeof r.commit_count === "number" && r.commit_count > 0 && (
          <span className="flex items-center gap-1 shrink-0 tabular-nums" title={`${r.commit_count} commits`}>
            <History className="w-3 h-3" />{fmtCount(r.commit_count)}
          </span>
        )}
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
