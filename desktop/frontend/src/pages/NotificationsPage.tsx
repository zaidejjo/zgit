import { useEffect, useMemo, useState } from "react";
import {
  Bell, BellOff, RefreshCw, Search, Check, CheckCheck, ExternalLink, Inbox,
  GitPullRequest, CircleDot, Play, GitCommitHorizontal, Tag, MessagesSquare,
} from "lucide-react";
import { useAppStore, type NotificationItem } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

function timeAgo(iso?: string): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (!t) return "";
  const s = Math.floor((Date.now() - t) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 30) return `${Math.floor(s / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
}

function prettyReason(reason?: string): string {
  if (!reason) return "";
  return reason.replace(/_/g, " ");
}

function TypeIcon({ type, className }: { type?: string; className?: string }) {
  switch (type) {
    case "PullRequest":
      return <GitPullRequest className={cn("text-violet-500", className)} />;
    case "Issue":
      return <CircleDot className={cn("text-green-500", className)} />;
    case "CheckSuite":
      return <Play className={cn("text-yellow-500", className)} />;
    case "Commit":
      return <GitCommitHorizontal className={cn("text-blue-500", className)} />;
    case "Release":
      return <Tag className={cn("text-orange-500", className)} />;
    case "Discussion":
      return <MessagesSquare className={cn("text-cyan-500", className)} />;
    default:
      return <Bell className={cn("text-muted-foreground", className)} />;
  }
}

export default function NotificationsPage() {
  const {
    notifications, loading, ghAuthenticated, setLoginDialogOpen,
    fetchNotifications, markNotificationRead, markAllNotificationsRead,
  } = useAppStore();

  const [showAll, setShowAll] = useState(false);
  const [query, setQuery] = useState("");
  const [reason, setReason] = useState<string | null>(null);

  useEffect(() => {
    if (ghAuthenticated) fetchNotifications(showAll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ghAuthenticated, showAll]);

  const busy = !!loading.notifications;
  const unread = useMemo(() => notifications.filter((n) => n.unread).length, [notifications]);

  const reasons = useMemo(() => {
    const counts = new Map<string, number>();
    for (const n of notifications) {
      if (n.reason) counts.set(n.reason, (counts.get(n.reason) || 0) + 1);
    }
    return [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 6);
  }, [notifications]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return notifications.filter((n) => {
      if (reason && n.reason !== reason) return false;
      if (!q) return true;
      return `${n.title} ${n.repo_name} ${n.reason} ${n.type}`.toLowerCase().includes(q);
    });
  }, [notifications, query, reason]);

  const openExternal = (n: NotificationItem) => {
    const url = n.html_url || n.repo_url;
    if (url) window.open(url, "_blank");
    if (n.unread) markNotificationRead(n.id);
  };

  if (!ghAuthenticated) {
    return (
      <div className="h-full flex items-center justify-center p-6">
        <EmptyState
          icon={<Bell className="w-16 h-16" />}
          title="Your GitHub inbox"
          description="Reviews requested, mentions, assignments and failing CI — triage without opening a browser."
          action={{ label: "Sign in with GitHub", onClick: () => setLoginDialogOpen(true) }}
          className="max-w-md w-full"
        />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col gap-3 min-h-0">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2 mr-auto">
          <h2 className="text-xl font-bold">Notifications</h2>
          {unread > 0 && (
            <Badge className="text-[11px] tabular-nums bg-primary text-primary-foreground hover:bg-primary">
              {unread} unread
            </Badge>
          )}
        </div>
        <div className="relative w-full sm:w-52">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter notifications..."
            className="h-8 pl-8 text-xs"
          />
        </div>
        <Button
          size="sm" variant="outline" className="h-8"
          onClick={() => markAllNotificationsRead()}
          disabled={busy || unread === 0}
          title="Mark everything read"
        >
          <CheckCheck className="w-3.5 h-3.5 mr-1.5" />
          Mark all read
        </Button>
        <Button size="sm" variant="outline" className="h-8" onClick={() => fetchNotifications(showAll)} disabled={busy}>
          <RefreshCw className={cn("w-3.5 h-3.5 mr-1.5", busy && "animate-spin")} />
          Refresh
        </Button>
      </div>

      <div className="flex items-center gap-1 flex-wrap">
        <button
          onClick={() => setShowAll(false)}
          className={cn(
            "px-2.5 py-1 rounded-full text-xs font-medium transition-colors",
            !showAll ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
          )}
        >
          Unread{unread > 0 ? ` (${unread})` : ""}
        </button>
        <button
          onClick={() => setShowAll(true)}
          className={cn(
            "px-2.5 py-1 rounded-full text-xs font-medium transition-colors",
            showAll ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
          )}
        >
          All
        </button>
        {reasons.length > 0 && <span className="w-px h-4 bg-border mx-1" />}
        {reasons.map(([r, count]) => (
          <button
            key={r}
            onClick={() => setReason((prev) => (prev === r ? null : r))}
            className={cn(
              "px-2 py-1 rounded-full text-[11px] font-mono transition-colors border",
              reason === r
                ? "border-primary/50 bg-primary/10 text-primary"
                : "border-border/50 text-muted-foreground hover:text-foreground hover:bg-accent/40",
            )}
          >
            {prettyReason(r)} · {count}
          </button>
        ))}
      </div>

      <div className="flex-1 min-h-0 rounded-xl border border-border/50 bg-card/30 overflow-hidden flex flex-col">
        <ScrollArea className="flex-1 min-h-0">
          {busy && notifications.length === 0 ? (
            <div className="p-2 space-y-1.5">
              {[0, 1, 2, 3].map((i) => (
                <div key={i} className="rounded-lg border border-border/40 p-3 animate-pulse">
                  <div className="h-3.5 w-2/3 rounded bg-muted" />
                  <div className="h-3 w-1/3 rounded bg-muted mt-2" />
                </div>
              ))}
            </div>
          ) : visible.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-14 px-6 text-center">
              {notifications.length === 0 ? (
                <>
                  <Inbox className="w-8 h-8 text-muted-foreground/40 mb-2" />
                  <p className="text-sm font-medium">Inbox zero</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {showAll ? "No notifications at all. Enjoy the quiet." : "Nothing unread. Switch to All to browse history."}
                  </p>
                </>
              ) : (
                <>
                  <BellOff className="w-8 h-8 text-muted-foreground/40 mb-2" />
                  <p className="text-sm font-medium">No matches</p>
                  <p className="text-xs text-muted-foreground mt-1">Try a different filter or search.</p>
                </>
              )}
            </div>
          ) : (
            <div className="p-2 space-y-1">
              {visible.map((n) => (
                <div
                  key={n.id}
                  onClick={() => openExternal(n)}
                  className={cn(
                    "group flex items-start gap-2.5 rounded-lg border px-3 py-2.5 cursor-pointer transition-colors",
                    n.unread
                      ? "border-primary/25 bg-primary/[0.04] hover:bg-primary/[0.08]"
                      : "border-transparent hover:border-border/60 hover:bg-accent/40",
                  )}
                >
                  <span className={cn("w-1.5 h-1.5 rounded-full mt-1.5 shrink-0", n.unread ? "bg-primary" : "bg-transparent")} />
                  <TypeIcon type={n.type} className="w-4 h-4 mt-0.5 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className={cn("text-sm truncate leading-tight", n.unread ? "font-semibold" : "font-medium text-muted-foreground")}>
                      {n.title || "(no title)"}
                    </p>
                    <p className="text-[11px] text-muted-foreground mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5">
                      <span className="font-medium truncate max-w-[220px]">{n.repo_name}</span>
                      {n.reason && (
                        <span className="font-mono px-1 py-px rounded bg-muted/60">{prettyReason(n.reason)}</span>
                      )}
                      {n.updated_at && <span>{timeAgo(n.updated_at)}</span>}
                    </p>
                  </div>
                  <div className="flex gap-0.5 shrink-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    {n.unread && (
                      <button
                        onClick={(e) => { e.stopPropagation(); markNotificationRead(n.id); }}
                        className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent"
                        title="Mark read"
                      >
                        <Check className="w-3.5 h-3.5" />
                      </button>
                    )}
                    <button
                      onClick={(e) => { e.stopPropagation(); openExternal(n); }}
                      className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent"
                      title="Open in browser"
                    >
                      <ExternalLink className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </ScrollArea>
      </div>
    </div>
  );
}
