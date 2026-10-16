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
  FolderKanban, Plus, Trash2, FolderOpen, RefreshCw, X, Pencil, Check, GripVertical,
  Brush, GitBranch, GitCommitHorizontal, AlertTriangle,
} from "lucide-react";
import { useAppStore, type Project, type RepoOverview } from "@/store/app";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogClose,
} from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/EmptyState";

const PROJECT_COLORS = [
  "#60a5fa", "#a78bfa", "#f472b6", "#f87171",
  "#fb923c", "#facc15", "#4ade80", "#2dd4bf",
];

const asMemberId = (path: string) => `member::${path}`;
const asPoolId = (path: string) => `pool::${path}`;
const stripPrefix = (id: string) => id.replace(/^(member|pool)::/, "");

/** 1234 -> "1.2k", 999 -> "999" */
function fmtCount(n?: number): string {
  if (!n) return "0";
  if (n < 1000) return String(n);
  const v = n / 1000;
  return `${v >= 100 ? Math.round(v) : v.toFixed(1).replace(/\.0$/, "")}k`;
}

function timeAgo(unix?: number): string {
  if (!unix) return "";
  const s = Math.floor(Date.now() / 1000 - unix);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

export default function ProjectsPage() {
  const {
    projects, activeProjectId, dashboard, repoPath, loading,
    fetchProjects, fetchDashboard, openRepo,
    createProject, renameProject, deleteProject,
    addProjectRepo, removeProjectRepo, setProjectRepos, fetchProject,
  } = useAppStore();

  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState(PROJECT_COLORS[0]);
  const [showDelete, setShowDelete] = useState(false);
  const [showCleanup, setShowCleanup] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
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

  const memberStats = useMemo(() => {
    let dirty = 0;
    let commits = 0;
    for (const p of active?.repos || []) {
      const s = statusByPath.get(p);
      if (!s || s.error) continue;
      if (!s.is_clean) dirty++;
      commits += s.commit_count || 0;
    }
    return { dirty, commits };
  }, [active, statusByPath]);

  const handleDragStart = (e: DragStartEvent) => setActiveDrag(String(e.active.id));

  const handleDragEnd = async (e: DragEndEvent) => {
    setActiveDrag(null);
    if (!active) return;
    const { active: a, over } = e;
    if (!over) return;
    const fromPath = stripPrefix(String(a.id));

    if (String(a.id).startsWith("member::") && String(over.id).startsWith("member::")) {
      const order = active.repos.slice();
      const from = order.indexOf(fromPath);
      const to = order.indexOf(stripPrefix(String(over.id)));
      if (from >= 0 && to >= 0 && from !== to) {
        await setProjectRepos(active.id, arrayMove(order, from, to));
      }
      return;
    }
    if (String(over.id) === "project-zone" || String(over.id).startsWith("member::")) {
      if (!memberSet.has(fromPath)) {
        await addProjectRepo(active.id, fromPath);
      }
    }
  };

  const handleCreate = async () => {
    if (!newName.trim()) return;
    await createProject(newName.trim(), newColor);
    setNewName("");
  };

  const handleAddRepos = async (paths: string[]) => {
    if (!active || paths.length === 0) return;
    for (const p of paths) {
      await addProjectRepo(active.id, p);
    }
    setShowAdd(false);
  };

  const handleRename = async () => {
    if (!active || !renameName.trim() || renameName.trim() === active.name) {
      setRenaming(false);
      return;
    }
    await renameProject(active.id, renameName.trim());
    setRenaming(false);
  };

  return (
    <div className="h-full flex flex-col gap-3 min-h-0">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2 mr-auto">
          <h2 className="text-xl font-bold">Projects</h2>
          {projects.length > 0 && (
            <Badge variant="outline" className="text-xs tabular-nums">{projects.length}</Badge>
          )}
        </div>
      </div>

      {projects.length === 0 ? (
        <div className="flex-1 min-h-0 flex flex-col md:flex-row gap-4">
          <div className="w-full md:w-80 shrink-0 flex flex-col gap-3">
            <div className="rounded-xl border border-border/50 bg-card p-5">
              <div className="flex items-center gap-2.5 mb-1.5">
                <span className="flex items-center justify-center w-9 h-9 rounded-xl bg-accent text-muted-foreground">
                  <FolderKanban className="w-5 h-5" />
                </span>
                <div>
                  <p className="text-sm font-semibold leading-tight">Start your first project</p>
                  <p className="text-[11px] text-muted-foreground">One for each context you work in</p>
                </div>
              </div>
              <p className="text-xs text-muted-foreground my-3 leading-relaxed">
                Group related repositories — work, OSS, demos — then
                bulk-fetch and clean them up together.
              </p>
              <label className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
                Name
              </label>
              <input
                className="mt-1.5 flex h-10 w-full rounded-lg border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                placeholder="e.g. Work, OSS, Demos"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                autoFocus
              />
              <div className="mt-3.5 mb-1.5 flex items-center justify-between">
                <label className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
                  Color
                </label>
                <span className="text-[11px] font-mono text-muted-foreground">
                  {newName.trim() || "Preview"}
                </span>
              </div>
              <div
                className="flex items-center gap-2 rounded-lg border border-border/50 bg-background/60 px-3 py-2.5"
              >
                <span
                  className="text-sm font-semibold truncate flex-1 min-w-0"
                  style={{ color: newColor }}
                >
                  {newName.trim() || "Your project"}
                </span>
                <span
                  className="flex items-center gap-2 shrink-0"
                >
                  {PROJECT_COLORS.map((c) => {
                    const active = newColor === c;
                    return (
                      <button
                        key={c}
                        onClick={() => setNewColor(c)}
                        title={c}
                        style={{
                          width: 18,
                          height: 18,
                          borderRadius: 9999,
                          backgroundColor: c,
                          border: "none",
                          padding: 0,
                          cursor: "pointer",
                          outline: active ? `2px solid ${c}` : "none",
                          outlineOffset: 2,
                          opacity: active ? 1 : 0.55,
                        }}
                        onMouseEnter={(e) => { e.currentTarget.style.opacity = "1"; }}
                        onMouseLeave={(e) => { e.currentTarget.style.opacity = active ? "1" : "0.55"; }}
                      />
                    );
                  })}
                </span>
              </div>
              <Button
                variant="secondary"
                className="w-full h-10 mt-4 font-medium"
                onClick={handleCreate}
                disabled={!newName.trim()}
              >
                <Plus className="w-4 h-4 mr-1.5" />
                Create project
              </Button>
              <p className="text-center text-[11px] text-muted-foreground mt-2">
                press <kbd className="px-1 py-0.5 rounded bg-muted font-mono text-[10px]">Enter</kbd> to create
              </p>
            </div>
            <ol className="rounded-xl border border-border/50 p-4 space-y-3 text-xs text-muted-foreground">
              {[
                ["1", "Create a project", "One for each context you work in."],
                ["2", "Drag repos into it", "Or add them from the pool list."],
                ["3", "Fetch & clean together", "One click across all members."],
              ].map(([n, title, sub]) => (
                <li key={n} className="flex gap-2.5">
                  <span className="flex items-center justify-center w-5 h-5 rounded-full bg-accent text-[10px] font-semibold text-foreground shrink-0">{n}</span>
                  <span><span className="font-medium text-foreground">{title}</span><br />{sub}</span>
                </li>
              ))}
            </ol>
          </div>
          <div className="flex-1 min-w-0 rounded-xl border border-dashed border-border/60 bg-card/20">
            <EmptyState
              icon={<FolderKanban className="w-16 h-16" />}
              title="Group repos into projects"
              description="Bulk-fetch, cleanup branches, and open members without switching context."
              className="h-full"
            />
          </div>
        </div>
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <div className="flex flex-col xl:flex-row gap-4 flex-1 min-h-0">
            {/* Project rail */}
            <div className="w-full xl:w-72 shrink-0 flex flex-col gap-2 min-h-0">
              <ScrollArea className="max-h-44 xl:max-h-none xl:flex-1 min-h-0">
                <div className="space-y-1.5 pr-1">
                  {projects.map((p) => (
                    <ProjectRow
                      key={p.id}
                      project={p}
                      active={p.id === active?.id}
                      statusByPath={statusByPath}
                      onSelect={() => useAppStore.setState({ activeProjectId: p.id })}
                    />
                  ))}
                </div>
              </ScrollArea>
              <div className="flex gap-1.5">
                <input
                  className="flex h-8 flex-1 min-w-0 rounded-md border border-input bg-background px-2.5 text-xs ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  placeholder="New project... (Enter)"
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
              <div className="flex-1 min-w-0 flex flex-col lg:flex-row gap-4 min-h-0">
                <ProjectZone
                  project={active}
                  statusByPath={statusByPath}
                  commits={memberStats.commits}
                  dirtyCount={memberStats.dirty}
                  repoPath={repoPath}
                  renaming={renaming}
                  renameName={renameName}
                  setRenameName={setRenameName}
                  onStartRename={() => { setRenameName(active.name); setRenaming(true); }}
                  onCancelRename={() => setRenaming(false)}
                  onRename={handleRename}
                  onDelete={() => setShowDelete(true)}
                  onCleanup={() => setShowCleanup(true)}
                  onAdd={() => setShowAdd(true)}
                  onFetch={() => fetchProject(active.id)}
                  onOpen={openRepo}
                  onRemove={(path) => removeProjectRepo(active.id, path)}
                />
                {/* Pool */}
                <div className="w-64 shrink-0 hidden lg:flex flex-col min-h-0">
                  <h3 className="text-sm font-medium text-muted-foreground mb-2">
                    Repositories ({pool.length})
                  </h3>
                  <ScrollArea className="flex-1 min-h-0">
                    <div className="space-y-1 pr-1">
                      {pool.map((r) => (
                        <PoolCard key={r.path} path={r.path} name={r.name} commits={r.commit_count} />
                      ))}
                      {pool.length === 0 && (
                        <p className="text-xs text-muted-foreground">All repos are in this project.</p>
                      )}
                    </div>
                  </ScrollArea>
                  <p className="text-[11px] text-muted-foreground mt-2">
                    Drag repos in — or Add for several at once.
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

      {/* Add repos dialog */}
      {active && (
        <AddReposDialog
          open={showAdd}
          onOpenChange={setShowAdd}
          projectName={active.name}
          pool={pool}
          onAdd={handleAddRepos}
        />
      )}
    </div>
  );
}

function ProjectRow({ project, active: isActive, statusByPath, onSelect }: {
  project: Project; active: boolean;
  statusByPath: Map<string, RepoOverview>;
  onSelect: () => void;
}) {
  let dirty = 0;
  let commits = 0;
  for (const p of project.repos) {
    const s = statusByPath.get(p);
    if (!s || s.error) continue;
    if (!s.is_clean) dirty++;
    commits += s.commit_count || 0;
  }
  return (
    <button
      onClick={onSelect}
      className={cn(
        "relative flex items-center gap-2.5 w-full pl-3 pr-2.5 py-2.5 rounded-xl text-left transition-all overflow-hidden",
        isActive
          ? "bg-accent border border-border shadow-sm"
          : "hover:bg-accent/40 border border-transparent",
      )}
    >
      <span
        className="absolute left-0 top-2 bottom-2 w-1 rounded-full"
        style={{ backgroundColor: project.color || "var(--primary)", opacity: isActive ? 1 : 0.45 }}
      />
      <span className="flex-1 min-w-0">
        <span className="block text-sm font-semibold truncate leading-tight">{project.name}</span>
        <span className="mt-0.5 flex items-center gap-2 text-[11px] text-muted-foreground">
          <span>{project.repos.length} {project.repos.length === 1 ? "repo" : "repos"}</span>
          {dirty > 0 && <span className="text-yellow-500">· {dirty} dirty</span>}
          {commits > 0 && (
            <span className="flex items-center gap-0.5">
              · <GitCommitHorizontal className="w-3 h-3" />{fmtCount(commits)}
            </span>
          )}
        </span>
      </span>
      {isActive && <Check className="w-3.5 h-3.5 text-primary shrink-0" />}
    </button>
  );
}

function ProjectZone(props: {
  project: Project;
  statusByPath: Map<string, RepoOverview>;
  commits: number;
  dirtyCount: number;
  repoPath: string;
  renaming: boolean;
  renameName: string;
  setRenameName: (v: string) => void;
  onStartRename: () => void;
  onCancelRename: () => void;
  onRename: () => void;
  onDelete: () => void;
  onCleanup: () => void;
  onAdd: () => void;
  onFetch: () => void;
  onOpen: (path: string) => void;
  onRemove: (path: string) => void;
}) {
  const { project, statusByPath } = props;
  const { setNodeRef, isOver } = useDroppable({ id: "project-zone" });
  const [colorOpen, setColorOpen] = useState(false);
  const { setProjectColor } = useAppStore();
  const selectedColor = project.color || PROJECT_COLORS[0];

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex-1 min-w-0 rounded-xl border flex flex-col min-h-0 overflow-hidden isolate",
        isOver ? "border-primary/60 bg-primary/5" : "border-border/50 bg-card",
      )}
    >
      {/* Hero — flat matte, no gradient: gradients inside
          rounded+overflow containers stripe on some compositors. */}
      <div className="px-4 pt-4 pb-3 border-b border-border/40 bg-muted/20">
        <div className="flex items-center gap-2.5 min-w-0">
          <button
            onClick={() => setColorOpen((o) => !o)}
            title="Change project color"
            style={{
              width: 16,
              height: 16,
              borderRadius: 6,
              flexShrink: 0,
              backgroundColor: selectedColor,
              border: "none",
              padding: 0,
              cursor: "pointer",
              boxShadow: colorOpen ? `0 0 0 2px var(--background), 0 0 0 4px ${selectedColor}` : "0 1px 3px rgba(0,0,0,0.4)",
            }}
          />
          {props.renaming ? (
            <input
              className="h-8 rounded-md border border-input bg-background px-2.5 text-base font-bold flex-1 min-w-0"
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
            <h3 className="font-bold text-base truncate flex-1" title={project.name}>
              {project.name}
            </h3>
          )}
          <div className="flex gap-1 shrink-0">
            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={props.onAdd} title="Add one or more repositories">
              <Plus className="w-3.5 h-3.5 mr-1" />
              Add
            </Button>
            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={props.onCleanup} title="Preview and delete merged branches">
              <Brush className="w-3.5 h-3.5 mr-1" />
              Cleanup
            </Button>
            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={props.onFetch} title="Fetch all member repos">
              <RefreshCw className="w-3.5 h-3.5 mr-1" />
              Fetch
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
        {/* Color editor — inline expanding row, fully inline-styled so it
            can never degrade into unstyled stripes if a utility class
            fails to generate. */}
        {colorOpen && (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              marginTop: 10,
              padding: "8px 10px",
              borderRadius: 10,
              border: "1px solid var(--border)",
              backgroundColor: "var(--popover)",
            }}
          >
            {PROJECT_COLORS.map((c) => {
              const active = selectedColor === c;
              return (
                <button
                  key={c}
                  onClick={() => { setProjectColor(project.id, c); setColorOpen(false); }}
                  title={c}
                  style={{
                    width: 22,
                    height: 22,
                    borderRadius: 9999,
                    backgroundColor: c,
                    border: "none",
                    padding: 0,
                    cursor: "pointer",
                    outline: active ? `2px solid ${c}` : "2px solid transparent",
                    outlineOffset: 2,
                    opacity: active ? 1 : 0.65,
                  }}
                  onMouseEnter={(e) => { e.currentTarget.style.opacity = "1"; }}
                  onMouseLeave={(e) => { e.currentTarget.style.opacity = active ? "1" : "0.65"; }}
                />
              );
            })}
          </div>
        )}
        {/* Stat chips */}
        <div className="flex items-center gap-1.5 mt-2.5 flex-wrap">          <Badge variant="secondary" className="text-[11px]">
            {project.repos.length} {project.repos.length === 1 ? "repo" : "repos"}
          </Badge>
          {props.commits > 0 && (
            <Badge variant="outline" className="text-[11px] font-mono" title="Total commits across members">
              <GitCommitHorizontal className="w-3 h-3 mr-1" />
              {fmtCount(props.commits)}
            </Badge>
          )}
          {props.dirtyCount > 0 ? (
            <Badge variant="outline" className="text-[11px] text-yellow-500 border-yellow-500/30">
              {props.dirtyCount} dirty
            </Badge>
          ) : (
            project.repos.length > 0 && (
              <Badge variant="outline" className="text-[11px] text-green-500 border-green-500/30">
                all clean
              </Badge>
            )
          )}
        </div>
      </div>
      <ScrollArea className="flex-1 min-h-0">
        <div className="p-2 space-y-1">
          {project.repos.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border/60 px-4 py-10 text-center">
              <FolderOpen className="w-8 h-8 mx-auto mb-2 text-muted-foreground/50" />
              <p className="text-sm font-medium">Empty project</p>
              <p className="text-xs text-muted-foreground mt-1">
                Drag repositories here from the list on the right.
              </p>
              <Button size="sm" variant="outline" className="h-8 mt-3" onClick={props.onAdd}>
                <Plus className="w-3.5 h-3.5 mr-1.5" />
                Add repositories
              </Button>
            </div>
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
  const dead = !status || !!status.error;
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      onClick={() => { if (!current && !dead) onOpen(); }}
      className={cn(
        "group flex items-center gap-2.5 px-2.5 py-2 rounded-lg border transition-all",
        isDragging
          ? "bg-accent border-border opacity-70 shadow-lg"
          : current
            ? "border-primary/40 bg-primary/5"
            : "border-transparent hover:border-border/60 hover:bg-accent/40 cursor-pointer",
      )}
      title={dead ? `${path} (unreachable)` : path}
    >
      <span {...attributes} {...listeners}
        className="cursor-grab active:cursor-grabbing text-muted-foreground/40 hover:text-muted-foreground shrink-0 p-0.5"
        title="Drag to reorder"
        onClick={(e) => e.stopPropagation()}
      >
        <GripVertical className="w-3.5 h-3.5" />
      </span>
      <span className={cn(
        "w-2 h-2 rounded-full shrink-0",
        dead ? "bg-gray-500" : status.is_clean ? "bg-green-500" : "bg-yellow-500",
      )} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate leading-tight">
          {name}
          {current && <span className="ml-1.5 text-[10px] text-primary font-semibold">OPEN</span>}
        </p>
        <p className="text-[11px] text-muted-foreground truncate font-mono mt-0.5">
          {dead ? "unreachable" : (
            <>
              {status.branch || "HEAD"}
              {typeof status.commit_count === "number" && status.commit_count > 0 && (
                <span> · {fmtCount(status.commit_count)} commits</span>
              )}
              {status.last_message && <span> · {status.last_message}</span>}
            </>
          )}
        </p>
      </div>
      {!dead && (status.ahead > 0 || status.behind > 0) && (
        <span className="text-[11px] font-mono shrink-0 text-muted-foreground">
          {status.ahead > 0 && <span className="text-green-500">↑{status.ahead}</span>}
          {status.behind > 0 && <span className="text-red-500">↓{status.behind}</span>}
        </span>
      )}
      {!current && !dead && (
        <span className="text-[11px] text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0 hidden sm:inline">
          open →
        </span>
      )}
      <button
        onClick={(e) => { e.stopPropagation(); onRemove(); }}
        className="p-1 rounded text-muted-foreground/40 hover:text-destructive shrink-0"
        title="Remove from project"
      >
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

function PoolCard({ path, name, commits }: { path: string; name: string; commits?: number }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: asPoolId(path) });
  return (
    <div
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      style={transform ? { transform: CSS.Translate.toString(transform) } : undefined}
      className={cn(
        "px-3 py-2 rounded-lg border border-border/40 cursor-grab active:cursor-grabbing transition-colors hover:border-primary/40 hover:bg-accent/30",
        isDragging && "opacity-50",
      )}
      title={path}
    >
      <p className="text-xs font-medium truncate">{name}</p>
      <p className="text-[10px] text-muted-foreground truncate font-mono mt-0.5">
        {path}
        {typeof commits === "number" && commits > 0 && ` · ${fmtCount(commits)} commits`}
      </p>
    </div>
  );
}

/* ─── Add Repositories Dialog ─── */

function AddReposDialog({ open, onOpenChange, projectName, pool, onAdd }: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectName: string;
  pool: RepoOverview[];
  onAdd: (paths: string[]) => void;
}) {
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (open) {
      setQuery("");
      setPicked(new Set());
    }
  }, [open ]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return pool;
    return pool.filter((r) =>
      `${r.name} ${r.path}`.toLowerCase().includes(q),
    );
  }, [pool, query]);

  const toggle = (path: string) => {
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const toggleAll = () => {
    if (visible.every((r) => picked.has(r.path))) {
      setPicked((prev) => {
        const next = new Set(prev);
        for (const r of visible) next.delete(r.path);
        return next;
      });
    } else {
      setPicked((prev) => new Set([...prev, ...visible.map((r) => r.path)]));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="w-5 h-5" />
            Add to {projectName}
          </DialogTitle>
          <DialogDescription>
            Pick one repository or several — all checked repos join at once.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            placeholder="Filter repositories..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          {visible.length > 0 && (
            <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
              <Checkbox
                checked={visible.every((r) => picked.has(r.path)) && visible.length > 0}
                onChange={toggleAll}
              />
              Select all ({visible.length})
            </label>
          )}
          {visible.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">
              {pool.length === 0
                ? "Every known repo is already in this project. Scan for more on the Dashboard."
                : "No repositories match."}
            </p>
          ) : (
            <ScrollArea className="max-h-64">
              <div className="space-y-0.5 pr-2">
                {visible.map((r) => (
                  <label
                    key={r.path}
                    className="flex items-start gap-2.5 px-2.5 py-2 rounded-lg hover:bg-accent/40 cursor-pointer transition-colors"
                    title={r.path}
                  >
                    <Checkbox checked={picked.has(r.path)} onChange={() => toggle(r.path)} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{r.name}</p>
                      <p className="text-[11px] text-muted-foreground truncate font-mono mt-0.5">
                        {r.branch || "HEAD"}
                        {typeof r.commit_count === "number" && r.commit_count > 0 && (
                          <span> · {fmtCount(r.commit_count)} commits</span>
                        )}
                      </p>
                    </div>
                    {!r.is_clean && (
                      <span className="w-2 h-2 rounded-full bg-yellow-500 shrink-0 mt-1.5" title="dirty" />
                    )}
                  </label>
                ))}
              </div>
            </ScrollArea>
          )}
          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button onClick={() => onAdd([...picked])} disabled={picked.size === 0}>
              <Plus className="w-4 h-4 mr-1.5" />
              Add {picked.size > 0 ? `(${picked.size})` : ""}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
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
