import { useEffect, useState } from "react";
import { Tag, Plus, Trash2 } from "lucide-react";
import { useAppStore } from "@/store/app";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import EmptyState from "@/components/EmptyState";

export default function TagsPage() {
  const { tags, fetchTags, createTag, deleteTag } = useAppStore();

  const [addOpen, setAddOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newTarget, setNewTarget] = useState("");
  const [annotated, setAnnotated] = useState(false);
  const [newMessage, setNewMessage] = useState("");

  useEffect(() => {
    fetchTags();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleAdd = async () => {
    if (!newName.trim()) return;
    await createTag(newName.trim(), newTarget.trim(), annotated ? newMessage.trim() : "");
    setNewName("");
    setNewTarget("");
    setNewMessage("");
    setAnnotated(false);
    setAddOpen(false);
  };

  const handleDelete = async (name: string) => {
    await deleteTag(name);
  };

  return (
    <div className="h-full flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-bold">Tags</h2>
          <span className="text-sm text-muted-foreground">
            {tags.length > 0 ? `${tags.length} tag${tags.length !== 1 ? "s" : ""}` : ""}
          </span>
        </div>
        <Button size="sm" onClick={() => setAddOpen(true)}>
          <Plus className="w-4 h-4 mr-1.5" />
          Create Tag
        </Button>
      </div>

      {/* Tag list */}
      <ScrollArea className="flex-1">
        {tags.length === 0 ? (
          <EmptyState
            icon={<Tag className="w-16 h-16" />}
            title="No tags yet"
            description="Create an annotated tag to mark a release."
            action={{ label: "Create Tag", onClick: () => setAddOpen(true) }}
            className="flex-1"
          />
        ) : (
          <div className="space-y-1">
            {tags.map((tag) => (
              <div
                key={tag}
                className="group flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-accent/50 transition-colors"
              >
                <Tag className="w-4 h-4 text-muted-foreground shrink-0" />
                <span className="font-mono text-sm flex-1">{tag}</span>
                <button
                  className="opacity-0 group-hover:opacity-100 p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all"
                  onClick={() => handleDelete(tag)}
                  title="Delete tag"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}
      </ScrollArea>

      {/* Create tag dialog */}
      <Dialog open={addOpen} onOpenChange={(open) => { if (!open) { setAddOpen(false); setNewName(""); setNewMessage(""); setNewTarget(""); setAnnotated(false); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Plus className="w-4 h-4" /> Create Tag
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">Tag name</label>
              <Input
                placeholder="e.g. v1.0.0"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleAdd(); }}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">
                Target <span className="text-muted-foreground">(optional, default: HEAD)</span>
              </label>
              <Input
                placeholder="Commit hash, branch name, or ref"
                value={newTarget}
                onChange={(e) => setNewTarget(e.target.value)}
              />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="annotated"
                checked={annotated}
                onChange={() => setAnnotated(!annotated)}
              />
              <label htmlFor="annotated" className="text-sm cursor-pointer">
                Annotated tag (add message)
              </label>
            </div>
            {annotated && (
              <div className="space-y-2">
                <label className="text-sm font-medium">Message</label>
                <textarea
                  className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
                  placeholder="Tag message"
                  rows={3}
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                />
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setAddOpen(false)}>Cancel</Button>
              <Button onClick={handleAdd} disabled={!newName.trim()}>Create Tag</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
