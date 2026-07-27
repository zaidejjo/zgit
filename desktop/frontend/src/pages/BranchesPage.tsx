import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  GitBranch, Plus, ArrowRightFromLine, Trash2, Pencil,
  Merge, AlertTriangle, CheckCheck, X, GitCommitHorizontal,
  ArrowUpWideNarrow, ArrowDownWideNarrow,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

export default function BranchesPage() {
  const navigate = useNavigate();
  const {
    branches, currentBranch, loading, fetchBranches,
    checkoutBranch, createBranch, deleteBranch, renameBranch, gitMerge,
    repoPath,
  } = useAppStore();

  // New branch dialog state
  const [showNewBranch, setShowNewBranch] = useState(false);
  const [newBranchName, setNewBranchName] = useState("");

  // Rename dialog state
  const [renameTarget, setRenameTarget] = useState<string | null>(null);
  const [renameName, setRenameName] = useState("");

  // Delete dialog state
  const [deleteTarget, setDeleteTarget] = useState<{ name: string } | null>(null);
  const [forceDelete, setForceDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // Merge dialog state
  const [mergeTarget, setMergeTarget] = useState<string | null>(null);
  const [merging, setMerging] = useState(false);
  const [mergeResult, setMergeResult] = useState<string | null>(null);

  useEffect(() => {
    fetchBranches();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reset merge result when dialog closes
  useEffect(() => {
    if (!mergeTarget) {
      setMergeResult(null);
    }
  }, [mergeTarget]);

  if (loading.branches && branches.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        Loading branches...
      </div>
    );
  }

  if (!loading.branches && branches.length === 0) {
    return (
      <div className="h-full flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Branches</h2>
        </div>
        <EmptyState
          icon={<GitBranch className="w-16 h-16" />}
          title="No branches found yet"
          description={repoPath ? "Create your first commit to initialize the main branch!" : "Open a repository to view its branches."}
          action={repoPath ? { label: "Go to Status", onClick: () => navigate({ to: "/" }) } : undefined}
          className="flex-1"
        />
      </div>
    );
  }

  const localBranches = branches.filter((b) => !b.name.startsWith("remotes/"));
  const remoteBranches = branches.filter((b) => b.name.startsWith("remotes/"));

  const handleCreateBranch = async () => {
    if (!newBranchName.trim() || loading.branches) return;
    await createBranch(newBranchName.trim());
    setNewBranchName("");
    setShowNewBranch(false);
  };

  const handleCheckout = async (name: string) => {
    await checkoutBranch(name);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    await deleteBranch(deleteTarget.name, forceDelete);
    setDeleting(false);
    setDeleteTarget(null);
    setForceDelete(false);
  };

  const handleRename = async () => {
    if (!renameTarget || !renameName.trim() || renameName === renameTarget) return;
    await renameBranch(renameTarget, renameName.trim());
    setRenameTarget(null);
    setRenameName("");
  };

  const openRename = (name: string) => {
    setRenameTarget(name);
    setRenameName(name);
  };

  const handleMerge = async () => {
    if (!mergeTarget) return;
    setMerging(true);
    setMergeResult(null);
    const result = await gitMerge(mergeTarget);
    setMerging(false);
    if (result !== null) {
      setMergeResult(result);
    } else {
      setMergeTarget(null);
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Branches</h2>
          {currentBranch && (
            <Badge variant="outline" className="font-mono text-xs">
              <GitBranch className="w-3 h-3 mr-1" />
              {currentBranch}
            </Badge>
          )}
        </div>
        <Dialog open={showNewBranch} onOpenChange={setShowNewBranch}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="w-4 h-4 mr-1.5" />
              New Branch
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>New Branch</DialogTitle>
              <DialogDescription>
                Create a new branch from <span className="font-mono font-medium">{currentBranch && currentBranch !== "HEAD" ? currentBranch : "HEAD"}</span>
              </DialogDescription>
            </DialogHeader>
            <div className="flex gap-2">
              <input
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                placeholder="Branch name"
                value={newBranchName}
                onChange={(e) => setNewBranchName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreateBranch()}
                autoFocus
              />
              <Button onClick={handleCreateBranch} disabled={!newBranchName.trim() || loading.branches}>
                <CheckCheck className="w-4 h-4 mr-1.5" />
                Create
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Branch lists */}
      <ScrollArea className="h-[calc(100vh-12rem)]">
        {/* Local branches */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-muted-foreground mb-2">
            Local ({localBranches.length})
          </h3>
          <div className="space-y-0.5">
            {localBranches.map((branch) => (
              <BranchRow
                key={branch.name}
                branch={branch}
                isCurrent={branch.is_head}
                onCheckout={() => handleCheckout(branch.name)}
                onDelete={() => setDeleteTarget({ name: branch.name })}
                onMerge={() => setMergeTarget(branch.name)}
                onRename={() => openRename(branch.name)}
              />
            ))}
          </div>
        </div>

        <Separator className="my-4" />

        {/* Remote branches */}
        {remoteBranches.length > 0 && (
          <div>
            <h3 className="text-sm font-medium text-muted-foreground mb-2">
              Remote ({remoteBranches.length})
            </h3>
            <div className="space-y-0.5">
              {remoteBranches.map((branch) => (
                <RemoteBranchRow key={branch.name} name={branch.name} />
              ))}
            </div>
          </div>
        )}
      </ScrollArea>

      {/* ─── Delete Confirmation Dialog ─── */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => { if (!open) { setDeleteTarget(null); setForceDelete(false); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <Trash2 className="w-5 h-5" />
              Delete Branch
            </DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{" "}
              <span className="font-mono font-medium text-foreground">{deleteTarget?.name}</span>?
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center gap-2 p-3 rounded-lg bg-muted/50 text-sm">
              <AlertTriangle className="w-4 h-4 text-destructive shrink-0" />
              <span className="text-muted-foreground">
                {forceDelete
                  ? "Force delete will remove the branch even if it has unmerged changes."
                  : "Safe delete will only remove the branch if it has been fully merged."}
              </span>
            </div>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <Checkbox
                checked={forceDelete}
                onChange={() => setForceDelete(!forceDelete)}
              />
              <span>Force delete unmerged branch (<span className="font-mono">-D</span>)</span>
            </label>
            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={deleting}
              >
                <Trash2 className="w-4 h-4 mr-1.5" />
                {deleting ? "Deleting..." : "Delete"}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* ─── Rename Branch Dialog ─── */}
      <Dialog open={!!renameTarget} onOpenChange={(open) => { if (!open) { setRenameTarget(null); setRenameName(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Pencil className="w-5 h-5" />
              Rename Branch
            </DialogTitle>
            <DialogDescription>
              Rename <span className="font-mono font-medium text-foreground">{renameTarget}</span> to a new name
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <input
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={renameName}
              onChange={(e) => setRenameName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleRename()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              <Button
                onClick={handleRename}
                disabled={!renameName.trim() || renameName === renameTarget || loading.branches}
              >
                <CheckCheck className="w-4 h-4 mr-1.5" />
                Rename
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* ─── Merge Confirmation Dialog ─── */}
      <Dialog open={!!mergeTarget} onOpenChange={(open) => { if (!open) setMergeTarget(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Merge className="w-5 h-5" />
              Merge Branch
            </DialogTitle>
            <DialogDescription>
              Merge <span className="font-mono font-medium text-foreground">{mergeTarget}</span>{" "}
              into <span className="font-mono font-medium text-foreground">{currentBranch}</span>
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {/* Merge info */}
            <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50 text-sm">
              <ArrowRightFromLine className="w-4 h-4 text-primary shrink-0" />
              <div>
                <span className="font-mono">{mergeTarget}</span>
                <span className="text-muted-foreground mx-2">→</span>
                <span className="font-mono">{currentBranch}</span>
              </div>
            </div>

            {/* Merge result display */}
            {mergeResult && (
              <div className="p-3 rounded-lg bg-primary/5 border border-primary/10 text-sm">
                <pre className="whitespace-pre-wrap font-sans text-muted-foreground text-xs">
                  {mergeResult}
                </pre>
              </div>
            )}

            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline">
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              {!mergeResult && (
                <Button onClick={handleMerge} disabled={merging}>
                  <Merge className="w-4 h-4 mr-1.5" />
                  {merging ? "Merging..." : "Merge"}
                </Button>
              )}
              {mergeResult && (
                <DialogClose asChild>
                  <Button variant="default">
                    <CheckCheck className="w-4 h-4 mr-1.5" />
                    Done
                  </Button>
                </DialogClose>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ─── Branch Row ─── */

interface BranchRowProps {
  branch: {
    name: string;
    is_head: boolean;
    upstream?: string;
    ahead?: number;
    behind?: number;
    latest_msg?: string;
  };
  isCurrent: boolean;
  onCheckout: () => void;
  onDelete: () => void;
  onMerge: () => void;
  onRename: () => void;
}

function BranchRow({ branch, isCurrent, onCheckout, onDelete, onMerge, onRename }: BranchRowProps) {
  const ahead = branch.ahead ?? 0;
  const behind = branch.behind ?? 0;

  return (
    <div
      className={cn(
        "group flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors",
        isCurrent
          ? "bg-primary/10 border border-primary/20"
          : "hover:bg-accent/50"
      )}
    >
      {/* Active indicator */}
      <span className={cn(
        "w-3 h-3 rounded-full shrink-0",
        isCurrent ? "bg-primary" : "bg-muted-foreground/30 group-hover:bg-muted-foreground/50"
      )} />

      {/* Branch name */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={cn(
            "font-mono text-sm font-medium truncate",
            isCurrent && "text-primary"
          )}>
            {branch.name}
          </span>
          {isCurrent && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4 font-medium shrink-0">
              Current
            </Badge>
          )}
        </div>
        {/* Latest commit message (subtitle) */}
        {branch.latest_msg && (
          <p className="text-xs text-muted-foreground truncate mt-0.5">
            {branch.latest_msg}
          </p>
        )}
      </div>

      {/* Upstream tracking info */}
      {branch.upstream && (
        <div className="hidden sm:flex items-center gap-1 text-xs text-muted-foreground shrink-0">
          <ArrowUpWideNarrow className="w-3 h-3" />
          <span className="truncate max-w-[120px]">{branch.upstream}</span>
        </div>
      )}

      {/* Ahead/Behind counts */}
      {(ahead > 0 || behind > 0) && (
        <div className="flex items-center gap-1.5 text-xs shrink-0">
          {ahead > 0 && (
            <span className="flex items-center gap-0.5 text-green-500">
              <ArrowUpWideNarrow className="w-3 h-3" />
              {ahead}
            </span>
          )}
          {behind > 0 && (
            <span className="flex items-center gap-0.5 text-red-500">
              <ArrowDownWideNarrow className="w-3 h-3" />
              {behind}
            </span>
          )}
        </div>
      )}

      {/* Actions — always visible for current (no actions), on hover for others */}
      {!isCurrent && (
        <div className="flex gap-1 shrink-0">
          {/* Merge into current */}
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={(e) => { e.stopPropagation(); onMerge(); }}
          >
            <Merge className="w-3.5 h-3.5 mr-1" />
            Merge
          </Button>
          {/* Rename */}
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={(e) => { e.stopPropagation(); onRename(); }}
          >
            <Pencil className="w-3.5 h-3.5 mr-1" />
            Rename
          </Button>
          {/* Checkout */}
          <Button
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={(e) => { e.stopPropagation(); onCheckout(); }}
          >
            <ArrowRightFromLine className="w-3.5 h-3.5 mr-1" />
            Checkout
          </Button>
          {/* Delete */}
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-muted-foreground hover:text-destructive"
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
          >
            <Trash2 className="w-3.5 h-3.5" />
          </Button>
        </div>
      )}
    </div>
  );
}

/* ─── Remote Branch Row ─── */

function RemoteBranchRow({ name }: { name: string }) {
  // Strip the "remotes/" prefix for display
  const displayName = name.startsWith("remotes/") ? name.slice(8) : name;

  return (
    <div className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-accent/50 transition-colors">
      <span className="w-3 h-3 rounded-full bg-muted-foreground/20 shrink-0" />
      <span className="flex-1 font-mono text-sm text-muted-foreground truncate">
        {displayName}
      </span>
      <Badge variant="outline" className="text-[10px] text-muted-foreground shrink-0">
        remote
      </Badge>
    </div>
  );
}
