import { useEffect, useState, useRef } from "react";
import { RotateCcw, ChevronDown, X, History } from "lucide-react";
import { useAppStore, ReflogEntry } from "../store/app";

export default function UndoBanner() {
  const {
    reflog, undoDescription, fetchReflog, undoLastAction, clearUndoDescription,
  } = useAppStore();

  const [showDropdown, setShowDropdown] = useState(false);
  const [undone, setUndone] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  // Fetch reflog on mount
  useEffect(() => {
    fetchReflog(5);
  }, [fetchReflog]);

  // Auto-dismiss undo description after 8 seconds
  useEffect(() => {
    if (undoDescription) {
      setUndone(true);
      timerRef.current = setTimeout(() => {
        clearUndoDescription();
        setUndone(false);
      }, 8000);
      return () => clearTimeout(timerRef.current);
    }
  }, [undoDescription, clearUndoDescription]);

  // Close dropdown on outside click
  useEffect(() => {
    if (!showDropdown) return;
    const handleClick = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [showDropdown]);

  // Find last undoable action
  const lastUndoable = reflog.find((e) => e.undoable);

  // No undoable action and no pending undo description → hidden
  if (!lastUndoable && !undone) return null;

  const handleUndo = async () => {
    setShowDropdown(false);
    await undoLastAction();
  };

  const dismissUndo = () => {
    clearUndoDescription();
    setUndone(false);
  };

  return (
    <div className="border-b border-border/30 bg-card/60">
      <div className="flex items-center gap-2 px-4 py-1.5 text-xs">
        <History className="w-3.5 h-3.5 text-muted-foreground shrink-0" />

        {undone && undoDescription ? (
          /* 📋 Undo success message */
          <>
            <span className="text-foreground font-medium">✓</span>
            <span className="text-foreground/80 truncate">{undoDescription}</span>
            <button
              onClick={dismissUndo}
              className="ml-auto p-1 rounded hover:bg-muted transition-colors"
              title="Dismiss"
            >
              <X className="w-3 h-3" />
            </button>
          </>
        ) : lastUndoable ? (
          /* 🔄 Undo available — show button + dropdown */
          <>
            <span className="text-muted-foreground truncate flex-1">
              Last action:{" "}
              <span className="font-mono text-foreground/70">{truncate(lastUndoable.subject, 55)}</span>
            </span>

            {/* Undo button */}
            <button
              onClick={handleUndo}
              className="flex items-center gap-1 px-2 py-1 rounded bg-primary/10 text-primary hover:bg-primary/20 transition-colors font-medium"
              title="Undo last action"
            >
              <RotateCcw className="w-3 h-3" />
              Undo
            </button>

            {/* Dropdown toggle */}
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setShowDropdown(!showDropdown)}
                className="p-1 rounded hover:bg-muted transition-colors"
                title="Recent actions"
              >
                <ChevronDown className="w-3 h-3" />
              </button>

              {showDropdown && (
                <div className="absolute right-0 top-full mt-1 w-72 bg-popover border border-border rounded-md shadow-lg z-50 overflow-hidden">
                  <div className="px-3 py-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider border-b border-border">
                    Recent Actions
                  </div>
                  <div className="max-h-48 overflow-y-auto">
                    {reflog.length === 0 ? (
                      <div className="px-3 py-4 text-center text-muted-foreground text-[11px]">
                        No recent actions
                      </div>
                    ) : (
                      reflog.map((entry) => (
                        <div
                          key={entry.sequence}
                          className="flex items-center gap-2 px-3 py-2 hover:bg-muted/50 transition-colors"
                        >
                          <span className="text-[10px] font-mono text-muted-foreground shrink-0">
                            {hashPrefix(entry.hash)}
                          </span>
                          <span className="text-xs truncate flex-1">{entry.subject}</span>
                          <ActionBadge action={entry.action} />
                        </div>
                      ))
                    )}
                  </div>
                </div>
              )}
            </div>
          </>
        ) : null}
      </div>
    </div>
  );
}

function ActionBadge({ action }: { action: string }) {
  const colors: Record<string, string> = {
    commit: "bg-success/10 text-success",
    reset: "bg-warning/10 text-warning",
    merge: "bg-primary/10 text-primary",
    rebase: "bg-[hsl(var(--pr-merged))/0.1] text-[hsl(var(--pr-merged))]",
    "cherry-pick": "bg-primary/10 text-primary",
    revert: "bg-destructive/10 text-destructive",
    checkout: "bg-muted/40 text-muted-foreground",
    branch: "bg-primary/10 text-primary",
    amend: "bg-warning/10 text-warning",
  };
  const colorClass = colors[action] || "bg-muted/40 text-muted-foreground";
  return (
    <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium uppercase ${colorClass}`}>
      {action}
    </span>
  );
}

function truncate(s: string, len: number): string {
  if (s.length <= len) return s;
  return s.slice(0, len - 3) + "...";
}

function hashPrefix(hash: string): string {
  return hash.length > 7 ? hash.slice(0, 7) : hash;
}
