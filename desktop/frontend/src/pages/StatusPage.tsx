import { useEffect, useState, useCallback } from "react";
import {
  Download, Upload, Undo2, Archive, RotateCcw, Play, X,
  GitCommitHorizontal, SquarePen, AlignLeft, CheckCheck,
  AlertTriangle, ChevronDown, FileText,
  RefreshCw, AlertOctagon, ArrowRight, Sparkles,
  GitFork, Eye, EyeOff, Plus, Trash2,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Checkbox } from "@/components/ui/checkbox";
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip";
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import DiffViewer from "@/components/DiffViewer";
import MergeEditor from "@/components/MergeEditor";

const CONVENTIONAL_PREFIXES = [
  { value: "feat", label: "feat", desc: "New feature", color: "text-success" },
  { value: "fix", label: "fix", desc: "Bug fix", color: "text-destructive" },
  { value: "docs", label: "docs", desc: "Documentation", color: "text-primary" },
  { value: "style", label: "style", desc: "Formatting", color: "text-[hsl(var(--pr-merged))]" },
  { value: "refactor", label: "refactor", desc: "Code restructure", color: "text-warning" },
  { value: "test", label: "test", desc: "Tests", color: "text-warning" },
  { value: "chore", label: "chore", desc: "Maintenance", color: "text-muted-foreground" },
] as const;

// StatusType enum values from Go models
const STATUS_UNTRACKED = 0;
const STATUS_UNMODIFIED = 7;
const STATUS_UNMERGED = 6;

export default function StatusPage() {
  const {
    status, diff, loading, fetchStatus, fetchDiff,
    stageFile, unstageFile, stageAll, unstageAll, clearDiff,
    commitAndPush, commit, discardFile,
    stashes, fetchStashes, stashPush, stashPop, stashApply, stashDrop,
    gitFetch, gitPull, gitPushForce,
    resolveConflict, openMergeEditor,
    aiConfig, aiGenerating, generateCommitMessage, fetchAIConfig,
  } = useAppStore();

  // Diff selection
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [diffVisible, setDiffVisible] = useState(false);

  // Commit form state
  const [summary, setSummary] = useState("");
  const [description, setDescription] = useState("");
  const [committing, setCommitting] = useState(false);

  // Stash state
  const [stashMsg, setStashMsg] = useState("");
  const [stashing, setStashing] = useState(false);

  // Multi-select staging: set of file paths checked
  const [checkedFiles, setCheckedFiles] = useState<Set<string>>(new Set());

  useEffect(() => {
    fetchStatus();
    fetchStashes();
    fetchAIConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reset checkbox selections when status refreshes
  useEffect(() => {
    setCheckedFiles(new Set());
  }, [status?.files]);

  const handleFileClick = (path: string) => {
    setSelectedFile(path);
    setDiffVisible(true);
    fetchDiff(path);
  };

  const handleCloseDiff = () => {
    setSelectedFile(null);
    setDiffVisible(false);
    clearDiff();
  };

  const toggleCheck = (path: string) => {
    setCheckedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const handleStageChecked = useCallback(async () => {
    const files = Array.from(checkedFiles);
    if (files.length === 0) return;
    for (const f of files) {
      await stageFile(f);
    }
  }, [checkedFiles, stageFile]);

  const handleUnstageChecked = useCallback(async () => {
    const files = Array.from(checkedFiles);
    if (files.length === 0) return;
    for (const f of files) {
      await unstageFile(f);
    }
  }, [checkedFiles, unstageFile]);

  const handleCommitClick = async (withPush: boolean) => {
    if (!summary.trim()) return;
    setCommitting(true);
    const fn = withPush ? commitAndPush : commit;
    await fn(summary.trim(), description);
    setCommitting(false);
    setSummary("");
    setDescription("");
  };

  const handleForcePushThenPush = async () => {
    try {
      await gitPushForce();
    } catch {
      // Error handled in store
    }
  };

  const handleStashPush = async () => {
    setStashing(true);
    await stashPush(stashMsg);
    setStashing(false);
    setStashMsg("");
  };

  // Determine which view to show: if no diff, full width files; if diff, split layout
  const showDiffPanel = selectedFile && diff && diffVisible;

  if (!status) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground text-sm">
        {loading.status ? "Loading status..." : "No repository open"}
      </div>
    );
  }

  const stagedFiles = (status.files || []).filter(
    (f) => f.staged !== STATUS_UNMODIFIED && f.staged !== STATUS_UNTRACKED
  );
  const unstagedFiles = (status.files || []).filter(
    (f) => f.unstaged !== STATUS_UNMODIFIED
  );
  const untrackedFiles = (status.files || []).filter(
    (f) => f.staged === STATUS_UNTRACKED
  );
  const totalChanges = stagedFiles.length + unstagedFiles.length + untrackedFiles.length;

  const hasChecked = checkedFiles.size > 0;
  const checkedStaged = Array.from(checkedFiles).filter((p) =>
    stagedFiles.some((f) => f.path === p)
  );
  const checkedUnstaged = Array.from(checkedFiles).filter((p) =>
    unstagedFiles.some((f) => f.path === p) || untrackedFiles.some((f) => f.path === p)
  );

  const canCommit = stagedFiles.length > 0 && summary.trim().length > 0;

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex gap-5 h-full">
        {/* ─── Left: File List ─── */}
        <div className={cn(
          "flex-1 min-w-0 flex flex-col",
          showDiffPanel ? "w-1/2" : "w-full"
        )}>
          {/* File list header */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2.5">
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground font-medium">
                  {totalChanges} change{totalChanges !== 1 ? "s" : ""}
                </span>
                {status.branch && (
                  <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-md bg-primary/5 border border-primary/10">
                    <GitFork className="w-3 h-3 text-primary" />
                    <span className="text-[11px] font-mono text-primary font-medium">{status.branch}</span>
                  </div>
                )}
                {(status.ahead > 0 || status.behind > 0) && (
                  <span className="text-[11px] text-muted-foreground font-mono">
                    {status.ahead > 0 && <span className="text-success">+{status.ahead}</span>}
                    {status.ahead > 0 && status.behind > 0 && " "}
                    {status.behind > 0 && <span className="text-destructive">-{status.behind}</span>}
                  </span>
                )}
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => gitFetch()}
                disabled={loading.fetch}
                className="px-2.5 py-1.5 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale disabled:opacity-40"
              >
                <Download className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => gitPull(false)}
                disabled={loading.pull}
                className="px-2.5 py-1.5 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale disabled:opacity-40"
              >
                <Upload className="w-3.5 h-3.5" />
              </button>
              <div className="w-px h-4 bg-border/50 mx-1" />
              {hasChecked ? (
                <>
                  {checkedStaged.length > 0 && (
                    <button
                      onClick={handleUnstageChecked}
                      className="px-2.5 py-1.5 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale"
                    >
                      Unstage ({checkedStaged.length})
                    </button>
                  )}
                  {checkedUnstaged.length > 0 && (
                    <button
                      onClick={handleStageChecked}
                      className="px-2.5 py-1.5 text-xs rounded-md text-success hover:text-success hover:bg-success/10 transition-colors press-scale"
                    >
                      Stage ({checkedUnstaged.length})
                    </button>
                  )}
                </>
              ) : (
                <>
                  <button
                    onClick={() => stageAll()}
                    className="px-2.5 py-1.5 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale"
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => unstageAll()}
                    className="px-2.5 py-1.5 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors press-scale"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </>
              )}
            </div>
          </div>

          {/* Merge conflict resolution */}
          {status.is_merging && (
            <MergeConflictBanner
              files={status.files || []}
              onResolve={(file, side) => resolveConflict(file, side)}
              onOpenEditor={(file) => openMergeEditor(file)}
            />
          )}

          {/* File list scroll area */}
          <ScrollArea className="flex-1 min-h-0 -mx-5 px-5">
            {status.is_clean && (
              <div className="flex flex-col items-center justify-center py-20 text-center">
                <div className="w-14 h-14 rounded-xl bg-primary/5 border border-primary/10 flex items-center justify-center mb-4">
                  <CheckCheck className="w-7 h-7 text-primary/30" />
                </div>
                <h3 className="text-sm font-semibold text-foreground mb-1">Clean working tree</h3>
                <p className="text-xs text-muted-foreground max-w-xs">
                  No modified or untracked files.
                </p>
              </div>
            )}

            {!status.is_clean && (
              <div className="space-y-5 pb-4">
                {/* Staged */}
                {stagedFiles.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="status-dot bg-success" />
                      <span className="text-[11px] font-medium text-success uppercase tracking-wider">
                        Staged
                      </span>
                      <span className="text-[11px] text-muted-foreground font-mono">
                        {stagedFiles.length}
                      </span>
                    </div>
                    <div className="space-y-0.5">
                      {stagedFiles.map((f) => (
                        <FileRow
                          key={f.path}
                          file={f}
                          type="staged"
                          selected={selectedFile === f.path}
                          checked={checkedFiles.has(f.path)}
                          onToggleCheck={() => toggleCheck(f.path)}
                          onClick={() => handleFileClick(f.path)}
                          onStage={() => unstageFile(f.path)}
                        />
                      ))}
                    </div>
                  </div>
                )}

                {/* Unstaged */}
                {unstagedFiles.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="status-dot bg-warning" />
                      <span className="text-[11px] font-medium text-warning uppercase tracking-wider">
                        Unstaged
                      </span>
                      <span className="text-[11px] text-muted-foreground font-mono">
                        {unstagedFiles.length}
                      </span>
                    </div>
                    <div className="space-y-0.5">
                      {unstagedFiles.map((f) => (
                        <FileRow
                          key={f.path}
                          file={f}
                          type="unstaged"
                          selected={selectedFile === f.path}
                          checked={checkedFiles.has(f.path)}
                          onToggleCheck={() => toggleCheck(f.path)}
                          onClick={() => handleFileClick(f.path)}
                          onStage={() => stageFile(f.path)}
                          onDiscard={() => discardFile(f.path)}
                        />
                      ))}
                    </div>
                  </div>
                )}

                {/* Untracked */}
                {untrackedFiles.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="status-dot bg-muted-foreground/50" />
                      <span className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
                        Untracked
                      </span>
                      <span className="text-[11px] text-muted-foreground font-mono">
                        {untrackedFiles.length}
                      </span>
                    </div>
                    <div className="space-y-0.5">
                      {untrackedFiles.map((f) => (
                        <FileRow
                          key={f.path}
                          file={f}
                          type="untracked"
                          selected={selectedFile === f.path}
                          checked={checkedFiles.has(f.path)}
                          onToggleCheck={() => toggleCheck(f.path)}
                          onClick={() => handleFileClick(f.path)}
                          onStage={() => stageFile(f.path)}
                          onDiscard={() => discardFile(f.path)}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
        </div>

        {/* ─── Right: Commit Panel + Diff ─── */}
        <div className={cn(
          "flex flex-col gap-4",
          showDiffPanel ? "w-[30rem]" : "w-[26rem]"
        )}>
          {/* Commit Card — glass */}
          <div className="glass rounded-xl overflow-hidden">
            {/* Header */}
            <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
              <GitCommitHorizontal className="w-4 h-4 text-primary shrink-0" />
              <span className="text-xs text-muted-foreground">Commit to</span>
              <span className="text-xs font-mono text-primary font-medium truncate">{status.branch}</span>
            </div>

            <div className="p-4 space-y-3">
              {/* Conventional commit chips */}
              <ConventionalChips
                summary={summary}
                onApplyPrefix={(prefix) => {
                  const regex = /^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/;
                  if (regex.test(summary)) {
                    setSummary(summary.replace(regex, `${prefix}: `));
                  } else {
                    setSummary(`${prefix}: ${summary}`);
                  }
                }}
                onRemovePrefix={() => {
                  setSummary(summary.replace(/^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/, ""));
                }}
              />

              {/* Summary input */}
              <div className="relative">
                <SquarePen className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
                <input
                  className="flex h-9 w-full rounded-lg border border-input/60 bg-background/50 pl-9 pr-9 py-2 text-sm ring-offset-background placeholder:text-muted-foreground/60 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring/50 transition-all duration-150"
                  placeholder="Summary (required)"
                  value={summary}
                  onChange={(e) => setSummary(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      if (canCommit && !committing) handleCommitClick(false);
                    }
                  }}
                  autoFocus
                />
                {aiConfig?.provider && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        onClick={async () => {
                          const msg = await generateCommitMessage();
                          if (msg) {
                            const nl = msg.indexOf("\n");
                            if (nl >= 0) {
                              setSummary(msg.slice(0, nl).trim());
                              setDescription(msg.slice(nl + 1).trim());
                            } else {
                              setSummary(msg.trim());
                            }
                          }
                        }}
                        disabled={aiGenerating}
                        className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded-md text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors disabled:opacity-40"
                      >
                        {aiGenerating ? (
                          <span className="block w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
                        ) : (
                          <Sparkles className="w-3.5 h-3.5" />
                        )}
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="top">Generate commit message with AI</TooltipContent>
                  </Tooltip>
                )}
              </div>

              {/* Description textarea */}
              <div className="relative">
                <AlignLeft className="absolute left-3 top-3 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
                <textarea
                  className="flex w-full rounded-lg border border-input/60 bg-background/50 pl-9 pr-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground/60 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring/50 transition-all duration-150 resize-none"
                  placeholder="Description (optional)"
                  rows={2}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>

              {/* Commit action buttons */}
              <div className="flex items-center gap-2 pt-1">
                <button
                  disabled={!canCommit || committing}
                  onClick={() => handleCommitClick(false)}
                  className={cn(
                    "flex-1 h-8 rounded-lg text-xs font-medium transition-all duration-150 press-scale",
                    "bg-primary text-primary-foreground hover:brightness-110",
                    "disabled:opacity-40 disabled:cursor-not-allowed disabled:press-scale-none"
                  )}
                >
                  {committing ? "Committing..." : `Commit to ${status.branch}`}
                </button>

                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      disabled={!canCommit || committing}
                      className="h-8 px-2.5 rounded-lg text-xs font-medium transition-all duration-150 press-scale border border-border/50 text-muted-foreground hover:text-foreground hover:bg-accent/50 disabled:opacity-40"
                    >
                      <Upload className="w-3.5 h-3.5" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" side="top" className="w-44">
                    <DropdownMenuItem
                      disabled={!canCommit || committing}
                      onClick={() => handleCommitClick(true)}
                    >
                      <Upload className="w-4 h-4 mr-2" />
                      Commit & Push
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={handleForcePushThenPush}
                      className="text-destructive focus:text-destructive"
                    >
                      <AlertTriangle className="w-4 h-4 mr-2" />
                      Force Push
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </div>

          {/* Stash Section — glass */}
          <div className="glass rounded-xl overflow-hidden">
            <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
              <Archive className="w-4 h-4 text-muted-foreground shrink-0" />
              <span className="text-xs font-medium text-foreground">Stash</span>
            </div>
            <div className="p-3 space-y-2">
              {!status.is_clean && (
                <div className="flex gap-2">
                  <input
                    className="flex-1 h-8 rounded-lg border border-input/60 bg-background/50 px-3 text-xs placeholder:text-muted-foreground/60 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring/50 transition-all duration-150"
                    placeholder="Stash message"
                    value={stashMsg}
                    onChange={(e) => setStashMsg(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleStashPush(); }}
                  />
                  <button
                    onClick={handleStashPush}
                    disabled={stashing}
                    className="h-8 px-2.5 rounded-lg text-xs font-medium transition-all duration-150 press-scale bg-accent/50 text-muted-foreground hover:text-foreground hover:bg-accent disabled:opacity-40"
                  >
                    {stashing ? "..." : "Stash"}
                  </button>
                </div>
              )}

              {stashes.length === 0 ? (
                <p className="text-[11px] text-muted-foreground/60 px-1">No stashes</p>
              ) : (
                <ScrollArea className="max-h-[140px] -mx-1">
                  <div className="space-y-0.5 px-1">
                    {stashes.map((s) => (
                      <div key={s.index}
                        className="flex items-center gap-2 px-2 py-1.5 rounded-lg text-xs hover:bg-accent/30 transition-colors group">
                        <span className="font-mono text-muted-foreground shrink-0 text-[10px]">stash@{s.index}</span>
                        <span className="flex-1 truncate text-muted-foreground/80">{s.message || "(no message)"}</span>
                        <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button className="h-5 w-5 flex items-center justify-center rounded text-muted-foreground hover:text-foreground"
                            onClick={() => stashPop(s.index)} title="Pop">
                            <Upload className="w-3 h-3" />
                          </button>
                          <button className="h-5 w-5 flex items-center justify-center rounded text-muted-foreground hover:text-foreground"
                            onClick={() => stashApply(s.index)} title="Apply">
                            <Play className="w-3 h-3" />
                          </button>
                          <button className="h-5 w-5 flex items-center justify-center rounded text-muted-foreground hover:text-destructive"
                            onClick={() => stashDrop(s.index)} title="Drop">
                            <Archive className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              )}
            </div>
          </div>

          {/* Diff Panel */}
          {showDiffPanel && (
            <div className="flex-1 min-h-0 glass rounded-xl overflow-hidden flex flex-col">
              <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/20 shrink-0">
                <div className="flex items-center gap-2 min-w-0">
                  <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  <span className="text-xs font-mono font-medium truncate">{selectedFile}</span>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-[11px] text-success font-medium">+{diff.total_additions}</span>
                  <span className="text-[11px] text-destructive font-medium">-{diff.total_deletions}</span>
                  <button
                    onClick={() => fetchDiff(selectedFile)}
                    className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
                    title="Refresh diff"
                  >
                    <RefreshCw className="w-3 h-3" />
                  </button>
                  <button
                    onClick={handleCloseDiff}
                    className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
                    title="Close diff"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              </div>
              <ScrollArea className="flex-1">
                <div className="p-2">
                  {(diff.files || []).map((f, idx) => (
                    <DiffViewer key={idx} file={f} />
                  ))}
                </div>
              </ScrollArea>
            </div>
          )}
        </div>
      </div>

      {/* 3-Way Merge Editor modal */}
      <MergeEditor />
    </TooltipProvider>
  );
}

/* ─── Conventional Commit Chips ─── */

function ConventionalChips({
  summary,
  onApplyPrefix,
  onRemovePrefix,
}: {
  summary: string;
  onApplyPrefix: (prefix: string) => void;
  onRemovePrefix: () => void;
}) {
  const currentPrefix = CONVENTIONAL_PREFIXES.find((p) =>
    summary.startsWith(`${p.value}:`) || summary.startsWith(`${p.value}(`)
  );

  return (
    <div className="flex flex-wrap gap-1">
      {CONVENTIONAL_PREFIXES.map((p) => {
        const isActive = currentPrefix?.value === p.value;
        return (
          <button
            key={p.value}
            className={cn(
              "text-[10px] px-1.5 py-0.5 rounded-md border transition-all duration-100 press-scale",
              isActive
                ? "bg-primary/10 border-primary/40 text-primary"
                : "border-border/50 text-muted-foreground hover:border-primary/30 hover:text-foreground"
            )}
            onClick={() => {
              if (isActive) onRemovePrefix();
              else onApplyPrefix(p.value);
            }}
            title={p.desc}
          >
            <span className={p.color}>{p.value}</span>
          </button>
        );
      })}
      {currentPrefix && (
        <button
          className="text-[10px] px-1.5 py-0.5 rounded-md border border-destructive/30 text-destructive hover:bg-destructive/10 transition-colors press-scale"
          onClick={onRemovePrefix}
          title="Remove prefix"
        >
          <X className="w-2.5 h-2.5" />
        </button>
      )}
    </div>
  );
}

/* ─── File Row ─── */

interface FileRowProps {
  file: { path: string; staged: number; unstaged: number };
  type: "staged" | "unstaged" | "untracked";
  selected: boolean;
  checked: boolean;
  onToggleCheck: () => void;
  onClick: () => void;
  onStage: () => void;
  onDiscard?: () => void;
}

function FileRow({ file, type, selected, checked, onToggleCheck, onClick, onStage, onDiscard }: FileRowProps) {
  const config = {
    staged:    { dot: "bg-success",      border: "border-l-success",    label: "M", labelColor: "text-success" },
    unstaged:  { dot: "bg-warning",      border: "border-l-warning",    label: "M", labelColor: "text-warning" },
    untracked: { dot: "bg-muted-foreground/50", border: "border-l-muted-foreground/30", label: "?", labelColor: "text-muted-foreground" },
  }[type];

  return (
    <div
      className={cn(
        "group flex items-center gap-2.5 px-3 py-2 border-l-[3px] cursor-pointer rounded-r-lg transition-all duration-100",
        config.border,
        selected
          ? "bg-accent/40 shadow-sm"
          : "hover:bg-accent/20"
      )}
      onClick={onClick}
    >
      {/* Checkbox (stop propagation so click doesn't also open diff) */}
      <div className="flex items-center shrink-0" onClick={(e) => e.stopPropagation()}>
        <Checkbox
          checked={checked}
          onChange={onToggleCheck}
          className="cursor-pointer data-[state=checked]:bg-primary data-[state=checked]:border-primary"
        />
      </div>

      {/* Status badge */}
      <div className={cn("w-5 h-5 flex items-center justify-center rounded-md text-[10px] font-bold font-mono", config.labelColor, "bg-current/5")}>
        <span className="opacity-80">{config.label}</span>
      </div>

      {/* File path */}
      <span className="flex-1 truncate text-xs font-mono text-foreground/80 group-hover:text-foreground transition-colors">
        {file.path}
      </span>

      {/* Actions (visible on hover) */}
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
        {type !== "staged" && onDiscard && (
          <button
            className="h-6 w-6 flex items-center justify-center rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
            onClick={onDiscard}
            title="Discard changes"
          >
            <Undo2 className="w-3 h-3" />
          </button>
        )}
        <button
          className={cn(
            "h-6 px-2 text-[10px] font-medium rounded-md transition-colors press-scale",
            type === "staged"
              ? "text-muted-foreground hover:text-foreground hover:bg-accent/50"
              : "text-success hover:text-success hover:bg-success/10"
          )}
          onClick={onStage}
        >
          {type === "staged" ? "Unstage" : "Stage"}
        </button>
      </div>
    </div>
  );
}

/* ─── Merge Conflict Banner ─── */

function MergeConflictBanner({
  files,
  onResolve,
  onOpenEditor,
}: {
  files: Array<{ path: string; staged: number; unstaged: number }>;
  onResolve: (file: string, side: "ours" | "theirs") => void;
  onOpenEditor?: (file: string) => void;
}) {
  const conflictedFiles = files.filter(
    (f) => f.staged === STATUS_UNMERGED || f.unstaged === STATUS_UNMERGED
  );

  if (conflictedFiles.length === 0) return null;

  return (
    <div className="mb-4 rounded-xl border border-destructive/20 bg-destructive/[0.03] overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3 bg-destructive/[0.06] border-b border-destructive/10">
        <AlertOctagon className="w-5 h-5 text-destructive shrink-0" />
        <div>
          <p className="text-sm font-semibold text-destructive">Merge Conflict</p>
          <p className="text-[11px] text-destructive/70">
            {conflictedFiles.length} file{conflictedFiles.length !== 1 ? "s" : ""} — resolve before committing
          </p>
        </div>
      </div>

      <div className="divide-y divide-destructive/5">
        {conflictedFiles.map((f) => (
          <div key={f.path} className="flex items-center gap-3 px-4 py-2.5 hover:bg-destructive/[0.02] transition-colors">
            <FileText className="w-4 h-4 text-destructive/50 shrink-0" />
            <span className="text-xs font-mono flex-1 truncate text-foreground/80">{f.path}</span>
            <div className="flex gap-1.5 shrink-0">
              <button
                className="text-[10px] px-2 py-1 rounded-md border border-primary/30 text-primary hover:bg-primary/10 transition-colors press-scale font-medium"
                onClick={() => onResolve(f.path, "ours")}
              >
                Ours
              </button>
              <button
                className="text-[10px] px-2 py-1 rounded-md border border-[hsl(var(--pr-merged))/0.3] text-[hsl(var(--pr-merged))] hover:bg-[hsl(var(--pr-merged))/0.1] transition-colors press-scale font-medium"
                onClick={() => onResolve(f.path, "theirs")}
              >
                Theirs
              </button>
              <span className="text-muted-foreground/20 self-center">|</span>
              <button
                className="text-[10px] px-2 py-1 rounded-md bg-muted/40 text-muted-foreground hover:bg-muted/60 transition-colors press-scale font-mono"
                onClick={() => onOpenEditor?.(f.path)}
              >
                Open
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}


