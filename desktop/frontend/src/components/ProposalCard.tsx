import { useState } from "react";
import {
  Check, X, GitBranch, GitCommitHorizontal, ArrowUpFromLine,
  ArrowDownToLine, Trash2, Pencil, Merge, RotateCcw,
  Tags, FileX, Files, StickyNote,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AgentActionProposal } from "@/store/app";

interface ProposalCardProps {
  proposal: AgentActionProposal;
  onApprove: (id: string) => void;
  onReject: (id: string, feedback?: string) => void;
  disabled?: boolean;
}

const actionMeta: Record<string, { icon: React.ReactNode; color: string; label: string }> = {
  create_branch:              { icon: <GitBranch className="w-3.5 h-3.5" />, color: "text-blue-500", label: "Create Branch" },
  checkout:                   { icon: <ArrowUpFromLine className="w-3.5 h-3.5" />, color: "text-cyan-500", label: "Checkout" },
  create_branch_and_checkout: { icon: <GitBranch className="w-3.5 h-3.5" />, color: "text-blue-500", label: "Branch & Switch" },
  commit:                     { icon: <GitCommitHorizontal className="w-3.5 h-3.5" />, color: "text-green-500", label: "Commit" },
  push:                       { icon: <ArrowUpFromLine className="w-3.5 h-3.5" />, color: "text-orange-500", label: "Push" },
  pull:                       { icon: <ArrowDownToLine className="w-3.5 h-3.5" />, color: "text-purple-500", label: "Pull" },
  stash_push:                 { icon: <StickyNote className="w-3.5 h-3.5" />, color: "text-yellow-500", label: "Stash" },
  stash_pop:                  { icon: <RotateCcw className="w-3.5 h-3.5" />, color: "text-yellow-500", label: "Pop Stash" },
  reset_commit:               { icon: <RotateCcw className="w-3.5 h-3.5" />, color: "text-red-500", label: "Reset" },
  revert_commit:              { icon: <RotateCcw className="w-3.5 h-3.5" />, color: "text-red-500", label: "Revert" },
  merge:                      { icon: <Merge className="w-3.5 h-3.5" />, color: "text-violet-500", label: "Merge" },
  delete_branch:              { icon: <Trash2 className="w-3.5 h-3.5" />, color: "text-red-500", label: "Delete Branch" },
  stage_all:                  { icon: <Files className="w-3.5 h-3.5" />, color: "text-green-500", label: "Stage All" },
  unstage_all:                { icon: <Files className="w-3.5 h-3.5" />, color: "text-muted-foreground", label: "Unstage All" },
  discard_file:               { icon: <FileX className="w-3.5 h-3.5" />, color: "text-red-500", label: "Discard File" },
  discard_all:                { icon: <FileX className="w-3.5 h-3.5" />, color: "text-red-500", label: "Discard All" },
  tag_create:                 { icon: <Tags className="w-3.5 h-3.5" />, color: "text-emerald-500", label: "Create Tag" },
  tag_delete:                 { icon: <Trash2 className="w-3.5 h-3.5" />, color: "text-red-500", label: "Delete Tag" },
  conflict_resolve:           { icon: <Pencil className="w-3.5 h-3.5" />, color: "text-amber-500", label: "Resolve Conflict" },
};

export default function ProposalCard({ proposal, onApprove, onReject, disabled }: ProposalCardProps) {
  const [feedback, setFeedback] = useState("");
  const [showFeedback, setShowFeedback] = useState(false);
  const [approving, setApproving] = useState(false);
  const [rejecting, setRejecting] = useState(false);

  const meta = actionMeta[proposal.type] || { icon: <Pencil className="w-3.5 h-3.5" />, color: "text-muted-foreground", label: proposal.type };

  const isDone = proposal.status === "executed" || proposal.status === "failed";

  const handleApprove = async () => {
    setApproving(true);
    await onApprove(proposal.id);
    setApproving(false);
  };

  const handleReject = async () => {
    setRejecting(true);
    await onReject(proposal.id, showFeedback ? feedback : undefined);
    setRejecting(false);
    setFeedback("");
    setShowFeedback(false);
  };

  if (isDone) {
    const succeeded = proposal.status === "executed";
    return (
      <div className={cn(
        "rounded-md border px-3 py-2 text-xs flex items-center gap-2",
        succeeded ? "border-green-500/30 bg-green-500/5" : "border-red-500/30 bg-red-500/5"
      )}>
        <span>{succeeded ? "✅" : "❌"}</span>
        <span className="text-muted-foreground flex-1">{proposal.description}</span>
        <span className={cn("font-medium", succeeded ? "text-green-500" : "text-red-500")}>
          {succeeded ? "Done" : "Failed"}
        </span>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-border/60 bg-card shadow-sm">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border/30">
        <span className={cn("flex items-center gap-1 text-xs font-medium", meta.color)}>
          {meta.icon}
          {meta.label}
        </span>
        <span className="text-[10px] text-muted-foreground/60 ml-auto uppercase">
          Proposed
        </span>
      </div>

      {/* Body */}
      <div className="px-3 py-2 space-y-1.5">
        <p className="text-sm font-medium leading-snug">{proposal.description}</p>
        <p className="text-xs text-muted-foreground leading-relaxed">{proposal.reasoning}</p>
        {proposal.diff_preview && (
          <pre className="text-[11px] font-mono bg-muted/30 rounded p-2 overflow-x-auto border border-border/20 mt-1 text-muted-foreground/80">
            {proposal.diff_preview}
          </pre>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1.5 px-3 py-2 border-t border-border/30">
        <Button
          size="sm"
          variant="default"
          className="h-7 text-xs gap-1"
          onClick={handleApprove}
          disabled={disabled || approving}
        >
          <Check className="w-3.5 h-3.5" />
          {approving ? "Running..." : "Approve & Run"}
        </Button>

        {showFeedback ? (
          <div className="flex items-center gap-1 flex-1">
            <input
              type="text"
              className="flex-1 h-7 px-2 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring"
              placeholder="Feedback for agent..."
              value={feedback}
              onChange={(e) => setFeedback(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleReject()}
              autoFocus
            />
            <Button
              size="sm"
              variant="outline"
              className="h-7 text-xs gap-1"
              onClick={handleReject}
              disabled={disabled || rejecting}
            >
              {rejecting ? "..." : "Send"}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-7 w-7 p-0"
              onClick={() => { setShowFeedback(false); setFeedback(""); }}
            >
              <X className="w-3 h-3" />
            </Button>
          </div>
        ) : (
          <>
            <Button
              size="sm"
              variant="outline"
              className="h-7 text-xs gap-1 text-muted-foreground"
              onClick={() => setShowFeedback(true)}
              disabled={disabled}
            >
              <X className="w-3.5 h-3.5" />
              Reject
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
