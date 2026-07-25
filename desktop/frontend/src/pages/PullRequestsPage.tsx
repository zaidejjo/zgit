import { useEffect } from "react";
import { useAppStore } from "@/store/app";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { formatTimeAgo, truncate } from "@/lib/utils";
import { cn } from "@/lib/utils";

export default function PullRequestsPage() {
  const { pullRequests, loading, error, fetchPullRequests } = useAppStore();

  useEffect(() => {
    fetchPullRequests();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading.prs && pullRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading pull requests...
      </div>
    );
  }

  if (error && pullRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <div className="text-center">
          <p>Failed to load pull requests</p>
          <p className="text-xs mt-1">{error}</p>
        </div>
      </div>
    );
  }

  if (pullRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        No open pull requests
      </div>
    );
  }

  return (
    <div>
      <h2 className="text-xl font-bold mb-4">Pull Requests ({pullRequests.length})</h2>
      <ScrollArea className="h-[calc(100vh-12rem)]">
        <div className="space-y-1">
          {pullRequests.map((pr) => (
            <div
              key={pr.Number}
              className="flex items-start gap-3 px-3 py-2.5 rounded-lg hover:bg-accent/50 transition-colors"
            >
              <span
                className={cn(
                  "mt-0.5 text-lg",
                  pr.State === "OPEN" && !pr.IsDraft && "text-green-500",
                  pr.State === "MERGED" && "text-purple-500",
                  pr.State === "CLOSED" && "text-red-500",
                  pr.IsDraft && "text-yellow-500"
                )}
              >
                {pr.IsDraft ? "◇" : "◆"}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">
                  <span className="text-muted-foreground font-mono mr-1">
                    #{pr.Number}
                  </span>
                  {truncate(pr.Title, 80)}
                  {pr.IsDraft && (
                    <Badge variant="warning" className="ml-2 text-xs">DRAFT</Badge>
                  )}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {pr.Author} &middot; {formatTimeAgo(pr.CreatedAt)}
                  <span className="mx-1">&middot;</span>
                  {pr.HeadRef} &rarr; {pr.BaseRef}
                </p>
              </div>
              <div className="shrink-0 flex gap-1 items-center">
                {pr.Labels?.slice(0, 3).map((label) => (
                  <Badge key={label} variant="secondary" className="text-xs">
                    {label}
                  </Badge>
                ))}
              </div>
            </div>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
