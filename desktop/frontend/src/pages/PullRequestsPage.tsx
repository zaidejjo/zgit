import { useEffect, useState, useCallback } from "react";
import {
  GitPullRequest, GitPullRequestDraft, GitMerge, GitBranch, Plus, X,
  CheckCheck, MessageSquare, ArrowUpWideNarrow, ArrowDownWideNarrow,
  FileText, SquarePen, ExternalLink, ArrowRight, Check, ListChecks,
  AlertTriangle, LoaderCircle, ChevronDown,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import type { PRSummary, PullRequestDetail } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

export default function PullRequestsPage() {
  const {
    pullRequests, branches, currentBranch, selectedPRDetail,
    loading, error, ghAuthenticated,
    setLoginDialogOpen, fetchPullRequests, fetchPRDetail, clearPRDetail,
    createPullRequest, mergePullRequest, fetchBranches,
  } = useAppStore();

  // Selection
  const [selectedPRNum, setSelectedPRNum] = useState<number | null>(null);

  // Create dialog
  const [showCreate, setShowCreate] = useState(false);
  const [prTitle, setPrTitle] = useState("");
  const [prBody, setPrBody] = useState("");
  const [prHead, setPrHead] = useState(currentBranch);
  const [prBase, setPrBase] = useState("main");
  const [prDraft, setPrDraft] = useState(false);
  const [creating, setCreating] = useState(false);

  // Merge dialog
  const [showMerge, setShowMerge] = useState(false);
  const [mergeMethod, setMergeMethod] = useState("merge");
  const [merging, setMerging] = useState(false);

  useEffect(() => {
    fetchPullRequests();
    // Fetch branches for create dialog branch pickers
    fetchBranches();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Keep head branch in sync with currentBranch
  useEffect(() => {
    setPrHead(currentBranch);
  }, [currentBranch]);

  // Fetch detail when a PR is selected
  useEffect(() => {
    if (selectedPRNum !== null) {
      fetchPRDetail(selectedPRNum);
    } else {
      clearPRDetail();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedPRNum]);

  const handleCreate = async () => {
    if (!prTitle.trim() || !prHead || !prBase) return;
    setCreating(true);
    await createPullRequest(prTitle.trim(), prBody, prHead, prBase, prDraft);
    setCreating(false);
    setShowCreate(false);
    setPrTitle("");
    setPrBody("");
    setPrDraft(false);
  };

  const handleMerge = async () => {
    if (!selectedPRNum) return;
    setMerging(true);
    await mergePullRequest(selectedPRNum, mergeMethod);
    setMerging(false);
    setShowMerge(false);
    setMergeMethod("merge");
  };

  // Extract local branches for the create dialog
  const localBranches = branches
    .filter((b) => !b.name.startsWith("remotes/"))
    .map((b) => b.name);

  // ───── Render ─────

  if (!ghAuthenticated) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-muted-foreground gap-3">
        <GitPullRequest className="w-12 h-12 opacity-40" />
        <p>Sign in to GitHub to manage pull requests</p>
        <Button variant="outline" size="sm" onClick={() => setLoginDialogOpen(true)}>
          Open Settings
        </Button>
      </div>
    );
  }

  if (loading.prs && pullRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <LoaderCircle className="w-5 h-5 mr-2 animate-spin" />
        Loading pull requests...
      </div>
    );
  }

  if (error && pullRequests.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <div className="text-center">
          <AlertTriangle className="w-8 h-8 mx-auto mb-2 text-destructive" />
          <p>Failed to load pull requests</p>
          <p className="text-xs mt-1 text-muted-foreground">{error}</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={fetchPullRequests}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Pull Requests</h2>
          {pullRequests.length > 0 && (
            <Badge variant="outline" className="text-xs">{pullRequests.length}</Badge>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onClick={fetchPullRequests} disabled={loading.prs}>
            <LoaderCircle className={cn("w-4 h-4 mr-1", loading.prs && "animate-spin")} />
            Refresh
          </Button>
          <Dialog open={showCreate} onOpenChange={setShowCreate}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="w-4 h-4 mr-1.5" />
                New PR
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-xl">
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  <GitPullRequest className="w-5 h-5" />
                  New Pull Request
                </DialogTitle>
                <DialogDescription>
                  Open a pull request on GitHub
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                {/* Branch selectors */}
                <div className="flex items-center gap-2">
                  <div className="flex-1 space-y-1">
                    <label className="text-xs text-muted-foreground">Base (target)</label>
                    <Select value={prBase} onValueChange={setPrBase}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {localBranches.map((b) => (
                          <SelectItem key={b} value={b}>{b}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <ArrowRight className="w-4 h-4 mt-5 text-muted-foreground shrink-0" />
                  <div className="flex-1 space-y-1">
                    <label className="text-xs text-muted-foreground">Head (source)</label>
                    <Select value={prHead} onValueChange={setPrHead}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {localBranches.map((b) => (
                          <SelectItem key={b} value={b}>{b}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {/* Title */}
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Title</label>
                  <input
                    className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    placeholder="PR title"
                    value={prTitle}
                    onChange={(e) => setPrTitle(e.target.value)}
                    autoFocus
                  />
                </div>

                {/* Body */}
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Description</label>
                  <textarea
                    className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
                    placeholder="Describe your changes (optional)"
                    rows={4}
                    value={prBody}
                    onChange={(e) => setPrBody(e.target.value)}
                  />
                </div>

                {/* Draft toggle */}
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    className="rounded border-border accent-primary"
                    checked={prDraft}
                    onChange={(e) => setPrDraft(e.target.checked)}
                  />
                  <span>Create as Draft</span>
                  <Badge variant="outline" className="text-[10px] text-muted-foreground">DRAFT</Badge>
                </label>

                <div className="flex justify-end gap-2 pt-2">
                  <DialogClose asChild>
                    <Button variant="outline" disabled={creating}>
                      <X className="w-4 h-4 mr-1.5" />
                      Cancel
                    </Button>
                  </DialogClose>
                  <Button
                    onClick={handleCreate}
                    disabled={!prTitle.trim() || !prHead || !prBase || creating}
                  >
                    {creating ? <LoaderCircle className="w-4 h-4 mr-1.5 animate-spin" /> : <GitPullRequest className="w-4 h-4 mr-1.5" />}
                    {creating ? "Creating..." : "Create PR"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Content: List + Detail */}
      {pullRequests.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
          <GitPullRequest className="w-10 h-10 mb-2 opacity-30" />
          <p className="text-sm">No open pull requests</p>
          <Button variant="link" size="sm" className="mt-1" onClick={() => setShowCreate(true)}>
            Create one
          </Button>
        </div>
      ) : (
        <div className="flex gap-4 h-[calc(100vh-12rem)]">
          {/* PR list */}
          <div className={cn(
            "min-w-0",
            selectedPRNum ? "w-80" : "flex-1"
          )}>
            <ScrollArea className="h-full">
              <div className="space-y-1 pr-2">
                {pullRequests.map((pr) => (
                  <PRListItem
                    key={pr.number}
                    pr={pr}
                    selected={selectedPRNum === pr.number}
                    onClick={() => setSelectedPRNum(selectedPRNum === pr.number ? null : pr.number)}
                  />
                ))}
              </div>
            </ScrollArea>
          </div>

          {/* PR Detail */}
          {selectedPRNum && (
            <div className="flex-1 min-w-0 border rounded-lg">
              {selectedPRDetail && selectedPRDetail.number === selectedPRNum ? (
                <PRDetailPanel
                  detail={selectedPRDetail}
                  loading={!!loading.prDetail}
                  onClose={() => setSelectedPRNum(null)}
                  onOpenMerge={() => setShowMerge(true)}
                  onRefresh={() => fetchPRDetail(selectedPRNum)}
                />
              ) : (
                <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
                  <LoaderCircle className="w-4 h-4 mr-2 animate-spin" />
                  Loading...
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* ─── Merge Dialog ─── */}
      <Dialog open={showMerge} onOpenChange={setShowMerge}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <GitMerge className="w-5 h-5" />
              Merge Pull Request
            </DialogTitle>
            <DialogDescription>
              Merge #{selectedPRNum} — {selectedPRDetail?.title}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {/* Merge method selector */}
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">Merge method</label>
              {(["merge", "squash", "rebase"] as const).map((method) => (
                <label
                  key={method}
                  className={cn(
                    "flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors",
                    mergeMethod === method
                      ? "border-primary bg-primary/5"
                      : "border-border hover:bg-accent/50"
                  )}
                  onClick={() => setMergeMethod(method)}
                >
                  <input
                    type="radio"
                    name="merge-method"
                    className="accent-primary"
                    checked={mergeMethod === method}
                    onChange={() => setMergeMethod(method)}
                  />
                  <div className="flex-1">
                    <p className="text-sm font-medium capitalize">{method} merge</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {method === "merge" && "Creates a merge commit"}
                      {method === "squash" && "Squashes all commits into one"}
                      {method === "rebase" && "Rebases commits onto base branch"}
                    </p>
                  </div>
                </label>
              ))}
            </div>

            <div className="flex justify-end gap-2">
              <DialogClose asChild>
                <Button variant="outline" disabled={merging}>
                  <X className="w-4 h-4 mr-1.5" />
                  Cancel
                </Button>
              </DialogClose>
              <Button onClick={handleMerge} disabled={merging}>
                {merging ? (
                  <LoaderCircle className="w-4 h-4 mr-1.5 animate-spin" />
                ) : (
                  <GitMerge className="w-4 h-4 mr-1.5" />
                )}
                {merging ? "Merging..." : `Merge (${mergeMethod})`}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ─── PR List Item ─── */

function PRListItem({ pr, selected, onClick }: { pr: PRSummary; selected: boolean; onClick: () => void }) {
  const stateColor = {
    OPEN: pr.is_draft ? "text-yellow-500" : "text-green-500",
    MERGED: "text-purple-500",
    CLOSED: "text-red-500",
  }[pr.state] || "text-muted-foreground";

  const Icon = pr.is_draft ? GitPullRequestDraft : GitPullRequest;

  return (
    <div
      className={cn(
        "flex items-start gap-2.5 px-3 py-2.5 rounded-lg cursor-pointer transition-colors",
        selected
          ? "bg-accent border border-border"
          : "hover:bg-accent/50"
      )}
      onClick={onClick}
    >
      <Icon className={cn("w-4 h-4 mt-0.5 shrink-0", stateColor)} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          <span className="text-muted-foreground font-mono text-xs mr-1">#{pr.number}</span>
          {pr.title}
        </p>
        <p className="text-xs text-muted-foreground mt-0.5 flex items-center gap-1">
          <span className="truncate">{pr.head_ref}</span>
          <ArrowRight className="w-2.5 h-2.5 shrink-0" />
          <span className="truncate">{pr.base_ref}</span>
        </p>
      </div>
      <div className="shrink-0 flex gap-1 items-center">
        {/* Review state indicator */}
        {pr.review_state && (
          <Badge variant="outline" className={cn(
            "text-[10px] px-1",
            pr.review_state === "APPROVED" && "text-green-500 border-green-500/30",
            pr.review_state === "CHANGES_REQUESTED" && "text-red-500 border-red-500/30",
            pr.review_state === "REVIEW_REQUIRED" && "text-yellow-500 border-yellow-500/30",
          )}>
            {pr.review_state === "APPROVED" ? "✓" : pr.review_state === "CHANGES_REQUESTED" ? "✗" : "?"}
          </Badge>
        )}
        {(pr.labels || []).slice(0, 2).map((label) => (
          <Badge key={label} variant="secondary" className="text-[10px] px-1 max-w-[60px] truncate">
            {label}
          </Badge>
        ))}
      </div>
    </div>
  );
}

/* ─── PR Detail Panel ─── */

function PRDetailPanel({
  detail, loading, onClose, onOpenMerge, onRefresh,
}: {
  detail: PullRequestDetail;
  loading: boolean;
  onClose: () => void;
  onOpenMerge: () => void;
  onRefresh: () => void;
}) {
  const canMerge = detail.state === "OPEN" && detail.mergeable === "MERGEABLE";
  const stateColor = {
    OPEN: detail.is_draft ? "text-yellow-500" : "text-green-500",
    MERGED: "text-purple-500",
    CLOSED: "text-red-500",
  }[detail.state] || "";

  return (
    <div className="flex flex-col h-full">
      {/* Detail header */}
      <div className="flex items-center justify-between px-4 py-3 border-b">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-semibold truncate">
            <span className="text-muted-foreground font-mono mr-1">#{detail.number}</span>
            {detail.title}
          </h3>
          <span className={cn("text-xs font-medium", stateColor)}>
            {detail.is_draft ? "DRAFT" : detail.state}
          </span>
        </div>
        <div className="flex gap-1 shrink-0">
          {canMerge && (
            <>
              <Button size="sm" className="h-7 text-xs" onClick={onOpenMerge}>
                <GitMerge className="w-3.5 h-3.5 mr-1" />
                Merge
              </Button>
            </>
          )}
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onRefresh} disabled={loading}>
            <LoaderCircle className={cn("w-3.5 h-3.5", loading && "animate-spin")} />
          </Button>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onClose}>
            <X className="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>

      {/* Scrollable body */}
      <ScrollArea className="flex-1">
        <div className="p-4 space-y-4">
          {/* Meta row */}
          <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
            <span>by <strong>{detail.author}</strong></span>
            <span>{new Date(detail.created_at).toLocaleDateString()}</span>
            <span className={cn(
              "font-mono",
              detail.additions > 0 && "text-green-500",
            )}>+{detail.additions}</span>
            <span className={cn(
              "font-mono",
              detail.deletions > 0 && "text-red-500",
            )}>-{detail.deletions}</span>
            <span>{detail.changed_files} files</span>
            <span>
              <MessageSquare className="w-3 h-3 inline mr-0.5" />
              {detail.comments}
            </span>
            {detail.merged_at && (
              <span className="text-purple-500">Merged by {detail.merged_by}</span>
            )}
          </div>

          {/* Branch info */}
          <div className="flex items-center gap-2 p-2 rounded bg-muted/30 text-xs font-mono">
            <GitBranch className="w-3 h-3 text-muted-foreground" />
            <span>{detail.head_ref}</span>
            <ArrowRight className="w-3 h-3 text-muted-foreground" />
            <span>{detail.base_ref}</span>
            {detail.mergeable && (
              <Badge variant="outline" className={cn(
                "text-[10px] ml-auto",
                detail.mergeable === "MERGEABLE" && "text-green-500",
                detail.mergeable === "CONFLICTING" && "text-red-500",
              )}>
                {detail.mergeable}
              </Badge>
            )}
          </div>

          {/* Body */}
          {detail.body && (
            <div className="text-sm leading-relaxed whitespace-pre-wrap text-muted-foreground bg-muted/20 rounded-lg p-3">
              {detail.body}
            </div>
          )}

          {/* Check runs */}
          {detail.check_runs && detail.check_runs.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
                Checks ({detail.check_runs.length})
              </h4>
              <div className="space-y-1">
                {detail.check_runs.map((cr, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs px-2 py-1 rounded hover:bg-accent/30">
                    <span className={cn(
                      "w-2 h-2 rounded-full shrink-0",
                      cr.conclusion === "SUCCESS" && "bg-green-500",
                      cr.conclusion === "FAILURE" && "bg-red-500",
                      cr.conclusion === "CANCELLED" && "bg-gray-500",
                      (cr.conclusion === "" || cr.conclusion === "null" || !cr.conclusion) && "bg-yellow-500",
                    )} />
                    <span className="truncate">{cr.name}</span>
                    <span className="text-muted-foreground ml-auto">
                      {cr.conclusion || cr.state}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Reviews */}
          {detail.reviews && detail.reviews.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
                Reviews ({detail.reviews.length})
              </h4>
              <div className="space-y-2">
                {detail.reviews.map((r) => (
                  <div key={r.id} className="text-sm p-3 rounded-lg border">
                    <div className="flex items-center gap-2 mb-1">
                      <span className={cn(
                        "text-xs font-medium",
                        r.state === "APPROVED" && "text-green-500",
                        r.state === "CHANGES_REQUESTED" && "text-red-500",
                        r.state === "COMMENTED" && "text-muted-foreground",
                      )}>
                        {r.state === "APPROVED" && "✓"}
                        {r.state === "CHANGES_REQUESTED" && "✗"}
                        {r.state === "COMMENTED" && "💬"}
                      </span>
                      <span className="font-medium text-xs">{r.author}</span>
                      <span className="text-xs text-muted-foreground">{r.state.replace("_", " ")}</span>
                      <span className="text-xs text-muted-foreground ml-auto">
                        {new Date(r.submitted_at).toLocaleDateString()}
                      </span>
                    </div>
                    {r.body && (
                      <p className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap">{r.body}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Commits */}
          {detail.commits && detail.commits.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
                Commits ({detail.commits.length})
              </h4>
              <div className="space-y-1">
                {detail.commits.slice(0, 20).map((c, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs px-2 py-1 rounded hover:bg-accent/30">
                    <span className="font-mono text-muted-foreground w-16 shrink-0">{c.hash.slice(0, 7)}</span>
                    <span className="truncate">{c.message}</span>
                    <span className="text-muted-foreground ml-auto shrink-0">{c.author}</span>
                  </div>
                ))}
                {detail.commits.length > 20 && (
                  <p className="text-xs text-muted-foreground text-center py-1">
                    ...and {detail.commits.length - 20} more commits
                  </p>
                )}
              </div>
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
