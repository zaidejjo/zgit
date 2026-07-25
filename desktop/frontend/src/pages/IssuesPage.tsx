import { useEffect } from "react";
import { useAppStore } from "@/store/app";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { formatTimeAgo, truncate } from "@/lib/utils";
import { cn } from "@/lib/utils";

export default function IssuesPage() {
  const { issues, loading, error, fetchIssues } = useAppStore();

  useEffect(() => {
    fetchIssues();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
        No open issues
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
              key={issue.Number}
              className="flex items-start gap-3 px-3 py-2.5 rounded-lg hover:bg-accent/50 transition-colors"
            >
              <span
                className={cn(
                  "mt-0.5",
                  issue.State === "OPEN" ? "text-green-500" : "text-red-500"
                )}
              >
                {issue.State === "OPEN" ? "●" : "●"}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">
                  <span className="text-muted-foreground font-mono mr-1">
                    #{issue.Number}
                  </span>
                  {truncate(issue.Title, 80)}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {issue.Author} &middot; {formatTimeAgo(issue.CreatedAt)}
                  {issue.Comments > 0 && (
                    <>
                      <span className="mx-1">&middot;</span>
                      {issue.Comments} comments
                    </>
                  )}
                </p>
              </div>
              <div className="shrink-0 flex gap-1 items-center flex-wrap">
                {issue.Labels?.slice(0, 4).map((label) => (
                  <Badge
                    key={label.Name}
                    variant="secondary"
                    className="text-xs"
                    style={label.Color ? {
                      borderColor: `#${label.Color}`,
                      backgroundColor: `#${label.Color}20`,
                    } : undefined}
                  >
                    {label.Name}
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
