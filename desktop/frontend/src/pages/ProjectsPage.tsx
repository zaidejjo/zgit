import { useEffect, useMemo, useState } from "react";
import {
  DndContext, DragEndEvent, DragOverlay, DragStartEvent, PointerSensor,
  closestCenter, useDroppable, useDraggable, useSensor, useSensors,
} from "@dnd-kit/core";
import {
  SortableContext, verticalListSortingStrategy, useSortable, arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  FolderKanban, Plus, Trash2, FolderOpen, RefreshCw, X, Pencil, Check, GripVertical, Brush,
} from "lucide-react";
import { useAppStore, type Project, type RepoOverview } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { Checkbox } from "@/components/ui/checkbox";
import EmptyState from "@/components/EmptyState";

const asMemberId = (path: string) => `member::${path}`;
const asPoolId = (path: string) => `pool::${path}`;
const stripPrefix = (id: string) => id.replace(/^(member|pool)::/, "");

export default function ProjectsPage() {
  const {
    projects, activeProjectId, dashboard, repoPath, loading,
    fetchProjects, fetchDashboard, openRepo,
    createProject, renameProject, deleteProject,
    addProjectRepo, removeProjectRepo, setProjectRepos, fetchProject,
  } = useAppStore();

  const [newName, setNewName] = useState("");
  const [showDelete, setShowDelete] = useState(false);
  const [showCleanup, setShowCleanup] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameName, setRenameName] = useState("");
  const [activeDrag, setActiveDrag] = useState<string | null>(null);

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  useEffect(() => {
    fetchProjects();
    fetchDashboard();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const active: Project | undefined = projects.find((p) => p.id === activeProjectId) || projects[0];
  const statusByPath = useMemo(() => {
    const m = new Map<string, RepoOverview>();
    for (const r of dashboard) m.set(r.path, r);
    return m;
  }, [dashboard]);
  const memberSet = useMemo(() => new Set(active?.repos || []), [active]);
  const pool = useMemo(
    () => dashboard.filter((r) => !memberSet.has(r.path) && !r.error),
    [dashboard, memberSet],
  );

  const handleDragStart = (e: DragStartEvent) => setActiveDrag(String(e.active.id));

  const handleDragEnd = async (e: DragEndEvent) => {
    setActiveDrag(null);
    if (!active) return;
    const { active: a, over } = e;
    if (!over) return;
    const fromPath = stripPrefix(String(a.id));

    // Reorder within members.
    if (String(a.id).startsWith("member::") && String(over.id).startsWith("member::")) {
      const order = active.repos.slice();
      const from = order.indexOf(fromPath);
      const to = order.indexOf(stripPrefix(String(over.id)));
      if (from >= 0 && to >= 0 && from !== to) {
        await setProjectRepos(active.id, arrayMove(order, from, to));
      }
      return;
    }
    // Drop pool repo (or member) onto the project zone.
    if (String(over.id) === "project-zone" || String(over.id).startsWith("member::")) {
      if (!memberSet.has(fromPath)) {
        await addProjectRepo(active.id, fromPath);
      }
    }
  };

  const handleCreate = async () => {
    if (!newName.trim()) return;
    await createProject(newName.trim());
    setNewName("");
  };

  const handleRename = async () => {
    if (!active || !renameName.trim() || renameName.trim() === active.name) {
      setRenaming(false);
      return;
    }
    await renameProject(active.id, renameName.trim());
    setRenaming(false);
  };

  const dirtyCount = (active?.repos || []).filter((p) => {
    const s = statusByPath.get(p);
    return s && !s.error && !s.is_clean;
  }).length;

  return (
    <div className="h-full flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Projects</h2>
          {projects.length > 0 && (
            <Badge variant="outline" className="text-xs">{projects.length}</Badge>
          )}
        </div>
      </div>

      {projects.length === 0 ? (
        <EmptyState
          icon={<FolderKanban className="w-16 h-16" />}
          title="No projects yet"
          description="Group repositories into projects and work with them together."
          className="flex-1"
        />
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <div className="flex gap-4 flex-1 min-h-0">
            {/* Project list */}
            <div className="w-64 shrink-0 flex flex-col gap-2 min-h-0">
              <ScrollArea className="flex-1 min-h-0">
                <div className="space-y-1 pr-1">
                  {projects.map((p) => (
                    <ProjectRow
                      key={p.id}
                      project={p}
                      active={p.id === active?.id}
                      onSelect={() => useAppStore.setState({ activeProjectId: p.id })}
                    />
                  ))}
                </div>
              </ScrollArea>
              <div className="flex gap-1.5">
                <input
                  className="flex h-8 flex-1 min-w-0 rounded-md border border-input bg-background px-2.5 text-xs ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  placeholder="New project..."
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                />
                <Button size="sm" className="h-8 px-2.5" onClick={handleCreate} disabled={!newName.trim()}>
                  <Plus className="w-4 h-4" />
                </Button>
              </div>
            </div>

            {/* Active project */}
            {active && (
              <div className="flex-1 min-w-0 flex gap-4">
                <ProjectZone
                  project={active}
                  statusByPath={statusByPath}
                  dirtyCount={dirtyCount}
                  repoPath={repoPath}
                  renaming={renaming}
                  renameName={renameName}
                  setRenameName={setRenameName}
                  onStartRename={() => { setRenameName(active.name); setRenaming(true); }}
                  onCancelRename={() => setRenaming(false)}
                  onRename={handleRename}
                  onDelete={() => setShowDelete(true)}
                  onFetch={() => fetchProject(active.id)}
                  onCleanup={() => setShowCleanup(true)}
                  onOpen={openRepo}
                  onRemove={(path) => removeProjectRepo(active.id, path)}
                  onReorder={(order) => setProjectRepos(active.id, order)}
                />
                {/* Pool */}
                <div className="w-64 shrink-0 hidden lg:flex flex-col min-h-0">
                  <h3 className="text-sm font-medium text-muted-foreground mb-2">
                    Repositories ({pool.length})
                  </h3>
                  <ScrollArea className="flex-1 min-h-0">
                    <div className="space-y-1 pr-1">
                      {pool.map((r) => (
                        <PoolCard key={r.path} path={r.path} name={r.name} />
                      ))}
                      {pool.length === 0 && (
                        <p className="text-xs text-muted-foreground">All repos are in this project.</p>
                      )}
                    </div>
                  </ScrollArea>
                  <p className="text-[11px] text-muted-foreground mt-2">
                    Drag a repo into the project to add it.
                  </p>
                </div>
              </div>
            )}
          </div>

          <DragOverlay>
            {activeDrag ? (
              <div className="px-3 py-2 rounded-lg bg-accent border border-border font-mono text-xs shadow-xl max-w-[280px] truncate">
                {stripPrefix(activeDrag).split("/").pop()}
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      )}

      {/* Cleanup dialog */}
      {active && (
        <CleanupDialog
          open={showCleanup}
          onOpenChange={setShowCleanup}
          projectId={active.id}
          projectName={active.name}
        />
      )}

      {/* Delete dialog */}
      <Dialog open={showDelete} onOpenChange={setShowDelete}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <Trash2 className="w-5 h-5" />
              Delete Project
            </DialogTitle>
            <DialogDescription>
              Delete <span className="font-medium text-foreground">{active?.name}</span>?
              Member repositories stay in the app.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={() => { if (active) deleteProject(active.id); setShowDelete(false); }}
            >
              <Trash2 className="w-4 h-4 mr-1.5" />
              Delete
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ProjectRow({ project, active: isActive, onSelect }: {
  project: Project; active: boolean; onSelect: () => void;
}) {
  return (
    <button
      onClick={onSelect}
      className={cn(
        "flex items-center gap-2 w-full px-3 py-2 rounded-lg text-left transition-colors",
        isActive ? "bg-accent border border-border" : "hover:bg-accent/50",
      )}
    >
      <span
        className="w-2.5 h-2.5 rounded-full shrink-0"
        style={{ backgroundColor: project.color || "var(--primary)" }}
      />
      <span className="flex-1 min-w-0">
        <span className="block text-sm font-medium truncate">{project.name}</span>
        <span className="block text-[11px] text-muted-foreground">
          {project.repos.length} {project.repos.length === 1 ? "repo" : "repos"}
        </span>
      </span>
    </button>
  );
}

function ProjectZone(props: {
  project: Project;
  statusByPath: Map<string, RepoOverview>;
  dirtyCount: number;
  repoPath: string;
  renaming: boolean;
  renameName: string;
  setRenameName: (v: string) => void;
  onStartRename: () => void;
  onCancelRename: () => void;
  onRename: () => void;
  onDelete: () => void;
  onFetch: () => void;
  onCleanup: () => void;
  onOpen: (path: string) => void;
  onRemove: (path: string) => void;
  onReorder: (order: string[]) => void;
}) {
  const { project, statusByPath } = props;
  const { setNodeRef, isOver } = useDroppable({ id: "project-zone" });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex-1 min-w-0 border rounded-lg flex flex-col min-h-0 transition-colors",
        isOver && "border-primary/60 bg-primary/5",
      )}
    >
      <div className="flex items-center gap-2 px-4 py-3 border-b">
        {props.renaming ? (
          <input
            className="h-7 rounded-md border border-input bg-background px-2 text-sm flex-1 min-w-0"
            value={props.renameName}
            onChange={(e) => props.setRenameName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") props.onRename();
              if (e.key === "Escape") props.onCancelRename();
            }}
            autoFocus
            onBlur={props.onRename}
          />
        ) : (
          <h3 className="font-semibold text-sm truncate flex-1" title={project.name}>
            {project.name}
          </h3>
        )}
        {props.dirtyCount > 0 && (
          <Badge variant="outline" className="text-[10px] text-yellow-500 shrink-0">
            {props.dirtyCount} dirty
          </Badge>
        )}
        <div className="flex gap-1 shrink-0">
          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={props.onFetch} title="Fetch all member repos">
            <RefreshCw className="w-3.5 h-3.5 mr-1" />
            Fetch
          </Button>
          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={props.onCleanup} title="Preview and delete merged branches">
            <Brush className="w-3.5 h-3.5 mr-1" />
            Cleanup
          </Button>
          {!props.renaming && (
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={props.onStartRename} title="Rename project">
              <Pencil className="w-3.5 h-3.5" />
            </Button>
          )}
          {props.renaming && (
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={props.onRename} title="Save name">
              <Check className="w-3.5 h-3.5" />
            </Button>
          )}
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive" onClick={props.onDelete} title="Delete project">
            <Trash2 className="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>
      <ScrollArea className="flex-1 min-h-0">
        <div className="p-2 space-y-1">
          {project.repos.length === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-8">
              Drag repositories here to add them.
            </p>
          ) : (
            <SortableContext items={project.repos.map(asMemberId)} strategy={verticalListSortingStrategy}>
              {project.repos.map((path) => (
                <MemberRow
                  key={path}
                  path={path}
                  status={statusByPath.get(path)}
                  current={path === props.repoPath}
                  onOpen={() => props.onOpen(path)}
                  onRemove={() => props.onRemove(path)}
                />
              ))}
            </SortableContext>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

function MemberRow({ path, status, current, onOpen, onRemove }: {
  path: string;
  status?: RepoOverview;
  current: boolean;
  onOpen: () => void;
  onRemove: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: asMemberId(path) });
  const name = path.split("/").pop() || path;
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        "flex items-center gap-2 px-2.5 py-2 rounded-lg border border-transparent transition-colors",
        isDragging ? "bg-accent border-border opacity-70" : "hover:bg-accent/40",
        current && "border-primary/40 bg-primary/5",
      )}
    >
      <span {...attributes} {...listeners} className="cursor-grab active:cursor-grabbing text-muted-foreground/50 hover:text-muted-foreground shrink-0 p-0.5" title="Drag to reorder">
        <GripVertical className="w-3.5 h-3.5" />
      </span>
      <span className={cn(
        "w-2 h-2 rounded-full shrink-0",
        !status || status.error ? "bg-gray-500" : status.is_clean ? "bg-green-500" : "bg-yellow-500",
      )} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate" title={path}>{name}</p>
        <p className="text-[11px] text-muted-foreground truncate font-mono">
          {status && !status.error ? status.branch || "HEAD" : path}
        </p>
      </div>
      {!current && (
        <Button size="sm" variant="ghost" className="h-6 px-2 text-xs shrink-0" onClick={onOpen}>
          <FolderOpen className="w-3 h-3 mr-1" />
          Open
        </Button>
      )}
      <button
        onClick={onRemove}
        className="p-1 rounded text-muted-foreground/40 hover:text-destructive shrink-0"
        title="Remove from project"
      >
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

function PoolCard({ path, name }: { path: string; name: string }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: asPoolId(path) });
  return (
    <div
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      style={transform ? { transform: CSS.Translate.toString(transform) } : undefined}
      className={cn(
        "px-3 py-2 rounded-lg border border-border/40 cursor-grab active:cursor-grabbing transition-colors hover:bg-accent/40",
        isDragging && "opacity-50",
      )}
      title={path}
    >
      <p className="text-xs font-medium truncate">{name}</p>
      <p className="text-[10px] text-muted-foreground truncate font-mono">{path}</p>
    </div>
  );
}

/* ─── Branch Cleanup Dialog ─── */

function CleanupDialog({ open, onOpenChange, projectId, projectName }: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  projectName: string;
}) {
  const { projectCleanup, loading, fetchProjectCleanup, deleteRepoBranch, fetchDashboard } = useAppStore();
  const [base, setBase] = useState("main");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [deleting, setDeleting] = useState(false);

  const data = projectCleanup[projectId];
  const scanning = !!loading.projectCleanup;

  useEffect(() => {
    if (open) {
      setSelected(new Set());
      fetchProjectCleanup(projectId, base.trim());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, projectId]);

  const toggle = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const toggleAll = () => {
    if (!data) return;
    if (selected.size === data.candidates.length) setSelected(new Set());
    else setSelected(new Set(data.candidates.map((c) => c.repo_path + "\0" + c.branch)));
  };

  const handleDelete = async () => {
    if (!data || selected.size === 0 || deleting) return;
    setDeleting(true);
    for (const key of selected) {
      const sep = key.indexOf("\0");
      await deleteRepoBranch(key.slice(0, sep), key.slice(sep + 1));
    }
    setDeleting(false);
    setSelected(new Set());
    await fetchProjectCleanup(projectId, base.trim());
    await fetchDashboard();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Brush className="w-5 h-5" />
            Cleanup {projectName}
          </DialogTitle>
          <DialogDescription>
            Branches fully merged into the base. Protected, checked-out and base
            branches are never listed.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex gap-2">
            <input
              className="flex h-9 w-40 rounded-md border border-input bg-background px-3 text-sm font-mono ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="base (e.g. main)"
              value={base}
              onChange={(e) => setBase(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && fetchProjectCleanup(projectId, base.trim())}
            />
            <Button size="sm" onClick={() => fetchProjectCleanup(projectId, base.trim())} disabled={scanning}>
              <RefreshCw className={cn("w-4 h-4 mr-1.5", scanning && "animate-spin")} />
              {scanning ? "Scanning..." : "Scan"}
            </Button>
            {data && data.candidates.length > 0 && (
              <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer ml-auto">
                <Checkbox checked={selected.size === data.candidates.length && data.candidates.length > 0} onChange={toggleAll} />
                All
              </label>
            )}
          </div>

          {scanning && !data ? (
            <p className="text-sm text-muted-foreground text-center py-6">Scanning member repos...</p>
          ) : !data || data.candidates.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">
              Nothing to clean — no merged branches found.
              {data && data.withheld.length > 0 && ` (${data.withheld.length} protected withheld)`}
            </p>
          ) : (
            <ScrollArea className="max-h-64">
              <div className="space-y-0.5 pr-2">
                {data.candidates.map((c) => {
                  const key = c.repo_path + "\0" + c.branch;
                  return (
                    <label
                      key={key}
                      className="flex items-start gap-2.5 px-2.5 py-2 rounded-lg hover:bg-accent/40 cursor-pointer transition-colors"
                    >
                      <Checkbox checked={selected.has(key)} onChange={() => toggle(key)} />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-mono truncate">
                          <span className="text-muted-foreground">{c.repo_name}/</span>{c.branch}
                        </p>
                        {c.last_message && (
                          <p className="text-xs text-muted-foreground truncate">{c.last_message}</p>
                        )}
                      </div>
                    </label>
                  );
                })}
              </div>
            </ScrollArea>
          )}

          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button variant="outline">Close</Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={selected.size === 0 || deleting}
            >
              <Trash2 className="w-4 h-4 mr-1.5" />
              {deleting ? "Deleting..." : `Delete ${selected.size > 0 ? `(${selected.size})` : ""}`}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
