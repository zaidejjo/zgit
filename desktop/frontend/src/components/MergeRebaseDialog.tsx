import { useAppStore } from "../store/app";
import { GitMerge, GitBranch, X } from "lucide-react";

export default function MergeRebaseDialog() {
  const {
    mergeRebaseDialog, closeMergeRebaseDialog,
    executeMerge, executeRebaseOnto,
    currentBranch,
  } = useAppStore();

  if (!mergeRebaseDialog) return null;

  const { branch: draggedBranch, targetHash, targetMsg } = mergeRebaseDialog;
  const isBranchDrop = draggedBranch.length < 40; // heuristic: short name = branch reference

  return (
    <div className="fixed inset-0 z-50 bg-background/60 backdrop-blur-sm flex items-center justify-center">
      <div className="w-full max-w-sm bg-background border border-border rounded-xl shadow-xl p-5">
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-base font-semibold">
            {isBranchDrop ? `Merge ${draggedBranch}` : "Rebase onto commit"}
          </h3>
          <button onClick={closeMergeRebaseDialog} className="p-1 rounded hover:bg-muted transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Description */}
        <p className="text-sm text-muted-foreground mb-4">
          {isBranchDrop ? (
            <>
              Merge <span className="font-mono font-medium text-foreground">{draggedBranch}</span>{" "}
              into <span className="font-mono font-medium text-foreground">{currentBranch || "current branch"}</span>?
            </>
          ) : (
            <>
              Rebase <span className="font-mono font-medium text-foreground">{currentBranch || "current branch"}</span>{" "}
              onto commit <span className="font-mono text-foreground">{targetHash.slice(0, 7)}</span>?
              <br />
              <span className="text-xs text-muted-foreground/70">{truncate(targetMsg, 60)}</span>
            </>
          )}
        </p>

        {/* Actions */}
        <div className="flex items-center gap-2 justify-end">
          <button
            onClick={closeMergeRebaseDialog}
            className="px-3 py-1.5 text-xs font-medium rounded-md border border-border text-muted-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>

          {isBranchDrop && (
            <button
              onClick={() => executeMerge(draggedBranch)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <GitMerge className="w-3.5 h-3.5" />
              Merge
            </button>
          )}

          <button
            onClick={() => {
              if (isBranchDrop) {
                // Rebase current branch onto the tip of the dragged branch
                executeRebaseOnto(draggedBranch, draggedBranch);
              } else {
                executeRebaseOnto(currentBranch!, targetHash);
              }
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md bg-purple-600 text-white hover:bg-purple-700 transition-colors"
          >
            <GitBranch className="w-3.5 h-3.5" />
            Rebase
          </button>
        </div>
      </div>
    </div>
  );
}

function truncate(s: string, len: number): string {
  if (!s || s.length <= len) return s || "";
  return s.slice(0, len - 3) + "...";
}
