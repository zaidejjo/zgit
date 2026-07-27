import { useEffect, useState, useCallback } from "react";
import {
  Download, Upload, Undo2, Archive, RotateCcw, Play, X,
  GitCommitHorizontal, SquarePen, AlignLeft, CheckCheck,
  AlertTriangle, ChevronDown, ListChecks, FileText,
  RefreshCw, AlertOctagon, ArrowRight, Sparkles,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
  { value: "feat", label: "feat", desc: "New feature", color: "text-green-500" },
  { value: "fix", label: "fix", desc: "Bug fix", color: "text-red-500" },
  { value: "docs", label: "docs", desc: "Documentation", color: "text-blue-500" },
  { value: "style", label: "style", desc: "Formatting", color: "text-purple-500" },
  { value: "refactor", label: "refactor", desc: "Code restructure", color: "text-yellow-500" },
  { value: "test", label: "test", desc: "Tests", color: "text-orange-500" },
  { value: "chore", label: "chore", desc: "Maintenance", color: "text-muted-foreground" },
] as const;

// StatusType enum values from Go models
const STATUS_UNTRACKED = 0;
const STATUS_UNMODIFIED = 7;
const STATUS_UNMERGED = 6; // StatusUpdatedButUnmerged

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
    fetchDiff(path);
  };

  const handleCloseDiff = () => {
    setSelectedFile(null);
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
    // Stage all checked files in sequence
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
    // Force push first, then do a normal commit+push sequence if needed
    // This is for when user wants to push after force push
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

  if (!status) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
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

  const hasChecked = checkedFiles.size > 0;
  const checkedStaged = Array.from(checkedFiles).filter((p) =>
    stagedFiles.some((f) => f.path === p)
  );
  const checkedUnstaged = Array.from(checkedFiles).filter((p) =>
    unstagedFiles.some((f) => f.path === p) || untrackedFiles.some((f) => f.path === p)
  );

  const canCommit = stagedFiles.length > 0 && summary.trim().length > 0;

  return (
    <>
    <TooltipProvider delayDuration={300}>
      <div className="flex gap-4 h-full">
        {/* Left column: file list */}
        <div className="flex-1 min-w-0 flex flex-col">
          {/* Header row */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold">Status</h2>
              {status.branch && (
                <Badge variant="outline" className="font-mono">
                  {status.branch}
                </Badge>
              )}
              {!status.is_clean && (
                <Badge variant="warning">
                  {status.ahead > 0 ? `+${status.ahead}` : ""}
                  {status.behind > 0 ? ` -${status.behind}` : ""}
                </Badge>
              )}
            </div>
            <div className="flex gap-1 items-center">
              <Button variant="ghost" size="sm" className="h-7 px-2 text-xs"
                onClick={() => gitFetch()} disabled={loading.fetch}>
                <Download className="w-3.5 h-3.5 mr-1" />
                Fetch
              </Button>
              <Button variant="ghost" size="sm" className="h-7 px-2 text-xs"
                onClick={() => gitPull(false)} disabled={loading.pull}>
                <Upload className="w-3.5 h-3.5 mr-1" />
                Pull
              </Button>
              <div className="w-px h-5 bg-border mx-1" />
              {/* Batch stage/unstage buttons */}
              {hasChecked ? (
                <>
                  {checkedStaged.length > 0 && (
                    <Button variant="outline" size="sm" onClick={handleUnstageChecked}>
                      Unstage Selected ({checkedStaged.length})
                    </Button>
                  )}
                  {checkedUnstaged.length > 0 && (
                    <Button variant="outline" size="sm" onClick={handleStageChecked}>
                      Stage Selected ({checkedUnstaged.length})
                    </Button>
                  )}
                </>
              ) : (
                <>
                  <Button variant="outline" size="sm" onClick={() => stageAll()}>
                    Stage All
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => unstageAll()}>
                    Unstage All
                  </Button>
                </>
              )}
            </div>
          </div>

          {/* Merge conflict resolution */}
          {status.is_merging && (
            <>
              <MergeConflictBanner
                files={status.files || []}
                onResolve={(file, side) => resolveConflict(file, side)}
                onOpenEditor={(file) => openMergeEditor(file)}
              />
            </>
          )}

          {/* File list scroll area */}
          <ScrollArea className="flex-1 min-h-0">
            {status.is_clean && (
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <div className="w-16 h-16 rounded-full bg-primary/5 flex items-center justify-center mb-4">
                  <CheckCheck className="w-8 h-8 text-primary/40" />
                </div>
                <h3 className="text-base font-semibold text-foreground mb-1">Working tree is clean</h3>
                <p className="text-sm text-muted-foreground max-w-xs">
                  No modified or untracked files. Make changes to your project to see them here.
                </p>
              </div>
            )}

            {!status.is_clean && (
              <>
                {/* Staged files */}
                {stagedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-green-600 dark:text-green-400 mb-2">
                      Staged ({stagedFiles.length})
                    </h3>
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
                )}

                {/* Staged counter */}
                {stagedFiles.length > 0 && (
                  <div className="flex items-center gap-1.5 mb-4 px-3 py-2 rounded-lg bg-primary/5 border border-primary/10 text-sm text-primary">
                    <ListChecks className="w-4 h-4" />
                    <span>
                      <strong>{stagedFiles.length}</strong> file{stagedFiles.length !== 1 ? "s" : ""} staged for commit
                    </span>
                  </div>
                )}

                {/* Unstaged files */}
                {unstagedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-yellow-600 dark:text-yellow-400 mb-2">
                      Unstaged ({unstagedFiles.length})
                    </h3>
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
                )}

                {/* Untracked files */}
                {untrackedFiles.length > 0 && (
                  <div className="mb-4">
                    <h3 className="text-sm font-medium text-muted-foreground mb-2">
                      Untracked ({untrackedFiles.length})
                    </h3>
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
                )}
              </>
            )}
          </ScrollArea>
        </div>

        {/* Right column: commit + stash + diff */}
        <div className="w-[30rem] min-w-0 flex flex-col gap-4">
          {/* Commit section — always visible */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2">
                <GitCommitHorizontal className="w-4 h-4" />
                Commit to{" "}
                <span className="font-mono text-primary">{status.branch}</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {/* Conventional commit chips */}
              <ConventionalChips
                summary={summary}
                onApplyPrefix={(prefix) => {
                  // Apply or remove prefix from summary
                  const regex = /^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/;
                  if (regex.test(summary)) {
                    // Replace existing prefix
                    setSummary(summary.replace(regex, `${prefix}: `));
                  } else {
                    // Prepend prefix
                    setSummary(`${prefix}: ${summary}`);
                  }
                }}
                onRemovePrefix={() => {
                  setSummary(summary.replace(/^(feat|fix|docs|style|refactor|test|chore)(\([^)]*\))?:\s*/, ""));
                }}
              />

              {/* Summary input */}
              <div className="relative">
                <SquarePen className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                <input
                  className="flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-10 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      onClick={async () => {
                        const msg = await generateCommitMessage();
                        if (msg) {
                          // Split on first newline to separate summary from description
                          const nl = msg.indexOf("\n");
                          if (nl >= 0) {
                            setSummary(msg.slice(0, nl).trim());
                            setDescription(msg.slice(nl + 1).trim());
                          } else {
                            setSummary(msg.trim());
                          }
                        }
                      }}
                      disabled={aiGenerating || !aiConfig?.provider}
                      className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                      title={aiConfig?.provider ? "Generate commit message with AI" : "Configure AI in Settings first"}
                    >
                      {aiGenerating ? (
                        <span className="block w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                      ) : (
                        <Sparkles className="w-4 h-4" />
                      )}
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {aiConfig?.provider ? "Generate commit message with AI" : "Configure AI in Settings first"}
                  </TooltipContent>
                </Tooltip>
              </div>

              {/* Description textarea */}
              <div className="relative">
                <AlignLeft className="absolute left-3 top-3 w-4 h-4 text-muted-foreground pointer-events-none" />
                <textarea
                  className="flex w-full rounded-md border border-input bg-background pl-9 pr-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
                  placeholder="Description (optional)"
                  rows={3}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>

              {/* Commit action buttons */}
              <div className="flex gap-2">
                {/* Primary: Commit to branch */}
                <Tooltip open={!canCommit ? undefined : false}>
                  <TooltipTrigger asChild>
                    <span tabIndex={0} className="flex-1">
                      <Button
                        size="sm"
                        className="w-full"
                        disabled={!canCommit || committing}
                        onClick={() => handleCommitClick(false)}
                      >
                        <CheckCheck className="w-4 h-4 mr-1.5" />
                        {committing ? "Committing..." : `Commit to ${status.branch}`}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {!canCommit && (
                    <TooltipContent side="bottom">
                      {stagedFiles.length === 0
                        ? "Stage at least one file first"
                        : "Enter a summary message"}
                    </TooltipContent>
                  )}
                </Tooltip>

                {/* Secondary: Commit & Push with force push dropdown */}
                <div className="flex">
                  <Button
                    variant="secondary"
                    size="sm"
                    className="rounded-r-none"
                    disabled={!canCommit || committing}
                    onClick={() => handleCommitClick(true)}
                  >
                    <Upload className="w-4 h-4 mr-1.5" />
                    {committing ? "Committing..." : "Commit & Push"}
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="secondary"
                        size="sm"
                        className="rounded-l-none border-l border-secondary-foreground/20 px-2"
                        disabled={committing}
                        aria-label="Push options"
                      >
                        <ChevronDown className="w-3.5 h-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" side="bottom">
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
            </CardContent>
          </Card>

          {/* Stash section */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Archive className="w-4 h-4" />
                Stash
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {!status.is_clean && (
                <div className="flex gap-2">
                  <input
                    className="flex-1 rounded border border-input bg-background px-2 py-1.5 text-xs"
                    placeholder="Stash message (optional)"
                    value={stashMsg}
                    onChange={(e) => setStashMsg(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleStashPush(); }}
                  />
                  <Button size="sm" className="h-7 text-xs" onClick={handleStashPush} disabled={stashing}>
                    <RotateCcw className="w-3 h-3 mr-1" />
                    {stashing ? "..." : "Stash"}
                  </Button>
                </div>
              )}

              {stashes.length === 0 ? (
                <p className="text-xs text-muted-foreground">No stashes</p>
              ) : (
                <ScrollArea className="max-h-48">
                  <div className="space-y-1">
                    {stashes.map((s) => (
                      <div key={s.index}
                        className="flex items-center gap-2 px-2 py-1.5 rounded text-xs hover:bg-accent/50 group">
                        <span className="font-mono text-muted-foreground shrink-0">stash@{s.index}</span>
                        <span className="flex-1 truncate">{s.message || "(no message)"}</span>
                        <div className="flex gap-0.5 opacity-0 group-hover:opacity-100">
                          <button className="h-5 px-1 text-muted-foreground hover:text-foreground"
                            onClick={() => stashPop(s.index)} title="Pop">
                            <Upload className="w-3 h-3" />
                          </button>
                          <button className="h-5 px-1 text-muted-foreground hover:text-foreground"
                            onClick={() => stashApply(s.index)} title="Apply">
                            <Play className="w-3 h-3" />
                          </button>
                          <button className="h-5 px-1 text-muted-foreground hover:text-destructive"
                            onClick={() => stashDrop(s.index)} title="Drop">
                            <Archive className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              )}
            </CardContent>
          </Card>

          {/* Diff panel */}
          {selectedFile && diff && (
            <div className="flex-1 min-h-0 border rounded-lg flex flex-col">
              <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/20 shrink-0">
                <h3 className="text-sm font-semibold font-mono truncate flex items-center gap-1.5">
                  <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                  {selectedFile}
                </h3>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-green-600 font-medium">+{diff.total_additions}</span>
                  <span className="text-xs text-red-600 font-medium">-{diff.total_deletions}</span>
                  <Button variant="ghost" size="sm" className="h-6 w-6 p-0"
                    onClick={() => fetchDiff(selectedFile)} title="Refresh diff">
                    <RefreshCw className="w-3.5 h-3.5" />
                  </Button>
                  <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={handleCloseDiff}>
                    <X className="w-3.5 h-3.5" />
                  </Button>
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
    </TooltipProvider>

    {/* 3-Way Merge Editor modal */}
    <MergeEditor />
    </>);
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
              "text-xs px-1.5 py-0.5 rounded border transition-colors",
              isActive
                ? "bg-primary/10 border-primary text-primary"
                : "border-border text-muted-foreground hover:border-primary/50 hover:text-foreground"
            )}
            onClick={() => {
              if (isActive) {
                onRemovePrefix();
              } else {
                onApplyPrefix(p.value);
              }
            }}
            title={p.desc}
          >
            <span className={p.color}>{p.value}</span>
          </button>
        );
      })}
      {currentPrefix && (
        <button
          className="text-xs px-1.5 py-0.5 rounded border border-destructive/50 text-destructive hover:bg-destructive/10 transition-colors"
          onClick={onRemovePrefix}
          title="Remove prefix"
        >
          <X className="w-3 h-3" />
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
  const colorClass = {
    staged: "border-l-green-500",
    unstaged: "border-l-yellow-500",
    untracked: "border-l-muted-foreground",
  }[type];

  const label = {
    staged: "M",
    unstaged: "M",
    untracked: "?",
  }[type];

  const labelColor = {
    staged: "text-green-500",
    unstaged: "text-yellow-500",
    untracked: "text-muted-foreground",
  }[type];

  return (
    <div
      className={cn(
        "group flex items-center gap-2 px-3 py-1.5 border-l-2 cursor-pointer hover:bg-accent/50 text-sm transition-colors",
        colorClass,
        selected && "bg-accent"
      )}
    >
      {/* Checkbox */}
      <div className="flex items-center shrink-0" onClick={(e) => e.stopPropagation()}>
        <Checkbox
          checked={checked}
          onChange={onToggleCheck}
          className="cursor-pointer"
        />
      </div>

      {/* Status label */}
      <span className={cn("w-5 text-center font-mono text-xs", labelColor)}>
        {label}
      </span>

      {/* File path — click to view diff */}
      <span className="flex-1 truncate" onClick={onClick}>
        {file.path}
      </span>

      {/* Actions */}
      <div className="flex gap-0.5 opacity-0 group-hover:opacity-100" onClick={(e) => e.stopPropagation()}>
        {type !== "staged" && onDiscard && (
          <button
            className="h-6 px-1.5 text-xs text-muted-foreground hover:text-destructive transition-colors"
            onClick={onDiscard}
            title="Discard changes"
          >
            <Undo2 className="w-3 h-3" />
          </button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-xs"
          onClick={onStage}
        >
          {type === "staged" ? "Unstage" : "Stage"}
        </Button>
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
    <div className="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 overflow-hidden">
      {/* Banner header */}
      <div className="flex items-center gap-2 px-4 py-3 bg-destructive/10 border-b border-destructive/20">
        <AlertOctagon className="w-5 h-5 text-destructive shrink-0" />
        <div>
          <p className="text-sm font-semibold text-destructive">Merge Conflict</p>
          <p className="text-xs text-destructive/80">
            {conflictedFiles.length} file{conflictedFiles.length !== 1 ? "s" : ""} with conflicts — resolve before committing
          </p>
        </div>
      </div>

      {/* Conflicted files */}
      <div className="divide-y divide-destructive/10">
        {conflictedFiles.map((f) => (
          <div key={f.path} className="flex items-center gap-3 px-4 py-2.5 hover:bg-destructive/5 transition-colors">
            <FileText className="w-4 h-4 text-destructive/60 shrink-0" />
            <span className="text-sm font-mono flex-1 truncate">{f.path}</span>
            <div className="flex gap-1.5 shrink-0">
              <button
                className="text-xs px-2 py-1 rounded border border-blue-500/30 text-blue-600 hover:bg-blue-500/10 transition-colors"
                onClick={() => onResolve(f.path, "ours")}
                title="Use our version"
              >
                <ArrowRight className="w-3 h-3 mr-0.5 inline" />
                Ours
              </button>
              <button
                className="text-xs px-2 py-1 rounded border border-purple-500/30 text-purple-600 hover:bg-purple-500/10 transition-colors"
                onClick={() => onResolve(f.path, "theirs")}
                title="Use their version"
              >
                Theirs
                <ArrowRight className="w-3 h-3 ml-0.5 inline" />
              </button>
              <span className="text-muted-foreground/30">|</span>
              <button
                className="text-xs px-2 py-1 rounded bg-muted text-muted-foreground hover:bg-muted/80 transition-colors font-mono"
                onClick={() => onOpenEditor?.(f.path)}
                title="Open 3-way merge editor"
              >
                Open Editor
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}


