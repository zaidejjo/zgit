import { Upload, X, GitCommitHorizontal, ArrowUpFromLine } from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";

export default function PushConfirmDialog() {
  const { showPushDialog, pushCommits, confirmPush, cancelPush, status } = useAppStore();

  if (!showPushDialog) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-[36rem] max-h-[80vh] bg-background rounded-xl border shadow-2xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-full bg-primary/10">
              <Upload className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="text-lg font-semibold">Push to origin</h2>
              <p className="text-sm text-muted-foreground">
                {status?.branch ? (
                  <>
                    Pushing <span className="font-mono">{status.branch}</span> to <span className="font-mono">origin/{status.branch}</span>
                  </>
                ) : "Push pending changes"}
              </p>
            </div>
          </div>
          <button
            className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            onClick={cancelPush}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Commit list */}
        <div className="flex-1 min-h-0 p-4">
          {pushCommits.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-muted-foreground">
              <ArrowUpFromLine className="w-8 h-8" />
              <p className="text-sm">No new commits to push</p>
              <p className="text-xs">Everything up to date</p>
            </div>
          ) : (
            <>
              <p className="text-sm font-medium mb-3 text-muted-foreground">
                {pushCommits.length} commit{pushCommits.length !== 1 ? "s" : ""} to push
              </p>
              <ScrollArea className="max-h-64">
                <div className="space-y-1">
                  {pushCommits.map((c) => (
                    <div
                      key={c.hash}
                      className="flex items-start gap-3 px-3 py-2 rounded-lg bg-muted/30 border border-border/50"
                    >
                      <GitCommitHorizontal className="w-4 h-4 text-muted-foreground shrink-0 mt-0.5" />
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-mono truncate">{c.message}</p>
                        <div className="flex gap-3 text-xs text-muted-foreground mt-0.5">
                          <span>{c.author}</span>
                          <span className="font-mono">{c.hash.slice(0, 7)}</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-2 px-5 py-4 border-t bg-muted/20">
          <Button variant="outline" onClick={cancelPush}>
            Cancel
          </Button>
          <Button
            onClick={confirmPush}
            disabled={pushCommits.length === 0}
          >
            <Upload className="w-4 h-4 mr-1.5" />
            Push {pushCommits.length > 0 ? `(${pushCommits.length} commit${pushCommits.length !== 1 ? "s" : ""})` : ""}
          </Button>
        </div>
      </div>
    </div>
  );
}
