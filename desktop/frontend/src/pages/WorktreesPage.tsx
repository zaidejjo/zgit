import { useEffect, useState } from "react";
import {
  GitFork, Plus, Trash2, Lock, LockOpen, FolderOpen,
  CheckCheck, X, RefreshCw, AlertTriangle,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

export default function WorktreesPage() {
  const {
    worktrees, loading, repoPath, fetchWorktrees,
    addWorktree, removeWorktree, lockWorktree, unlockWorktree, pruneWorktrees,
    openRepo,
  } = useAppStore();

  const [showAdd, setShowAdd] = useState(false);
  const [newPath, setNewPath] = useState("");
  const [newRef, setNewRef] = useState("");
  const [newBranchMode, setNewBranchMode] = useState(false);
  const [forceAdd, setForceAdd] = useState(false);
  const [adding, setAdding] = useState(false);

  const [removeTarget, setRemoveTarget] = useState<string | null>(null);
  const [forceRemove, setForceRemove] = useState(false);
  const [removing, setRemoving] = useState(false);

  const [lockTarget, setLockTarget] = useState<string | null>(null);
  const [lockReason, setLockReason] = useState("");
  const [switchingTo, setSwitchingTo] = useState<string | null>(null);

  useEffect(() => {
    fetchWorktrees();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath]);

  const busy = !!loading.worktrees && worktrees.length === 0;

  const handleAdd = async () => {
    if (!newPath.trim() || adding) return;
    setAdding(true);
    await addWorktree(newPath.trim(), newRef.trim(), newBranchMode, forceAdd, false);
    setAdding(false);
    setShowAdd(false);
    setNewPath("");
    setNewRef("");
    setNewBranchMode(false);
    setForceAdd(false);
  };

  const handleRemove = async () => {
    if (!removeTarget || removing) return;
    setRemoving(true);
    await removeWorktree(removeTarget, forceRemove);
    setRemoving(false);
    setRemoveTarget(null);
    setForceRemove(false);
  };

  const handleLock = async () => {
    if (!lockTarget) return;
    await lockWorktree(lockTarget, lockReason.trim());
    setLockTarget(null);
    setLockReason("");
  };

  const handleOpen = async (path: string) => {
    setSwitchingTo(path);
    await openRepo(path);
    setSwitchingTo(null);
  };

  if (busy) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading worktrees...
      </div>
    );
  }

  if (!loading.worktrees && worktrees.length === 0) {
    return (
      <div className="h-full flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-bold">Worktrees</h2>
          <Button size="sm" onClick={() => setShowAdd(true)}>
            <Plus className="w-4 h-4 mr-1.5" />
            New Worktree
          </Button>
        </div>
        <EmptyState
          icon={<GitFork className="w-16 h-16" />}
          title="Single worktree"
          description={repoPath
            ? "Only the main worktree exists. Add a linked worktree to work on multiple branches at once."
            : "Open a repository to manage its worktrees."}
          action={repoPath ? { label: "New Worktree", onClick: () => setShowAdd(true) } : undefined}
          className="flex-1"
        />
        <AddWorktreeDialog
          open={showAdd}
          onOpenChange={setShowAdd}
          path={newPath}
          setPath={setNewPath}
          refName={newRef}
          setRefName={setNewRef}
          newBranchMode={newBranchMode}
          setNewBranchMode={setNewBranchMode}
          forceAdd={forceAdd}
          setForceAdd={setForceAdd}
          adding={adding}
          onAdd={handleAdd}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Worktrees</h2>
          <Badge variant="outline" className="text-xs">{worktrees.length}</Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => pruneWorktrees()} disabled={!!loading.worktrees}>
            <RefreshCw className="w-4 h-4 mr-1.5" />
            Prune
          </Button>
          <Button size="sm" onClick={() => setShowAdd(true)}>
            <Plus className="w-4 h-4 mr-1.5" />
            New Worktree
          </Button>
        </div>
      </div>

      <ScrollArea className="h-[calc(100vh-12rem)]">
        <div className="space-y-0.5">
          {worktrees.map((w) => {
            const isCurrent = repoPath !== "" && (w.path === repoPath);
            return (
              <div
                key={w.path}
                className={cn(
                  "group flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors",
                  isCurrent ? "bg-primary/10 border border-primary/20" : "hover:bg-accent/50",
                )}
              >
                <span className={cn(
                  "w-3 h-3 rounded-full shrink-0",
                  isCurrent ? "bg-primary" : "bg-muted-foreground/30 group-hover:bg-muted-foreground/50",
                )} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className={cn("font-mono text-sm font-medium truncate", isCurrent && "text-primary")}>
                      {w.path}
                    </span>
                    {w.is_main && <Badge variant="secondary" className="text-[10px] h-4 shrink-0">main</Badge>}
                    {w.is_bare && <Badge variant="outline" className="text-[10px] h-4 shrink-0">bare</Badge>}
                    {w.prunable && <Badge variant="destructive" className="text-[10px] h-4 shrink-0">prunable</Badge>}
                    {w.locked && (
                      <Badge variant="outline" className="text-[10px] h-4 shrink-0">
                        <Lock className="w-2.5 h-2.5 mr-0.5" />
                        locked
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">
                    {w.branch ? w.branch : w.is_detached ? "(detached)" : ""}
                    {w.head ? ` · ${w.head.slice(0, 8)}` : ""}
                    {w.lock_reason ? ` · ${w.lock_reason}` : ""}
                  </p>
                </div>
                <div className="flex gap-1 shrink-0">
                  {!isCurrent && !w.is_bare && (
                    <Button
                      size="sm"
                      className="h-7 px-2 text-xs"
                      disabled={switchingTo === w.path}
                      onClick={() => handleOpen(w.path)}
                    >
                      <FolderOpen className="w-3.5 h-3.5 mr-1" />
                      {switchingTo === w.path ? "Opening..." : "Open"}
                    </Button>
                  )}
                  {w.locked ? (
                    <Button
                      variant="ghost" size="sm" className="h-7 px-2 text-xs"
                      onClick={() => unlockWorktree(w.path)}
                    >
                      <LockOpen className="w-3.5 h-3.5 mr-1" />
                      Unlock
                    </Button>
                  ) : (
                    !w.is_main && (
                      <Button
                        variant="ghost" size="sm" className="h-7 px-2 text-xs"
                        onClick={() => { setLockTarget(w.path); setLockReason(""); }}
                      >
                        <Lock className="w-3.5 h-3.5 mr-1" />
                        Lock
                      </Button>
                    )
                  )}
                  {!w.is_main && (
                    <Button
                      variant="ghost" size="sm"
                      className="h-7 px-2 text-xs text-muted-foreground hover:text-destructive"
                      onClick={() => { setRemoveTarget(w.path); setForceRemove(false); }}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </ScrollArea>

      <AddWorktreeDialog
        open={showAdd}
        onOpenChange={setShowAdd}
        path={newPath}
        setPath={setNewPath}
        refName={newRef}
        setRefName={setNewRef}
        newBranchMode={newBranchMode}
        setNewBranchMode={setNewBranchMode}
        forceAdd={forceAdd}
        setForceAdd={setForceAdd}
        adding={adding}
        onAdd={handleAdd}
      />

      <Dialog open={!!removeTarget} onOpenChange={(open) => { if (!open) { setRemoveTarget(null); setForceRemove(false); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <Trash2 className="w-5 h-5" />
              Remove Worktree
            </DialogTitle>
            <DialogDescription>
              Remove <span className="font-mono font-medium text-foreground">{removeTarget}</span>?
              Uncommitted changes will block removal unless forced.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center gap-2 p-3 rounded-lg bg-muted/50 text-sm">
              <AlertTriangle className="w-4 h-4 text-destructive shrink-0" />
              <span className="text-muted-foreground">
                {forceRemove
                  ? "Force remove deletes the worktree even with dirty state. Uncommitted work is lost."
                  : "Safe remove refuses when the worktree is dirty or locked."}
              </span>
            </div>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <Checkbox checked={forceRemove} onChange={() => setForceRemove(!forceRemove)} />
              <span>Force remove (<span className="font-mono">--force</span>)</span>
            </label>
            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              <Button variant="destructive" onClick={handleRemove} disabled={removing}>
                <Trash2 className="w-4 h-4 mr-1.5" />
                {removing ? "Removing..." : "Remove"}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!lockTarget} onOpenChange={(open) => { if (!open) { setLockTarget(null); setLockReason(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Lock className="w-5 h-5" />
              Lock Worktree
            </DialogTitle>
            <DialogDescription>
              Lock <span className="font-mono font-medium text-foreground">{lockTarget}</span> to prevent pruning and accidental removal.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <input
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="Reason (optional)"
              value={lockReason}
              onChange={(e) => setLockReason(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleLock()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              <Button onClick={handleLock}>
                <CheckCheck className="w-4 h-4 mr-1.5" />
                Lock
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function AddWorktreeDialog(props: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  path: string;
  setPath: (v: string) => void;
  refName: string;
  setRefName: (v: string) => void;
  newBranchMode: boolean;
  setNewBranchMode: (v: boolean) => void;
  forceAdd: boolean;
  setForceAdd: (v: boolean) => void;
  adding: boolean;
  onAdd: () => void;
}) {
  const { open, onOpenChange, path, setPath, refName, setRefName, newBranchMode, setNewBranchMode, forceAdd, setForceAdd, adding, onAdd } = props;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Worktree</DialogTitle>
          <DialogDescription>
            Check out a branch in a separate directory. Useful for parallel work without switching branches.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <input
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring font-mono"
            placeholder="/path/to/worktree  (e.g. ../myrepo-hotfix)"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            autoFocus
          />
          <input
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring font-mono"
            placeholder={newBranchMode ? "New branch name" : "Branch / tag / commit (empty = current HEAD)"}
            value={refName}
            onChange={(e) => setRefName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && onAdd()}
          />
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <Checkbox checked={newBranchMode} onChange={() => setNewBranchMode(!newBranchMode)} />
            <span>Create new branch (<span className="font-mono">-b</span>)</span>
          </label>
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <Checkbox checked={forceAdd} onChange={() => setForceAdd(!forceAdd)} />
            <span>Force checkout already-used branch (<span className="font-mono">--force</span>)</span>
          </label>
          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button variant="outline">
                <X className="w-4 h-4 mr-1.5" />
                Cancel
              </Button>
            </DialogClose>
            <Button onClick={onAdd} disabled={!path.trim() || adding}>
              <CheckCheck className="w-4 h-4 mr-1.5" />
              {adding ? "Adding..." : "Add Worktree"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
