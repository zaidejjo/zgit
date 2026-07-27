import { useEffect, useState } from "react";
import {
  CircleDot, Plus, X, CheckCheck, LoaderCircle,
  AlertTriangle, MessageSquare, ArrowUpWideNarrow,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import type { Issue } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

export default function IssuesPage() {
  const {
    issues, selectedIssueDetail, loading, error, ghAuthenticated,
    setLoginDialogOpen, fetchIssues, fetchIssueDetail, clearIssueDetail,
    createIssue, closeIssue,
  } = useAppStore();

  // Selection
  const [selectedNum, setSelectedNum] = useState<number | null>(null);

  // Create dialog
  const [showCreate, setShowCreate] = useState(false);
  const [issueTitle, setIssueTitle] = useState("");
  const [issueBody, setIssueBody] = useState("");
  const [creating, setCreating] = useState(false);

  // Close dialog
  const [showClose, setShowClose] = useState(false);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    fetchIssues();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Fetch detail when issue selected
  useEffect(() => {
    if (selectedNum !== null) {
      fetchIssueDetail(selectedNum);
    } else {
      clearIssueDetail();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedNum]);

  const handleCreate = async () => {
    if (!issueTitle.trim()) return;
    setCreating(true);
    await createIssue(issueTitle.trim(), issueBody);
    setCreating(false);
    setShowCreate(false);
    setIssueTitle("");
    setIssueBody("");
  };

  const handleClose = async () => {
    if (!selectedNum) return;
    setClosing(true);
    await closeIssue(selectedNum);
    setClosing(false);
    setShowClose(false);
    setSelectedNum(null);
  };

  // ───── Render ─────

  if (!ghAuthenticated) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-muted-foreground gap-3">
        <CircleDot className="w-12 h-12 opacity-40" />
        <p>Sign in to GitHub to manage issues</p>
        <Button variant="outline" size="sm" onClick={() => setLoginDialogOpen(true)}>
          Open Settings
        </Button>
      </div>
    );
  }

  if (loading.issues && issues.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <LoaderCircle className="w-5 h-5 mr-2 animate-spin" />
        Loading issues...
      </div>
    );
  }

  if (error && issues.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <div className="text-center">
          <AlertTriangle className="w-8 h-8 mx-auto mb-2 text-destructive" />
          <p>Failed to load issues</p>
          <p className="text-xs mt-1">{error}</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={fetchIssues}>Retry</Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Issues</h2>
          {issues.length > 0 && (
            <Badge variant="outline" className="text-xs">{issues.length}</Badge>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onClick={fetchIssues} disabled={loading.issues}>
            <LoaderCircle className={cn("w-4 h-4 mr-1", loading.issues && "animate-spin")} />
            Refresh
          </Button>
          <Dialog open={showCreate} onOpenChange={setShowCreate}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="w-4 h-4 mr-1.5" />
                New Issue
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-xl">
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  <CircleDot className="w-5 h-5" />
                  New Issue
                </DialogTitle>
                <DialogDescription>
                  Open a new issue on GitHub
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Title</label>
                  <input
                    className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    placeholder="Issue title"
                    value={issueTitle}
                    onChange={(e) => setIssueTitle(e.target.value)}
                    autoFocus
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground mb-1 block">Description</label>
                  <textarea
                    className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
                    placeholder="Describe the issue (optional)"
                    rows={5}
                    value={issueBody}
                    onChange={(e) => setIssueBody(e.target.value)}
                  />
                </div>
                <div className="flex justify-end gap-2">
                  <DialogClose asChild>
                    <Button variant="outline" disabled={creating}>
                      <X className="w-4 h-4 mr-1.5" />
                      Cancel
                    </Button>
                  </DialogClose>
                  <Button
                    onClick={handleCreate}
                    disabled={!issueTitle.trim() || creating}
                  >
                    {creating ? (
                      <LoaderCircle className="w-4 h-4 mr-1.5 animate-spin" />
                    ) : (
                      <CircleDot className="w-4 h-4 mr-1.5" />
                    )}
                    {creating ? "Creating..." : "Create Issue"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Content: List + Detail */}
      {issues.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
          <CircleDot className="w-10 h-10 mb-2 opacity-30" />
          <p className="text-sm">No open issues</p>
          <Button variant="link" size="sm" className="mt-1" onClick={() => setShowCreate(true)}>
            Create one
          </Button>
        </div>
      ) : (
        <div className="flex gap-4 h-[calc(100vh-12rem)]">
          {/* Issue list */}
          <div className={cn("min-w-0", selectedNum ? "w-80" : "flex-1")}>
            <ScrollArea className="h-full">
              <div className="space-y-1 pr-2">
                {issues.map((issue) => (
                  <IssueListItem
                    key={issue.number}
                    issue={issue}
                    selected={selectedNum === issue.number}
                    onClick={() => setSelectedNum(selectedNum === issue.number ? null : issue.number)}
                  />
                ))}
              </div>
            </ScrollArea>
          </div>

          {/* Issue detail */}
          {selectedNum && (
            <div className="flex-1 min-w-0 border rounded-lg">
              {selectedIssueDetail && selectedIssueDetail.number === selectedNum ? (
                <IssueDetailPanel
                  issue={selectedIssueDetail}
                  loading={!!loading.issueDetail}
                  onClose={() => setSelectedNum(null)}
                  onCloseIssue={() => setShowClose(true)}
                  onRefresh={() => fetchIssueDetail(selectedNum)}
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

      {/* ─── Close Issue Dialog ─── */}
      <Dialog open={showClose} onOpenChange={setShowClose}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CircleDot className="w-5 h-5" />
              Close Issue
            </DialogTitle>
            <DialogDescription>
              Close <span className="font-mono font-medium text-foreground">#{selectedNum}</span>
              {selectedIssueDetail && <span> — {selectedIssueDetail.title}</span>}
            </DialogDescription>
          </DialogHeader>
          <div className="flex gap-2 justify-end">
            <DialogClose asChild>
              <Button variant="outline" disabled={closing}>
                <X className="w-4 h-4 mr-1.5" />
                Cancel
              </Button>
            </DialogClose>
            <Button variant="destructive" onClick={handleClose} disabled={closing}>
              {closing ? (
                <LoaderCircle className="w-4 h-4 mr-1.5 animate-spin" />
              ) : (
                <X className="w-4 h-4 mr-1.5" />
              )}
              {closing ? "Closing..." : "Close Issue"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ─── Issue List Item ─── */

function IssueListItem({ issue, selected, onClick }: { issue: Issue; selected: boolean; onClick: () => void }) {
  return (
    <div
      className={cn(
        "flex items-start gap-2.5 px-3 py-2.5 rounded-lg cursor-pointer transition-colors",
        selected ? "bg-accent border border-border" : "hover:bg-accent/50"
      )}
      onClick={onClick}
    >
      <CircleDot className={cn(
        "w-4 h-4 mt-0.5 shrink-0",
        issue.state === "OPEN" ? "text-green-500" : "text-red-500"
      )} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          <span className="text-muted-foreground font-mono text-xs mr-1">#{issue.number}</span>
          {issue.title}
        </p>
        <p className="text-xs text-muted-foreground mt-0.5">
          {issue.author}
          {issue.comments > 0 && (
            <>
              <span className="mx-1">&middot;</span>
              <MessageSquare className="w-3 h-3 inline mr-0.5" />
              {issue.comments}
            </>
          )}
        </p>
      </div>
      <div className="shrink-0 flex gap-1 items-center flex-wrap max-w-[120px]">
        {(issue.labels || []).slice(0, 3).map((label) => (
          <Badge
            key={label.name}
            variant="secondary"
            className="text-[10px] px-1 truncate max-w-[80px]"
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
  );
}

/* ─── Issue Detail Panel ─── */

function IssueDetailPanel({
  issue, loading, onClose, onCloseIssue, onRefresh,
}: {
  issue: Issue;
  loading: boolean;
  onClose: () => void;
  onCloseIssue: () => void;
  onRefresh: () => void;
}) {
  const isOpen = issue.state === "OPEN";

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-semibold truncate">
            <span className="text-muted-foreground font-mono mr-1">#{issue.number}</span>
            {issue.title}
          </h3>
          <Badge variant="outline" className={cn(
            "text-xs",
            isOpen ? "text-green-500 border-green-500/30" : "text-red-500 border-red-500/30"
          )}>
            {issue.state}
          </Badge>
        </div>
        <div className="flex gap-1 shrink-0">
          {isOpen && (
            <Button size="sm" className="h-7 text-xs" variant="destructive" onClick={onCloseIssue}>
              <X className="w-3.5 h-3.5 mr-1" />
              Close
            </Button>
          )}
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onRefresh} disabled={loading}>
            <LoaderCircle className={cn("w-3.5 h-3.5", loading && "animate-spin")} />
          </Button>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onClose}>
            <X className="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>

      {/* Body */}
      <ScrollArea className="flex-1">
        <div className="p-4 space-y-4">
          {/* Meta */}
          <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
            <span>by <strong>{issue.author}</strong></span>
            <span>opened {new Date(issue.created_at).toLocaleDateString()}</span>
            {issue.updated_at !== issue.created_at && (
              <span>updated {new Date(issue.updated_at).toLocaleDateString()}</span>
            )}
            {issue.closed_at && (
              <span>closed {new Date(issue.closed_at).toLocaleDateString()}</span>
            )}
            <span>
              <MessageSquare className="w-3 h-3 inline mr-0.5" />
              {issue.comments} comments
            </span>
          </div>

          {/* Labels */}
          {issue.labels && issue.labels.length > 0 && (
            <div className="flex gap-1 flex-wrap">
              {issue.labels.map((label) => (
                <Badge
                  key={label.name}
                  variant="secondary"
                  style={label.color ? {
                    borderColor: `#${label.color}`,
                    backgroundColor: `#${label.color}20`,
                  } : undefined}
                >
                  {label.name}
                </Badge>
              ))}
            </div>
          )}

          {/* Assignees */}
          {issue.assignees && issue.assignees.length > 0 && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>Assignees:</span>
              {issue.assignees.map((a) => (
                <Badge key={a} variant="outline" className="text-xs">{a}</Badge>
              ))}
            </div>
          )}

          {/* Body */}
          {issue.body && (
            <div className="text-sm leading-relaxed whitespace-pre-wrap text-muted-foreground bg-muted/20 rounded-lg p-3">
              {issue.body}
            </div>
          )}

          {!issue.body && (
            <p className="text-sm text-muted-foreground italic">No description provided.</p>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
