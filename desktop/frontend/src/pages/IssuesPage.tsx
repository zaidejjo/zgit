import { useEffect } from "react";
import { useAppStore } from "@/store/app";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { formatTimeAgo, truncate } from "@/lib/utils";
import { cn } from "@/lib/utils";

export default function IssuesPage() {
  const { issues, loading, error, ghAuthenticated, setLoginDialogOpen, fetchIssues } = useAppStore();

  useEffect(() => {
    fetchIssues();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Not authenticated
  if (!ghAuthenticated) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-muted-foreground gap-3">
        <span className="text-3xl">●</span>
        <p>Sign in to GitHub to view issues</p>
        <button
          className="text-sm text-primary underline underline-offset-2 hover:text-primary/80"
          onClick={() => setLoginDialogOpen(true)}
        >
          Open Settings
        </button>
      </div>
    );
  }

  if (loading.issues && issues.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading issues...
      </div>
    );
  }

  if (error && issues.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <div className="text-center">
          <p>Failed to load issues</p>
          <p className="text-xs mt-1">{error}</p>
        </div>
      </div>
    );
  }

  if (issues.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <div className="text-center">
          <p className="text-lg">No issues found</p>
          <p className="text-xs mt-1">All issues are closed or this repository has no issues.</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      <h2 className="text-xl font-bold mb-4">Issues ({issues.length})</h2>
      <ScrollArea className="h-[calc(100vh-12rem)]">
        <div className="space-y-1">
          {issues.map((issue) => (
            <div
              key={issue.number}
              className="flex items-start gap-3 px-3 py-2.5 rounded-lg hover:bg-accent/50 transition-colors"
            >
              <span
                className={cn(
                  "mt-0.5",
                  issue.state === "OPEN" ? "text-green-500" : "text-red-500"
                )}
              >
                {issue.state === "OPEN" ? "●" : "●"}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">
                    <span className="text-muted-foreground font-mono mr-1">
                      #{issue.number}
                    </span>
                  {truncate(issue.title, 80)}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {issue.author} &middot; {formatTimeAgo(issue.created_at)}
                  {issue.comments > 0 && (
                    <>
                      <span className="mx-1">&middot;</span>
                      {issue.comments} comments
                    </>
                  )}
                </p>
              </div>
              <div className="shrink-0 flex gap-1 items-center flex-wrap">
                {(issue.labels || []).slice(0, 4).map((label) => (
                  <Badge
                    key={label.name}
                    variant="secondary"
                    className="text-xs"
                    style={label.color ? {
                      borderColor: `#${label.color}`,
                      backgroundColor: `#${label.color}20`,
                    } : undefined}
                  >
                    {label.name}
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
