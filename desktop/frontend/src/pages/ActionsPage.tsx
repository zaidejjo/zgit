import { useEffect, useState } from "react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import type { WorkflowRun, Job } from "@/store/app";

const STATUS_COLORS: Record<string, string> = {
  QUEUED: "bg-blue-500/10 text-blue-500 border-blue-500/20",
  IN_PROGRESS: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
  COMPLETED: "bg-green-500/10 text-green-500 border-green-500/20",
  CANCELLED: "bg-gray-500/10 text-gray-500 border-gray-500/20",
  FAILURE: "bg-red-500/10 text-red-500 border-red-500/20",
  SUCCESS: "bg-green-500/10 text-green-500 border-green-500/20",
  SKIPPED: "bg-gray-500/10 text-gray-500 border-gray-500/20",
};

const EVENT_COLORS: Record<string, string> = {
  push: "bg-blue-500/10 text-blue-500",
  pull_request: "bg-purple-500/10 text-purple-500",
  workflow_dispatch: "bg-orange-500/10 text-orange-500",
  schedule: "bg-cyan-500/10 text-cyan-500",
};

function StatusBadge({ status, conclusion }: { status: string; conclusion?: string }) {
  const label = conclusion && conclusion !== "null" ? conclusion : status;
  const colorClass = STATUS_COLORS[conclusion || status] || STATUS_COLORS.QUEUED;
  return (
    <Badge variant="outline" className={cn("text-xs font-mono", colorClass)}>
      {label?.replace("_", " ")}
    </Badge>
  );
}

function EventBadge({ event }: { event: string }) {
  const colorClass = EVENT_COLORS[event] || "bg-muted text-muted-foreground";
  return (
    <span className={cn("text-xs px-1.5 py-0.5 rounded font-mono", colorClass)}>
      {event}
    </span>
  );
}

function JobStatusIcon({ status, conclusion }: { status: string; conclusion?: string }) {
  if (conclusion === "success") return <span className="text-green-500">✓</span>;
  if (conclusion === "failure") return <span className="text-red-500">✗</span>;
  if (status === "in_progress") return <span className="text-yellow-500 animate-pulse">◉</span>;
  if (status === "queued" || status === "pending") return <span className="text-muted-foreground">○</span>;
  if (conclusion === "cancelled" || conclusion === "skipped") return <span className="text-muted-foreground">−</span>;
  return <span className="text-muted-foreground">○</span>;
}

export default function ActionsPage() {
  const {
    workflowRuns, selectedRunJobs, selectedJobLogs,
    loading, ghAuthenticated, setLoginDialogOpen,
    fetchWorkflowRuns, reRunWorkflow, cancelWorkflowRun,
    fetchWorkflowJobs, fetchJobLogs, clearJobLogs,
  } = useAppStore();

  const [selectedRunID, setSelectedRunID] = useState<number | null>(null);
  const [selectedJobID, setSelectedJobID] = useState<number | null>(null);

  useEffect(() => {
    if (ghAuthenticated) {
      fetchWorkflowRuns();
    }
  }, [ghAuthenticated]);

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
      <div className="flex flex-col items-center justify-center h-64 text-muted-foreground gap-3">
        <span className="text-3xl">⚙</span>
        <p>Sign in to GitHub to view Actions</p>
        <button className="text-sm text-primary underline underline-offset-2" onClick={() => setLoginDialogOpen(true)}>
          Open Settings
        </button>
      </div>
    );
  }

  return (
    <div className="flex gap-4 h-[calc(100vh-8rem)]">
      {/* Run list */}
      <div className="w-1/2 min-w-0 flex flex-col">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-lg font-bold">Workflow Runs</h2>
          <Button variant="outline" size="sm" onClick={fetchWorkflowRuns} disabled={loading.workflowRuns}>
            Refresh
          </Button>
        </div>
        <ScrollArea className="flex-1">
          {loading.workflowRuns && (!workflowRuns || workflowRuns.length === 0) ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">Loading...</div>
          ) : !workflowRuns || workflowRuns.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">No workflow runs yet</div>
          ) : (
            <div className="space-y-1">
              {workflowRuns.map((run) => (
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

      {/* Job detail */}
      <div className="w-1/2 min-w-0 border-l pl-4 flex flex-col">
        {!selectedRunID ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            Select a workflow run to view jobs
          </div>
        ) : (
          <>
            <h3 className="text-sm font-semibold mb-2">Jobs</h3>
            <ScrollArea className="flex-1">
              {(!selectedRunJobs || selectedRunJobs.length === 0) && !selectedJobLogs ? (
                <div className="text-sm text-muted-foreground">No jobs available</div>
              ) : (
                <div className="space-y-1">
                  {(selectedRunJobs || []).map((job) => (
                    <JobCard
                      key={job.id}
                      job={job}
                      selected={selectedJobID === job.id}
                      onSelect={() => handleSelectJob(job.id)}
                    />
                  ))}
                </div>
              )}

              {/* Logs */}
              {selectedJobLogs && (
                <div className="mt-4">
                  <h4 className="text-xs font-semibold text-muted-foreground mb-1 uppercase tracking-wide">Logs</h4>
                  <pre className="text-xs font-mono bg-muted/30 rounded p-2 overflow-x-auto whitespace-pre-wrap max-h-96">
                    {selectedJobLogs}
                  </pre>
                </div>
              )}
            </ScrollArea>
          </>
        )}
      </div>
    </div>
  );
}

function RunCard({ run, selected, onSelect, onReRun, onCancel }: {
  run: WorkflowRun; selected: boolean; onSelect: () => void; onReRun: () => void; onCancel: () => void;
}) {
  const isActive = run.status === "IN_PROGRESS" || run.status === "QUEUED";
  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors text-sm",
        selected ? "bg-accent border border-border" : "hover:bg-accent/50"
      )}
      onClick={onSelect}
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium truncate">{run.workflow_name}</span>
          <StatusBadge status={run.status} conclusion={run.conclusion} />
        </div>
        <div className="flex items-center gap-2 mt-0.5 text-xs text-muted-foreground">
          <span className="font-mono">#{run.run_number}</span>
          <EventBadge event={run.event} />
          <span>{run.branch}</span>
          <span className="font-mono">{run.head_sha.slice(0, 7)}</span>
        </div>
      </div>
      <div className="flex gap-1 shrink-0">
        {isActive && (
          <Button variant="ghost" size="sm" className="h-6 px-1.5 text-xs" onClick={(e) => { e.stopPropagation(); onCancel(); }}>
            Cancel
          </Button>
        )}
        {run.conclusion === "failure" && (
          <Button variant="ghost" size="sm" className="h-6 px-1.5 text-xs" onClick={(e) => { e.stopPropagation(); onReRun(); }}>
            Re-run
          </Button>
        )}
      </div>
    </div>
  );
}

function JobCard({ job, selected, onSelect }: { job: Job; selected: boolean; onSelect: () => void }) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 rounded cursor-pointer transition-colors text-sm",
        selected ? "bg-accent" : "hover:bg-accent/50"
      )}
      onClick={onSelect}
    >
      <JobStatusIcon status={job.status} conclusion={job.conclusion} />
      <span className="flex-1">{job.name}</span>
      {job.runner_name && (
        <span className="text-xs text-muted-foreground">{job.runner_name}</span>
      )}
      {job.conclusion && (
        <Badge variant="outline" className={cn(
          "text-xs font-mono",
          job.conclusion === "success" && "text-green-500",
          job.conclusion === "failure" && "text-red-500",
          job.conclusion === "cancelled" && "text-muted-foreground",
        )}>
          {job.conclusion}
        </Badge>
      )}
    </div>
  );
}
