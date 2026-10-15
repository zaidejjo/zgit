import { useEffect, useMemo, useState } from "react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";
import type { WorkflowRun, Job } from "@/store/app";
import {
  Zap, RefreshCw, Search, Ban, RotateCcw, ExternalLink,
  CheckCircle2, XCircle, Loader2, Clock, MinusCircle,
  ChevronRight, TerminalSquare, GitBranch, Hash, History,
} from "lucide-react";

type Filter = "all" | "success" | "failure" | "active";

const FILTERS: { id: Filter; label: string }[] = [
  { id: "all", label: "All" },
  { id: "active", label: "Active" },
  { id: "success", label: "Passed" },
  { id: "failure", label: "Failed" },
];

function runMatches(run: WorkflowRun, f: Filter, q: string): boolean {
  if (q) {
    const hay = `${run.workflow_name} ${run.branch} ${run.head_sha} #${run.run_number}`.toLowerCase();
    if (!hay.includes(q.toLowerCase())) return false;
  }
  switch (f) {
    case "active": return run.status === "IN_PROGRESS" || run.status === "QUEUED";
    case "success": return run.conclusion === "success" || run.conclusion === "SUCCESS";
    case "failure": return run.conclusion === "failure" || run.conclusion === "FAILURE";
    default: return true;
  }
}

function timeAgo(iso?: string): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (!t) return "";
  const s = Math.floor((Date.now() - t) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 30) return `${Math.floor(s / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
}

function RunStatusIcon({ run, className }: { run: WorkflowRun; className?: string }) {
  const c = run.conclusion?.toLowerCase();
  const s = run.status?.toLowerCase();
  if (c === "success") return <CheckCircle2 className={cn("text-green-500", className)} />;
  if (c === "failure") return <XCircle className={cn("text-red-500", className)} />;
  if (c === "cancelled" || c === "skipped") return <MinusCircle className={cn("text-muted-foreground", className)} />;
  if (s === "in_progress") return <Loader2 className={cn("text-yellow-500 animate-spin", className)} />;
  return <Clock className={cn("text-blue-500", className)} />;
}

function runTone(run: WorkflowRun): string {
  const c = run.conclusion?.toLowerCase();
  if (c === "success") return "border-green-500/20 hover:border-green-500/40";
  if (c === "failure") return "border-red-500/20 hover:border-red-500/40";
  if (run.status?.toLowerCase() === "in_progress") return "border-yellow-500/25 hover:border-yellow-500/45";
  return "border-border/50 hover:border-border";
}

export default function ActionsPage() {
  const {
    workflowRuns, selectedRunJobs, selectedJobLogs, jobLogsError,
    loading, ghAuthenticated, setLoginDialogOpen,
    fetchWorkflowRuns, reRunWorkflow, cancelWorkflowRun,
    fetchWorkflowJobs, fetchJobLogs, clearJobLogs,
  } = useAppStore();

  const [selectedRunID, setSelectedRunID] = useState<number | null>(null);
  const [selectedJobID, setSelectedJobID] = useState<number | null>(null);
  const [filter, setFilter] = useState<Filter>("all");
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (ghAuthenticated) fetchWorkflowRuns();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ghAuthenticated]);

  const runs = useMemo(
    () => (workflowRuns || []).filter((r) => runMatches(r, filter, query)),
    [workflowRuns, filter, query],
  );
  const selectedRun = useMemo(
    () => (workflowRuns || []).find((r) => r.id === selectedRunID),
    [workflowRuns, selectedRunID],
  );
  const activeCount = useMemo(
    () => (workflowRuns || []).filter((r) => r.status === "IN_PROGRESS" || r.status === "QUEUED").length,
    [workflowRuns],
  );

  const handleSelectRun = (runID: number) => {
    setSelectedRunID(runID);
    setSelectedJobID(null);
    clearJobLogs();
    fetchWorkflowJobs(runID);
  };

  const handleSelectJob = (jobID: number) => {
    setSelectedJobID(jobID);
    fetchJobLogs(jobID);
  };

  if (!ghAuthenticated) {
    return (
      <div className="h-full flex items-center justify-center p-6">
        <EmptyState
          icon={<Zap className="w-16 h-16" />}
          title="CI at a glance"
          description="Sign in with GitHub to watch workflow runs, inspect jobs and tail logs without leaving zgit."
          action={{ label: "Sign in with GitHub", onClick: () => setLoginDialogOpen(true) }}
          className="max-w-md w-full"
        />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col gap-3 min-h-0">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2 mr-auto">
          <h2 className="text-xl font-bold">Actions</h2>
          {(workflowRuns?.length || 0) > 0 && (
            <Badge variant="outline" className="text-xs tabular-nums">{workflowRuns.length}</Badge>
          )}
          {activeCount > 0 && (
            <Badge className="text-[11px] bg-yellow-500/15 text-yellow-600 border-yellow-500/25 hover:bg-yellow-500/20">
              <Loader2 className="w-3 h-3 mr-1 animate-spin" />
              {activeCount} running
            </Badge>
          )}
        </div>
        <div className="relative w-full sm:w-52">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter runs..."
            className="h-8 pl-8 text-xs"
          />
        </div>
        <Button size="sm" variant="outline" className="h-8" onClick={() => fetchWorkflowRuns()} disabled={!!loading.workflowRuns}>
          <RefreshCw className={cn("w-3.5 h-3.5 mr-1.5", loading.workflowRuns && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {/* Filter pills */}
      <div className="flex items-center gap-1 flex-wrap">
        {FILTERS.map((f) => (
          <button
            key={f.id}
            onClick={() => setFilter(f.id)}
            className={cn(
              "px-2.5 py-1 rounded-full text-xs font-medium transition-colors",
              filter === f.id
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Content: stack on mobile, split on desktop */}
      <div className="flex-1 min-h-0 flex flex-col lg:flex-row gap-3">
        {/* Run list */}
        <div className="flex-1 lg:flex-[5] min-h-0 flex flex-col rounded-xl border border-border/50 bg-card/30 overflow-hidden">
          <div className="px-3 py-2 border-b border-border/40 flex items-center gap-2 text-xs text-muted-foreground">
            <History className="w-3.5 h-3.5" />
            <span className="font-medium">Workflow runs</span>
            <span className="ml-auto tabular-nums">{runs.length}</span>
          </div>
          <ScrollArea className="flex-1 min-h-0">
            {loading.workflowRuns && runs.length === 0 ? (
              <div className="p-2 space-y-1.5">
                {[0, 1, 2, 3].map((i) => (
                  <div key={i} className="rounded-lg border border-border/40 p-3 animate-pulse">
                    <div className="h-3.5 w-2/3 rounded bg-muted" />
                    <div className="h-3 w-1/3 rounded bg-muted mt-2" />
                  </div>
                ))}
              </div>
            ) : runs.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-14 px-6 text-center">
                <Zap className="w-8 h-8 text-muted-foreground/40 mb-2" />
                <p className="text-sm font-medium">No runs match</p>
                <p className="text-xs text-muted-foreground mt-1">Try a different filter or push a commit to trigger CI.</p>
              </div>
            ) : (
              <div className="p-2 space-y-1.5">
                {runs.map((run) => (
                  <RunCard
                    key={run.id}
                    run={run}
                    selected={selectedRunID === run.id}
                    onSelect={() => handleSelectRun(run.id)}
                    onReRun={() => reRunWorkflow(run.id)}
                    onCancel={() => cancelWorkflowRun(run.id)}
                  />
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        {/* Detail */}
        <div className="flex-1 lg:flex-[6] min-h-0 flex flex-col rounded-xl border border-border/50 bg-card/30 overflow-hidden">
          {!selectedRun ? (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center">
              <TerminalSquare className="w-8 h-8 text-muted-foreground/40 mb-2" />
              <p className="text-sm font-medium">Select a run to inspect jobs</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-[260px]">
                Jobs, step outcomes and full logs appear here.
              </p>
            </div>
          ) : (
            <>
              <DetailHeader
                run={selectedRun}
                onReRun={() => reRunWorkflow(selectedRun.id)}
                onCancel={() => cancelWorkflowRun(selectedRun.id)}
              />
              <ScrollArea className="flex-1 min-h-0">
                <div className="p-2.5 space-y-1">
                  {(selectedRunJobs || []).length === 0 && !selectedJobLogs && !jobLogsError ? (
                    <p className="text-xs text-muted-foreground text-center py-8">No jobs reported for this run yet.</p>
                  ) : (
                    (selectedRunJobs || []).map((job) => (
                      <JobCard
                        key={job.id}
                        job={job}
                        selected={selectedJobID === job.id}
                        onSelect={() => handleSelectJob(job.id)}
                      />
                    ))
                  )}
                </div>
                {selectedJobLogs ? (
                  <div className="px-2.5 pb-3">
                    <div className="rounded-lg overflow-hidden border border-border/50">
                      <div className="flex items-center gap-1.5 px-3 py-1.5 bg-muted/40 border-b border-border/40">
                        <span className="w-2.5 h-2.5 rounded-full bg-red-500/70" />
                        <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/70" />
                        <span className="w-2.5 h-2.5 rounded-full bg-green-500/70" />
                        <span className="ml-2 text-[11px] font-mono text-muted-foreground">job log</span>
                      </div>
                      <pre className="text-[11px] leading-relaxed font-mono bg-black/80 text-zinc-200 dark:bg-black/60 p-3 overflow-x-auto whitespace-pre-wrap max-h-80 overflow-y-auto">
                        {selectedJobLogs}
                      </pre>
                    </div>
                  </div>
                ) : jobLogsError ? (
                  <div className="px-2.5 pb-3">
                    <div className="flex items-start gap-2.5 rounded-lg border border-dashed border-border/60 px-3.5 py-3.5">
                      <Clock className="w-4 h-4 text-muted-foreground shrink-0 mt-0.5" />
                      <div className="min-w-0">
                        <p className="text-xs font-medium">Logs unavailable</p>
                        <p className="text-[11px] text-muted-foreground mt-0.5 leading-relaxed">{jobLogsError}</p>
                      </div>
                    </div>
                  </div>
                ) : null}
              </ScrollArea>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function DetailHeader({ run, onReRun, onCancel }: { run: WorkflowRun; onReRun: () => void; onCancel: () => void }) {
  const isActive = run.status === "IN_PROGRESS" || run.status === "QUEUED";
  return (
    <div className="px-3.5 py-3 border-b border-border/40 bg-gradient-to-b from-accent/40 to-transparent">
      <div className="flex items-start gap-2.5 min-w-0">
        <RunStatusIcon run={run} className="w-5 h-5 mt-0.5 shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-sm truncate">{run.workflow_name}</p>
          <p className="text-[11px] text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
            <span className="font-mono">#{run.run_number}</span>
            <span className="flex items-center gap-1"><GitBranch className="w-3 h-3" />{run.branch}</span>
            <span className="flex items-center gap-1 font-mono"><Hash className="w-3 h-3" />{run.head_sha.slice(0, 7)}</span>
            {run.created_at && <span>{timeAgo(run.created_at)}</span>}
          </p>
        </div>
        <div className="flex gap-1 shrink-0">
          {run.html_url && (
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" title="Open in browser"
              onClick={() => window.open(run.html_url, "_blank")}>
              <ExternalLink className="w-3.5 h-3.5" />
            </Button>
          )}
          {isActive ? (
            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs text-muted-foreground hover:text-red-500" onClick={onCancel} title="Cancel run">
              <Ban className="w-3.5 h-3.5 mr-1" /> Cancel
            </Button>
          ) : (
            run.conclusion?.toLowerCase() === "failure" && (
              <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={onReRun} title="Re-run workflow">
                <RotateCcw className="w-3.5 h-3.5 mr-1" /> Re-run
              </Button>
            )
          )}
        </div>
      </div>
    </div>
  );
}

function RunCard({ run, selected, onSelect, onReRun, onCancel }: {
  run: WorkflowRun; selected: boolean; onSelect: () => void; onReRun: () => void; onCancel: () => void;
}) {
  const isActive = run.status === "IN_PROGRESS" || run.status === "QUEUED";
  const eventLabel = run.event?.replace(/_/g, " ") || run.event;
  return (
    <div
      onClick={onSelect}
      className={cn(
        "rounded-lg border p-2.5 cursor-pointer transition-all bg-background/60",
        runTone(run),
        selected && "ring-1 ring-primary/50 bg-accent/40",
      )}
    >
      <div className="flex items-center gap-2 min-w-0">
        <RunStatusIcon run={run} className="w-4 h-4 shrink-0" />
        <span className="font-medium text-sm truncate flex-1">{run.workflow_name}</span>
        <ChevronRight className={cn("w-3.5 h-3.5 shrink-0 transition-colors", selected ? "text-primary" : "text-muted-foreground/40")} />
      </div>
      <div className="flex items-center gap-1.5 mt-1.5 ml-6 flex-wrap">
        <span className="font-mono text-[11px] text-muted-foreground">#{run.run_number}</span>
        <Badge variant="secondary" className="text-[10px] h-4.5 px-1.5 font-normal">{eventLabel}</Badge>
        <span className="text-[11px] text-muted-foreground flex items-center gap-1 max-w-[140px] truncate">
          <GitBranch className="w-3 h-3 shrink-0" />{run.branch}
        </span>
        <span className="font-mono text-[11px] text-muted-foreground">{run.head_sha.slice(0, 7)}</span>
        {run.created_at && (
          <span className="text-[11px] text-muted-foreground ml-auto">{timeAgo(run.created_at)}</span>
        )}
      </div>
      {(isActive || run.conclusion?.toLowerCase() === "failure") && (
        <div className="flex gap-1 mt-2 ml-6" onClick={(e) => e.stopPropagation()}>
          {isActive && (
            <Button variant="ghost" size="sm" className="h-6 px-2 text-[11px] text-muted-foreground hover:text-red-500" onClick={onCancel}>
              <Ban className="w-3 h-3 mr-1" /> Cancel
            </Button>
          )}
          {run.conclusion?.toLowerCase() === "failure" && (
            <Button variant="ghost" size="sm" className="h-6 px-2 text-[11px]" onClick={onReRun}>
              <RotateCcw className="w-3 h-3 mr-1" /> Re-run
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

function JobCard({ job, selected, onSelect }: { job: Job; selected: boolean; onSelect: () => void }) {
  const c = job.conclusion?.toLowerCase();
  return (
    <div
      onClick={onSelect}
      className={cn(
        "flex items-center gap-2.5 px-3 py-2 rounded-lg cursor-pointer transition-colors border",
        selected ? "bg-accent border-border" : "border-transparent hover:border-border/60 hover:bg-accent/40",
      )}
    >
      {c === "success" ? <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" />
        : c === "failure" ? <XCircle className="w-4 h-4 text-red-500 shrink-0" />
        : job.status === "in_progress" ? <Loader2 className="w-4 h-4 text-yellow-500 animate-spin shrink-0" />
        : c === "cancelled" || c === "skipped" ? <MinusCircle className="w-4 h-4 text-muted-foreground shrink-0" />
        : <Clock className="w-4 h-4 text-muted-foreground shrink-0" />}
      <span className="flex-1 text-sm truncate">{job.name}</span>
      {job.runner_name && (
        <span className="text-[11px] text-muted-foreground truncate max-w-[120px] hidden sm:inline">{job.runner_name}</span>
      )}
      {c && (
        <Badge variant="outline" className={cn(
          "text-[10px] font-mono h-5 shrink-0",
          c === "success" && "text-green-600 border-green-500/30",
          c === "failure" && "text-red-500 border-red-500/30",
        )}>
          {c}
        </Badge>
      )}
    </div>
  );
}
