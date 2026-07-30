import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  GitCommitHorizontal, Undo2, RotateCcw, AlertTriangle,
  GitBranch, GitMerge, GripVertical, Pencil, Trash2,
  Layers, ArrowUpDown, Network,
} from "lucide-react";
import {
  DndContext, DragEndEvent, DragOverlay, DragStartEvent,
  useDraggable, useDroppable, PointerSensor, useSensor, useSensors,
} from "@dnd-kit/core";
import {
  SortableContext, verticalListSortingStrategy,
  useSortable, arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useAppStore, RebaseCommitOp, Commit } from "@/store/app";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { cn, formatTimeAgo, truncate } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";
import MergeRebaseDialog from "@/components/MergeRebaseDialog";
import InteractiveGitGraph from "@/components/InteractiveGitGraph";

const RESET_MODES = [
  { value: "soft", label: "--soft", desc: "Keep all changes staged", danger: false },
  { value: "mixed", label: "--mixed", desc: "Keep changes, unstage them", danger: false },
  { value: "hard", label: "--hard", desc: "⚠ Discard all changes permanently", danger: true },
] as const;

type CommitAction = "pick" | "reword" | "squash" | "fixup" | "drop";

const ACTION_COLORS: Record<CommitAction, string> = {
  pick: "text-success bg-success/10",
  reword: "text-amber-600 bg-amber-500/10",
  squash: "text-blue-600 bg-blue-500/10",
  fixup: "text-cyan-600 bg-cyan-500/10",
  drop: "text-destructive bg-destructive/10 line-through",
};

export default function LogPage() {
  const navigate = useNavigate();
  const {
    log, graphLog, graphView, loading, fetchLog, fetchGraphLog, toggleGraphView,
    cherryPick, revertCommit, resetCommit,
    rebaseMode, rebaseCommits, rebaseOnto,
    enterRebaseMode, exitRebaseMode,
    reorderCommits, setCommitAction, setCommitMessage,
    applyRebase,
    showMergeRebaseDialog, mergeRebaseDialog,
    repoPath,
  } = useAppStore();

  // Cherry-pick / revert / reset state
  const [actionTarget, setActionTarget] = useState<Commit | null>(null);
  const [actionType, setActionType] = useState<"cherry-pick" | "revert" | null>(null);
  const [resetMode, setResetMode] = useState<string>("mixed");
  const [showReset, setShowReset] = useState(false);
  const [resetTarget, setResetTarget] = useState<Commit | null>(null);

  // DnD state
  const [activeDragId, setActiveDragId] = useState<string | null>(null);
  const [rewordIndex, setRewordIndex] = useState<number | null>(null);
  const [rewordText, setRewordText] = useState("");

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  );

  useEffect(() => {
    fetchLog();
    fetchGraphLog();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── Handlers ──

  const handleAction = async () => {
    if (!actionTarget) return;
    if (actionType === "cherry-pick") await cherryPick(actionTarget.hash);
    else if (actionType === "revert") await revertCommit(actionTarget.hash);
    setActionTarget(null);
    setActionType(null);
  };

  const handleReset = async () => {
    if (!resetTarget) return;
    await resetCommit(resetTarget.hash, resetMode);
    setShowReset(false);
    setResetTarget(null);
  };

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveDragId(event.active.id as string);
  }, []);

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    setActiveDragId(null);
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    // If in rebase mode and dragging a commit
    if (rebaseMode) {
      const oldIndex = rebaseCommits.findIndex((c) => c.sha === active.id);
      const newIndex = rebaseCommits.findIndex((c) => c.sha === over.id);
      if (oldIndex !== -1 && newIndex !== -1) {
        reorderCommits(oldIndex, newIndex);
      }
      return;
    }

    // Dragging a branch badge to a commit row
    const dropTarget = over.id as string;
    const dragData = event.active.data.current as { branch?: string; hash?: string } | undefined;
    if (dragData?.branch && dragData?.hash) {
      // Dragging a branch badge from another commit to this commit
      showMergeRebaseDialog(dragData.branch, dropTarget, "");
    }
  }, [rebaseMode, rebaseCommits, reorderCommits, showMergeRebaseDialog]);

  const commits = rebaseMode ? rebaseCommits : log;

  if (loading.log && log.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading commits...
      </div>
    );
  }

  if (!loading.log && log.length === 0 && !rebaseMode) {
    return (
      <div className="h-full flex flex-col gap-3">
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-bold">Commit Log</h2>
        </div>
        <EmptyState
          icon={<GitCommitHorizontal className="w-16 h-16" />}
          title="No commits yet"
          description={repoPath ? "Create your first commit to start your git timeline!" : "Open a repository to view its commit history."}
          action={repoPath ? { label: "Go to Status", onClick: () => navigate({ to: "/" }) } : undefined}
          className="flex-1"
        />
      </div>
    );
  }

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <div className="h-full flex flex-col gap-3">
        {/* Toolbar */}
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-bold">
            {rebaseMode ? "Interactive Rebase" : `Commit Log (${log.length})`}
          </h2>
          <div className="flex-1" />
          {rebaseMode ? (
            <>
              <input
                className="w-48 px-2 py-1 text-xs font-mono bg-background border border-border rounded"
                placeholder="Onto ref (e.g. main, HEAD~5)"
                value={rebaseOnto}
                onChange={(e) => useAppStore.setState({ rebaseOnto: e.target.value })}
              />
              <button
                onClick={() => applyRebase()}
                disabled={loading.rebase}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
              >
                {loading.rebase ? (
                  <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                ) : (
                  <GitCommitHorizontal className="w-3.5 h-3.5" />
                )}
                Apply Rebase
              </button>
              <button
                onClick={exitRebaseMode}
                className="px-2.5 py-1.5 text-xs font-medium rounded-md border border-border text-muted-foreground hover:bg-muted transition-colors"
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              <button
                onClick={toggleGraphView}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md transition-colors",
                  graphView
                    ? "bg-primary text-primary-foreground"
                    : "border border-border text-muted-foreground hover:bg-muted"
                )}
              >
                <Network className="w-3.5 h-3.5" />
                Graph
              </button>
              <button
                onClick={enterRebaseMode}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md bg-purple-600 text-white hover:bg-purple-700 transition-colors"
              >
                <ArrowUpDown className="w-3.5 h-3.5" />
                Rebase Mode
              </button>
            </>
          )}
        </div>

        {/* Legend (rebase mode) */}
        {rebaseMode && (
          <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
            <span className="text-success font-medium">pick</span>
            <span className="text-amber-600 font-medium">reword</span>
            <span className="text-blue-600 font-medium">squash</span>
            <span className="text-cyan-600 font-medium">fixup</span>
            <span className="text-destructive font-medium">drop</span>
            <span className="text-muted-foreground/50">|</span>
            <span>Drag to reorder</span>
          </div>
        )}

        <ScrollArea className="flex-1">
          {rebaseMode ? (
            /* ── Rebase Mode: Sortable Commit List ── */
            <SortableContext items={rebaseCommits.map((c) => c.sha)} strategy={verticalListSortingStrategy}>
              <div className="space-y-1">
                {rebaseCommits.map((rc, idx) => (
                  <RebaseSortableCommit
                    key={rc.sha + idx}
                    rc={rc}
                    index={idx}
                    isActive={activeDragId === rc.sha}
                    onActionChange={(action) => setCommitAction(idx, action)}
                    rewordIndex={rewordIndex}
                    rewordText={rewordText}
                    onRewordStart={() => { setRewordIndex(idx); setRewordText(rc.new_message || ""); }}
                    onRewordChange={setRewordText}
                    onRewordApply={() => {
                      if (rewordIndex !== null) {
                        setCommitMessage(rewordIndex, rewordText);
                        setRewordIndex(null);
                      }
                    }}
                    onRewordCancel={() => setRewordIndex(null)}
                  />
                ))}
                {rebaseCommits.length === 0 && (
                  <div className="py-8 text-center text-muted-foreground text-sm">No commits selected</div>
                )}
              </div>
            </SortableContext>
          ) : graphView ? (
            /* ── Graph View ── */
            <InteractiveGitGraph commits={graphLog} loading={loading.graphLog} />
          ) : (
            /* ── Normal Mode: Commit Log with DnD Branch Badges ── */
            <div className="space-y-1">
              {log.map((commit) => (
                <CommitRow
                  key={commit.hash}
                  commit={commit}
                  onCherryPick={() => { setActionTarget(commit); setActionType("cherry-pick"); }}
                  onRevert={() => { setActionTarget(commit); setActionType("revert"); }}
                  onReset={() => { setResetTarget(commit); setResetMode("mixed"); setShowReset(true); }}
                  isActive={false}
                />
              ))}
            </div>
          )}
        </ScrollArea>

        {/* ── Dialogs ── */}

        {/* Cherry-pick / Revert */}
        <Dialog open={!!actionTarget} onOpenChange={(open) => { if (!open) { setActionTarget(null); setActionType(null); } }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {actionType === "cherry-pick" ? (
                  <><GitCommitHorizontal className="w-4 h-4" /> Cherry-Pick Commit</>
                ) : (
                  <><Undo2 className="w-4 h-4" /> Revert Commit</>
                )}
              </DialogTitle>
              <DialogDescription>
                {actionType === "cherry-pick"
                  ? "Apply this commit's changes onto your current branch."
                  : "Create a new commit that undoes this commit's changes."}
              </DialogDescription>
            </DialogHeader>
            <div className="py-3 px-4 rounded-lg bg-muted/30 border text-sm space-y-1">
              <p className="font-mono text-xs text-muted-foreground">{actionTarget?.hash.slice(0, 7)}</p>
              <p className="font-medium">{truncate(actionTarget?.message || "", 80)}</p>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setActionTarget(null); setActionType(null); }}>
                Cancel
              </Button>
              <Button onClick={handleAction}>
                {actionType === "cherry-pick" ? "Cherry-Pick" : "Revert"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* Reset */}
        <Dialog open={showReset} onOpenChange={(open) => { if (!open) { setShowReset(false); setResetTarget(null); } }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <RotateCcw className="w-4 h-4" /> Reset to Commit
              </DialogTitle>
              <DialogDescription>
                Move the current branch pointer to the selected commit.
              </DialogDescription>
            </DialogHeader>
            <div className="py-3 px-4 rounded-lg bg-muted/30 border text-sm space-y-1">
              <p className="font-mono text-xs text-muted-foreground">{resetTarget?.hash.slice(0, 7)}</p>
              <p className="font-medium">{truncate(resetTarget?.message || "", 80)}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium">Reset mode</p>
              {RESET_MODES.map((mode) => (
                <button
                  key={mode.value}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-lg border text-left transition-colors",
                    resetMode === mode.value
                      ? mode.danger ? "border-destructive bg-destructive/10" : "border-primary bg-primary/5"
                      : "border-border hover:bg-accent/50"
                  )}
                  onClick={() => setResetMode(mode.value)}
                >
                  <div className={cn(
                    "w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0",
                    resetMode === mode.value
                      ? mode.danger ? "border-destructive" : "border-primary"
                      : "border-muted-foreground"
                  )}>
                    {resetMode === mode.value && (
                      <div className={cn("w-2 h-2 rounded-full", mode.danger ? "bg-destructive" : "bg-primary")} />
                    )}
                  </div>
                  <div>
                    <p className={cn("text-sm font-medium", mode.danger && resetMode === mode.value && "text-destructive")}>
                      {mode.label}
                    </p>
                    <p className="text-xs text-muted-foreground">{mode.desc}</p>
                  </div>
                </button>
              ))}
            </div>
            {resetMode === "hard" && (
              <div className="flex items-start gap-2 p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-sm text-destructive">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <p>This will permanently discard all uncommitted changes. This cannot be undone.</p>
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => { setShowReset(false); setResetTarget(null); }}>
                Cancel
              </Button>
              <Button variant={resetMode === "hard" ? "destructive" : "default"} onClick={handleReset}>
                {resetMode === "hard" ? "Reset --hard" : `Reset --${resetMode}`}
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* Drag Overlay */}
        <DragOverlay>
          {activeDragId ? (
            <div className="px-3 py-2 rounded-lg bg-muted border border-border shadow-lg text-sm font-mono">
              {activeDragId.slice(0, 12)}...
            </div>
          ) : null}
        </DragOverlay>
      </div>

      {/* Merge / Rebase Dialog */}
      <MergeRebaseDialog />
    </DndContext>
  );
}

/* ─── Normal Commit Row (with draggable branch badges) ─── */

function CommitRow({
  commit,
  onCherryPick,
  onRevert,
  onReset,
  isActive,
}: {
  commit: { hash: string; author: string; timestamp: string; message: string; ref_names?: string };
  onCherryPick: () => void;
  onRevert: () => void;
  onReset: () => void;
  isActive: boolean;
}) {
  const { setNodeRef } = useDroppable({ id: commit.hash, data: { commit } });

  // Parse ref names into branch and tag references
  const refs = commit.ref_names
    ? commit.ref_names.split(", ").filter(Boolean)
    : [];

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "group flex items-start gap-3 px-3 py-2 rounded-lg transition-colors",
        isActive ? "opacity-40" : "hover:bg-accent/50"
      )}
    >
      {/* Hash */}
      <code className="text-xs font-mono text-muted-foreground mt-0.5 shrink-0">
        {commit.hash.slice(0, 7)}
      </code>

      {/* Graph dot */}
      <div className="w-2 h-2 mt-1.5 rounded-full bg-primary/40 shrink-0" />

      {/* Message + meta */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {truncate(commit.message, 80)}
        </p>
        <p className="text-xs text-muted-foreground">
          {commit.author} &middot; {formatTimeAgo(commit.timestamp)}
        </p>
      </div>

      {/* Draggable ref tags */}
      {refs.length > 0 && (
        <div className="shrink-0 flex gap-1 items-center">
          {refs.slice(0, 4).map((ref) => {
            const isBranch = !ref.includes("tag: ");
            const displayName = ref.replace(/^tag: /, "");
            return isBranch ? (
              <DraggableBranchBadge key={ref} branch={displayName} commitHash={commit.hash} />
            ) : (
              <span
                key={ref}
                className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 font-mono"
              >
                {displayName}
              </span>
            );
          })}
        </div>
      )}

      {/* Hover actions */}
      <div className="shrink-0 flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
        <button className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent transition-colors" onClick={onCherryPick} title="Cherry-pick">
          <GitCommitHorizontal className="w-3.5 h-3.5" />
        </button>
        <button className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent transition-colors" onClick={onRevert} title="Revert">
          <Undo2 className="w-3.5 h-3.5" />
        </button>
        <button className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors" onClick={onReset} title="Reset">
          <RotateCcw className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

/* ─── Draggable Branch Badge ─── */

function DraggableBranchBadge({ branch, commitHash }: { branch: string; commitHash: string }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `branch:${branch}`,
    data: { branch, hash: commitHash },
  });

  const style = transform ? {
    transform: CSS.Translate.toString(transform),
    zIndex: 50,
  } : undefined;

  const isHEAD = branch === "HEAD";

  return (
    <span
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className={cn(
        "inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0.5 rounded font-mono cursor-grab active:cursor-grabbing transition-shadow hover:shadow-sm",
        isDragging ? "opacity-50 ring-2 ring-primary" : "",
        isHEAD
          ? "bg-primary/20 text-primary"
          : "bg-blue-500/10 text-blue-600"
      )}
      title={`Drag to merge/rebase onto another commit`}
    >
      <GitBranch className="w-2.5 h-2.5" />
      {branch}
    </span>
  );
}

/* ─── Rebase Sortable Commit ─── */

function RebaseSortableCommit({
  rc, index, isActive, onActionChange,
  rewordIndex, rewordText,
  onRewordStart, onRewordChange, onRewordApply, onRewordCancel,
}: {
  rc: RebaseCommitOp;
  index: number;
  isActive: boolean;
  onActionChange: (action: CommitAction) => void;
  rewordIndex: number | null;
  rewordText: string;
  onRewordStart: () => void;
  onRewordChange: (v: string) => void;
  onRewordApply: () => void;
  onRewordCancel: () => void;
}) {
  const {
    attributes, listeners, setNodeRef, transform, transition,
  } = useSortable({ id: rc.sha });
  const [showMenu, setShowMenu] = useState(false);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const actionLabel = rc.action.charAt(0).toUpperCase() + rc.action.slice(1);
  const colorClass = ACTION_COLORS[rc.action] || "";

  const actions: CommitAction[] = ["pick", "reword", "squash", "fixup", "drop"];

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex items-start gap-3 px-3 py-2 rounded-lg transition-colors relative",
        isActive ? "opacity-40" : "hover:bg-accent/50",
        rc.action === "drop" && "opacity-50"
      )}
    >
      {/* Drag handle */}
      <button
        className="p-1 mt-0.5 text-muted-foreground/40 hover:text-muted-foreground cursor-grab active:cursor-grabbing shrink-0"
        {...attributes}
        {...listeners}
        title="Drag to reorder"
      >
        <GripVertical className="w-3.5 h-3.5" />
      </button>

      {/* Index */}
      <span className="text-xs font-mono text-muted-foreground/50 mt-0.5 w-5 shrink-0 text-right">
        {index + 1}
      </span>

      {/* Action badge (clickable to change) */}
      <div className="relative shrink-0 mt-0.5">
        <button
          onClick={() => setShowMenu(!showMenu)}
          className={cn(
            "text-[10px] px-1.5 py-0.5 rounded font-mono font-medium uppercase hover:ring-1 hover:ring-foreground/20 transition-all",
            colorClass
          )}
        >
          {actionLabel}
        </button>

        {/* Action dropdown */}
        {showMenu && (
          <div className="absolute left-0 top-full mt-1 z-50 bg-popover border border-border rounded-md shadow-lg py-1 min-w-[100px]">
            {actions.map((a) => (
              <button
                key={a}
                className={cn(
                  "w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left hover:bg-muted transition-colors",
                  rc.action === a && "bg-muted font-semibold"
                )}
                onClick={() => { onActionChange(a); setShowMenu(false); }}
              >
                <span className={cn("w-2 h-2 rounded-full", {
                  "bg-green-500": a === "pick",
                  "bg-amber-500": a === "reword",
                  "bg-blue-500": a === "squash",
                  "bg-cyan-500": a === "fixup",
                  "bg-red-500": a === "drop",
                })} />
                <span>{a.charAt(0).toUpperCase() + a.slice(1)}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* SHA */}
      <span className="text-xs font-mono text-muted-foreground mt-0.5 shrink-0">
        {rc.sha.slice(0, 7)}
      </span>

      {/* Reword input or message */}
      <div className="flex-1 min-w-0">
        {rewordIndex === index ? (
          <div className="flex items-center gap-1">
            <input
              className="flex-1 px-2 py-0.5 text-xs font-mono bg-background border border-border rounded"
              value={rewordText}
              onChange={(e) => onRewordChange(e.target.value)}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") onRewordApply();
                if (e.key === "Escape") onRewordCancel();
              }}
            />
            <button onClick={onRewordApply} className="p-0.5 text-success hover:text-success/80" title="Apply">
              ✓
            </button>
            <button onClick={onRewordCancel} className="p-0.5 text-muted-foreground hover:text-foreground" title="Cancel">
              ✕
            </button>
          </div>
        ) : (
          <p className={cn("text-xs font-medium truncate", rc.action === "reword" && "text-amber-600")}>
            {rc.new_message || `Commit ${index + 1}`}
          </p>
        )}
      </div>

      {/* Reword button */}
      {rc.action === "reword" && rewordIndex !== index && (
        <button
          onClick={() => { onRewordStart(); }}
          className="p-1 text-muted-foreground/40 hover:text-amber-600 transition-colors shrink-0"
          title="Edit commit message"
        >
          <Pencil className="w-3 h-3" />
        </button>
      )}
    </div>
  );
}
