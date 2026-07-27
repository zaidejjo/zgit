import { useAppStore } from "@/store/app";
import { cn } from "@/lib/utils";
import { MessageCircle, Bot } from "lucide-react";

export default function ModeToggle() {
  const { aiMode, setAIMode } = useAppStore();

  return (
    <div className="flex items-center rounded-lg border border-border bg-muted/30 p-0.5 gap-0">
      <button
        onClick={() => setAIMode("ask")}
        className={cn(
          "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all duration-150",
          aiMode === "ask"
            ? "bg-background text-foreground shadow-sm border border-border/50"
            : "text-muted-foreground hover:text-foreground border border-transparent"
        )}
      >
        <MessageCircle className="w-3.5 h-3.5" />
        Ask
      </button>
      <button
        onClick={() => setAIMode("agent")}
        className={cn(
          "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all duration-150",
          aiMode === "agent"
            ? "bg-background text-foreground shadow-sm border border-border/50"
            : "text-muted-foreground hover:text-foreground border border-transparent"
        )}
      >
        <Bot className="w-3.5 h-3.5" />
        Agent
      </button>
    </div>
  );
}
