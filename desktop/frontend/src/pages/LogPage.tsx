import { useEffect } from "react";
import { useAppStore } from "@/store/app";
import { ScrollArea } from "@/components/ui/scroll-area";
import { formatTimeAgo, truncate } from "@/lib/utils";

export default function LogPage() {
  const { log, loading, fetchLog } = useAppStore();

  useEffect(() => {
    fetchLog();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading.log && log.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading commits...
      </div>
    );
  }

  if (log.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        No commits yet
      </div>
    );
  }

  return (
    <div>
      <h2 className="text-xl font-bold mb-4">Commit Log ({log.length})</h2>
      <ScrollArea className="h-[calc(100vh-12rem)]">
        <div className="space-y-1">
          {log.map((commit) => (
            <div
              key={commit.Hash}
              className="flex items-start gap-3 px-3 py-2 rounded-lg hover:bg-accent/50 transition-colors"
            >
              <code className="text-xs font-mono text-muted-foreground mt-0.5 shrink-0">
                {commit.Hash.slice(0, 7)}
              </code>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {truncate(commit.Message, 80)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {commit.Author} &middot; {formatTimeAgo(commit.Timestamp)}
                </p>
              </div>
              {commit.RefNames && (
                <div className="shrink-0 flex gap-1">
                  {commit.RefNames.split(", ").slice(0, 2).map((ref) => (
                    <span
                      key={ref}
                      className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-mono"
                    >
                      {ref}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
