import { useEffect, useRef, useState } from "react";
import { useAppStore } from "@/store/app";
import { cn } from "@/lib/utils";
import {
  Plus, MessageCircle, Bot, Trash2, PencilLine, PanelRightClose,
} from "lucide-react";

export default function SessionSidebar() {
  const {
    aiSessions, aiActiveSessionId, aiMode,
    fetchSessions, createSession, deleteSession,
    renameSession, switchSession,
  } = useAppStore();

  const [collapsed, setCollapsed] = useState(false);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const editInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  useEffect(() => {
    if (creating) inputRef.current?.focus();
  }, [creating]);

  useEffect(() => {
    if (editingId) editInputRef.current?.focus();
  }, [editingId]);

  const handleCreate = async () => {
    const name = newName.trim() || `${aiMode === "agent" ? "Agent" : "Ask"} Session`;
    await createSession(name, aiMode);
    setNewName("");
    setCreating(false);
  };

  const handleRename = async (id: string) => {
    const name = editName.trim();
    if (name) {
      await renameSession(id, name);
    }
    setEditingId(null);
    setEditName("");
  };

  if (collapsed) {
    return (
      <div className="flex flex-col items-center gap-2 px-1 py-3 border-r border-border bg-muted/20">
        <button
          onClick={() => setCollapsed(false)}
          className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          title="Show sessions"
        >
          <PanelRightClose className="w-4 h-4 rotate-180" />
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col w-48 border-r border-border bg-muted/20 shrink-0">
      {/* Header */}
      <div className="flex items-center justify-between px-3 h-10 border-b border-border">
        <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
          Sessions
        </span>
        <div className="flex items-center gap-0.5">
          <button
            onClick={() => setCreating(true)}
            className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            title="New session"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setCollapsed(true)}
            className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            title="Hide sessions"
          >
            <PanelRightClose className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* New session inline input */}
      {creating && (
        <div className="px-2 py-1.5 border-b border-border">
          <input
            ref={inputRef}
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleCreate();
              if (e.key === "Escape") { setCreating(false); setNewName(""); }
            }}
            onBlur={() => {
              if (newName.trim()) handleCreate();
              else { setCreating(false); setNewName(""); }
            }}
            className="w-full h-7 px-2 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="Session name..."
          />
        </div>
      )}

      {/* Session list */}
      <div className="flex-1 overflow-y-auto py-1">
        {aiSessions.length === 0 && (
          <p className="px-3 py-4 text-[10px] text-muted-foreground/50 text-center">
            No sessions yet
          </p>
        )}
        {aiSessions.map((s) => (
          <div
            key={s.id}
            className={cn(
              "group flex items-center gap-2 px-3 py-1.5 cursor-pointer transition-colors mx-1 rounded-md",
              s.id === aiActiveSessionId
                ? "bg-primary/10 text-primary"
                : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
            )}
            onClick={() => {
              if (s.id !== aiActiveSessionId) switchSession(s.id);
            }}
          >
            {s.mode === "agent" ? (
              <Bot className="w-3 h-3 shrink-0" />
            ) : (
              <MessageCircle className="w-3 h-3 shrink-0" />
            )}

            {editingId === s.id ? (
              <input
                ref={editInputRef}
                type="text"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleRename(s.id);
                  if (e.key === "Escape") { setEditingId(null); setEditName(""); }
                }}
                onBlur={() => handleRename(s.id)}
                className="flex-1 min-w-0 h-6 px-1 text-[11px] rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-ring"
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <span className="flex-1 min-w-0 truncate text-[11px]">{s.name}</span>
            )}

            {/* Action buttons on hover */}
            {editingId !== s.id && (
              <div className="hidden group-hover:flex items-center gap-0.5 shrink-0">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setEditingId(s.id);
                    setEditName(s.name);
                  }}
                  className="p-0.5 rounded text-muted-foreground/60 hover:text-foreground transition-colors"
                  title="Rename"
                >
                  <PencilLine className="w-3 h-3" />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteSession(s.id);
                  }}
                  className="p-0.5 rounded text-muted-foreground/60 hover:text-destructive transition-colors"
                  title="Delete"
                >
                  <Trash2 className="w-3 h-3" />
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Bottom info */}
      <div className="px-3 py-2 border-t border-border">
        <p className="text-[9px] text-muted-foreground/40">
          {aiSessions.length} session{aiSessions.length !== 1 ? "s" : ""}
        </p>
      </div>
    </div>
  );
}
