import { useEffect, useState } from "react";
import {
  FolderOpen, GitBranch, Globe, User, Sparkles, Eye, EyeOff,
  Save, X, Palette, Keyboard,
  Check, ChevronRight,
} from "lucide-react";
import { useAppStore, type Theme, THEMES } from "@/store/app";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import ModelSelectorPopover from "@/components/ModelSelectorPopover";

/* ─── Settings tabs ─── */
const SETTINGS_TABS = [
  { id: "appearance", label: "Appearance", Icon: Palette },
  { id: "git",        label: "Git Config", Icon: GitBranch },
  { id: "ai",         label: "AI & Integrations", Icon: Sparkles },
  { id: "keybindings", label: "Keybindings", Icon: Keyboard },
] as const;

type SettingsTab = (typeof SETTINGS_TABS)[number]["id"];

/* ─── Git config keys ─── */
const CONFIG_KEYS = [
  { key: "user.name", label: "User Name", placeholder: "Your Name" },
  { key: "user.email", label: "User Email", placeholder: "user@example.com" },
  { key: "core.autocrlf", label: "Auto CRLF", placeholder: "true / false / input" },
  { key: "core.editor", label: "Editor", placeholder: "code --wait" },
  { key: "init.defaultBranch", label: "Default Branch", placeholder: "main" },
];

/* ─── AI providers ─── */
const AI_PROVIDERS = [
  { value: "openai", label: "OpenAI", models: ["gpt-4o-mini", "gpt-4o", "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"] },
  { value: "anthropic", label: "Anthropic", models: ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-haiku-20240307"] },
  { value: "deepseek", label: "DeepSeek", models: ["deepseek-chat", "deepseek-coder", "deepseek-reasoner"] },
  { value: "openrouter", label: "OpenRouter", models: ["openai/gpt-4o-mini", "openai/gpt-4o", "anthropic/claude-3.5-sonnet", "deepseek/deepseek-chat", "deepseek/deepseek-r1", "google/gemini-2.0-flash-exp"] },
  { value: "custom", label: "Custom / Ollama", models: ["llama3.2", "qwen2.5-coder", "mistral", "codestral"] },
];

/* ─── Keybindings list ─── */
const KEYBINDINGS = [
  { keys: "Ctrl+K",            action: "Command Palette" },
  { keys: "Ctrl+Shift+A",      action: "Toggle AI Panel" },
  { keys: "Ctrl+Shift+F",      action: "Fullscreen AI Chat" },
  { keys: "Enter",             action: "Commit (in summary field)" },
  { keys: "Escape",            action: "Close dialogs / palettes" },
] as const;

/* ─── Theme preview colors (approximate) ─── */
const THEME_PREVIEW: Record<Theme, string> = {
  dark:       "bg-[#0F172A] border-[#22C55E]",
  catppuccin: "bg-[#1e1e2e] border-[#cba6f7]",
  tokyonight: "bg-[#1a1b26] border-[#7dcfff]",
  light:      "bg-white border-[#3b82f6]",
  dracula:    "bg-[#282a36] border-[#ff79c6]",
};

export default function SettingsPage() {
  const {
    repoPath, currentBranch, status,
    repository, fetchRepository,
    ghAuthenticated, ghUser,
    gitConfig, fetchGitConfig, setGitConfig,
    aiConfig, fetchAIConfig, setAIConfigAction,
    theme, setTheme, darkMode,
  } = useAppStore();

  const [activeTab, setActiveTab] = useState<SettingsTab>("appearance");
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");

  // AI form
  const [aiProvider, setAiProvider] = useState("");
  const [aiApiKey, setAiApiKey] = useState("");
  const [aiModel, setAiModel] = useState("");
  const [aiEndpoint, setAiEndpoint] = useState("");
  const [showApiKey, setShowApiKey] = useState(false);
  const [aiSaving, setAiSaving] = useState(false);

  useEffect(() => {
    fetchRepository();
    fetchGitConfig();
    fetchAIConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (aiConfig) {
      setAiProvider(aiConfig.provider || "");
      setAiApiKey(aiConfig.api_key || "");
      setAiModel(aiConfig.model || "");
      setAiEndpoint(aiConfig.endpoint || "");
    }
  }, [aiConfig]);

  const startEdit = (key: string) => {
    setEditingKey(key);
    setEditValue(gitConfig[key] || "");
  };

  const saveEdit = async () => {
    if (!editingKey) return;
    await setGitConfig(editingKey, editValue);
    setEditingKey(null);
  };

  const repoName = repoPath ? repoPath.split("/").pop() || repoPath : "No repository";

  return (
    <div className="flex gap-6 h-full">
      {/* ─── Sidebar Tabs ─── */}
      <aside className="w-52 shrink-0 space-y-1">
        {SETTINGS_TABS.map((tab) => {
          const TabIcon = tab.Icon;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                "sidebar-item",
                activeTab === tab.id && "active"
              )}
            >
              <TabIcon className="w-4 h-4 shrink-0" />
              <span className="flex-1 text-left">{tab.label}</span>
              {activeTab === tab.id && (
                <ChevronRight className="w-3 h-3 text-primary" />
              )}
            </button>
          );
        })}
      </aside>

      {/* ─── Content ─── */}
      <div className="flex-1 min-w-0 max-w-2xl">
        {activeTab === "appearance" && (
          <AppearanceTab
            theme={theme}
            darkMode={darkMode}
            onChangeTheme={setTheme}
          />
        )}

        {activeTab === "git" && (
          <GitConfigTab
            repoName={repoName}
            repoPath={repoPath}
            currentBranch={currentBranch || status?.branch}
            gitConfig={gitConfig}
            editingKey={editingKey}
            editValue={editValue}
            onStartEdit={startEdit}
            onEditValueChange={setEditValue}
            onSaveEdit={saveEdit}
            onCancelEdit={() => setEditingKey(null)}
          />
        )}

        {activeTab === "ai" && (
          <AIConfigTab
            aiConfig={aiConfig}
            aiProvider={aiProvider}
            aiApiKey={aiApiKey}
            aiModel={aiModel}
            aiEndpoint={aiEndpoint}
            showApiKey={showApiKey}
            aiSaving={aiSaving}
            ghAuthenticated={ghAuthenticated}
            ghUser={ghUser}
            repository={repository}
            onProviderChange={(p, m, e) => { setAiProvider(p); setAiModel(m); setAiEndpoint(e); }}
            onApiKeyChange={setAiApiKey}
            onModelChange={setAiModel}
            onEndpointChange={setAiEndpoint}
            onToggleShowKey={() => setShowApiKey(!showApiKey)}
            onSave={async () => {
              setAiSaving(true);
              await setAIConfigAction(aiProvider, aiApiKey, aiModel, aiEndpoint);
              setAiSaving(false);
            }}
          />
        )}

        {activeTab === "keybindings" && (
          <KeybindingsTab />
        )}
      </div>
    </div>
  );
}

/* ═══════════════════════ Appearance Tab ═══════════════════════ */

function AppearanceTab({
  theme,
  darkMode,
  onChangeTheme,
}: {
  theme: Theme;
  darkMode: boolean;
  onChangeTheme: (t: Theme) => void;
}) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold flex items-center gap-2">
          <Palette className="w-4 h-4 text-primary" />
          Theme
        </h3>
        <p className="text-xs text-muted-foreground mt-1">
          Choose your color theme. Settings persist across sessions.
        </p>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {THEMES.map((t) => {
          const isActive = theme === t.id;
          return (
            <button
              key={t.id}
              onClick={() => onChangeTheme(t.id)}
              className={cn(
                "relative flex flex-col items-center gap-2 p-3 rounded-xl border-2 transition-all duration-200 press-scale",
                isActive
                  ? "border-primary bg-primary/5"
                  : "border-border/50 hover:border-border hover:bg-accent/20"
              )}
            >
              {/* Preview swatch */}
              <div
                className={cn(
                  "w-full h-16 rounded-lg border",
                  isActive ? THEME_PREVIEW[t.id] : "opacity-80"
                )}
                style={{
                  background: t.id === "dark" ? "#0F172A" :
                    t.id === "catppuccin" ? "#1e1e2e" :
                    t.id === "tokyonight" ? "#1a1b26" :
                    t.id === "light" ? "#ffffff" : "#282a36",
                }}
              />
              {/* Accent indicator */}
              <div className="flex gap-1">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{
                    background: t.id === "dark" ? "#22C55E" :
                      t.id === "catppuccin" ? "#cba6f7" :
                      t.id === "tokyonight" ? "#7dcfff" :
                      t.id === "light" ? "#3b82f6" : "#ff79c6",
                  }}
                />
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-medium">{t.icon} {t.label}</span>
                {isActive && <Check className="w-3 h-3 text-primary" />}
              </div>
            </button>
          );
        })}
      </div>

      {/* Preview info */}
      <div className="glass rounded-xl p-4 space-y-2 text-xs">
        <p className="text-foreground font-medium">Theme preview</p>
        <div className="flex gap-3">
          <div className="w-4 h-4 rounded bg-primary" />
          <span className="text-muted-foreground">Primary accent</span>
        </div>
        <div className="flex gap-3">
          <div className="w-4 h-4 rounded bg-background border border-border" />
          <span className="text-muted-foreground">Background</span>
        </div>
        <div className="flex gap-3">
          <div className="w-4 h-4 rounded bg-card border border-border" />
          <span className="text-muted-foreground">Card surface</span>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ Git Config Tab ═══════════════════════ */

function GitConfigTab({
  repoName, repoPath, currentBranch,
  gitConfig, editingKey, editValue,
  onStartEdit, onEditValueChange, onSaveEdit, onCancelEdit,
}: {
  repoName: string;
  repoPath: string | null;
  currentBranch?: string;
  gitConfig: Record<string, string>;
  editingKey: string | null;
  editValue: string;
  onStartEdit: (key: string) => void;
  onEditValueChange: (v: string) => void;
  onSaveEdit: () => void;
  onCancelEdit: () => void;
}) {
  return (
    <div className="space-y-6">
      {/* Repository info */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
          <FolderOpen className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Repository</span>
        </div>
        <div className="p-4 space-y-3">
          <InfoRow label="Name" value={repoName} />
          <InfoRow label="Path" value={repoPath || "—"} mono />
          <InfoRow label="Branch" value={currentBranch || "—"} badge />
        </div>
      </div>

      {/* Git Configuration */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
          <GitBranch className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Git Configuration</span>
        </div>
        <div className="p-4 space-y-3">
          {CONFIG_KEYS.map(({ key, label, placeholder }) => (
            <div key={key} className="flex items-center gap-3 text-sm">
              <span className="w-28 text-muted-foreground shrink-0 text-xs">{label}</span>
              {editingKey === key ? (
                <div className="flex-1 flex items-center gap-1.5">
                  <Input
                    className="h-7 text-xs flex-1"
                    value={editValue}
                    onChange={(e) => onEditValueChange(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") onSaveEdit(); if (e.key === "Escape") onCancelEdit(); }}
                    placeholder={placeholder}
                    autoFocus
                  />
                  <button className="p-1 text-success hover:bg-success/10 rounded transition-colors" onClick={onSaveEdit} title="Save">
                    <Save className="w-3.5 h-3.5" />
                  </button>
                  <button className="p-1 text-muted-foreground hover:text-foreground rounded transition-colors" onClick={onCancelEdit} title="Cancel">
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ) : (
                <div
                  className="flex-1 flex items-center gap-2 cursor-pointer group"
                  onClick={() => onStartEdit(key)}
                >
                  <code className="font-mono text-xs bg-muted/30 px-2 py-1 rounded">
                    {gitConfig[key] || <span className="italic text-muted-foreground/60">not set</span>}
                  </code>
                  <span className="text-[10px] text-muted-foreground/50 opacity-0 group-hover:opacity-100 transition-opacity">
                    Click to edit
                  </span>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ AI Config Tab ═══════════════════════ */

function AIConfigTab({
  aiConfig, aiProvider, aiApiKey, aiModel, aiEndpoint,
  showApiKey, aiSaving,
  ghAuthenticated, ghUser, repository,
  onProviderChange, onApiKeyChange, onModelChange, onEndpointChange,
  onToggleShowKey, onSave,
}: {
  aiConfig: any;
  aiProvider: string;
  aiApiKey: string;
  aiModel: string;
  aiEndpoint: string;
  showApiKey: boolean;
  aiSaving: boolean;
  ghAuthenticated: boolean;
  ghUser: any;
  repository: any;
  onProviderChange: (provider: string, model: string, endpoint: string) => void;
  onApiKeyChange: (key: string) => void;
  onModelChange: (model: string) => void;
  onEndpointChange: (endpoint: string) => void;
  onToggleShowKey: () => void;
  onSave: () => Promise<void>;
}) {
  return (
    <div className="space-y-6">
      <div className="glass rounded-xl overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
          <Sparkles className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">AI Commit Messages</span>
        </div>
        <div className="p-4 space-y-4">
          <p className="text-xs text-muted-foreground">
            Configure an AI provider to generate conventional commit messages from your staged changes.
          </p>

          {/* Provider selector */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">Provider</label>
            <div className="flex flex-wrap gap-1.5">
              {AI_PROVIDERS.map((p) => (
                <button
                  key={p.value}
                  onClick={() => onProviderChange(p.value, p.models[0] || "", p.value === "custom" ? "https://" : "")}
                  className={cn(
                    "text-xs px-2.5 py-1.5 rounded-md border font-medium transition-colors",
                    aiProvider === p.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border/60 text-muted-foreground hover:border-primary/50 hover:text-foreground"
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          {/* API Key */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">API Key</label>
            <div className="flex items-center gap-1">
              <Input
                className="h-8 text-xs flex-1 font-mono"
                type={showApiKey ? "text" : "password"}
                value={aiApiKey}
                onChange={(e) => onApiKeyChange(e.target.value)}
                placeholder={aiProvider ? `Enter your ${AI_PROVIDERS.find((p) => p.value === aiProvider)?.label || ""} API key` : "Select a provider first"}
                disabled={!aiProvider}
              />
              <button
                onClick={onToggleShowKey}
                className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
                title={showApiKey ? "Hide API key" : "Show API key"}
              >
                {showApiKey ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Model */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">Model</label>
            <div className="flex items-center gap-2">
              <Input
                className="h-8 text-xs flex-1 font-mono"
                value={aiModel}
                onChange={(e) => onModelChange(e.target.value)}
                placeholder={aiProvider ? `e.g. ${(AI_PROVIDERS.find((p) => p.value === aiProvider)?.models[0] || "gpt-4o-mini")}` : "Select a provider first"}
                disabled={!aiProvider}
              />
              <ModelSelectorPopover variant="settings" />
            </div>
            <p className="text-[10px] text-muted-foreground/50 mt-0.5">
              Enter any model name or use the picker for recommendations
            </p>
          </div>

          {/* Endpoint */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">API Endpoint</label>
            <Input
              className="h-8 text-xs flex-1 font-mono"
              value={aiEndpoint}
              onChange={(e) => onEndpointChange(e.target.value)}
              placeholder={aiProvider === "custom" ? "https://api.example.com/v1/chat/completions" : "Auto-detected from provider (override if needed)"}
              disabled={!aiProvider}
            />
          </div>

          {/* Save */}
          <div className="flex justify-end pt-1">
            <button
              disabled={aiSaving || !aiProvider || !aiApiKey}
              onClick={onSave}
              className="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-150 press-scale bg-primary text-primary-foreground hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {aiSaving ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                  Saving...
                </span>
              ) : (
                <span className="inline-flex items-center gap-1.5">
                  <Save className="w-3.5 h-3.5" />
                  {aiConfig?.provider === aiProvider ? "Update" : "Save"} AI Config
                </span>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* GitHub Connection */}
      <GitHubSection
        ghAuthenticated={ghAuthenticated}
        ghUser={ghUser}
        repository={repository}
      />
    </div>
  );
}

/* ═══════════════════════ Keybindings Tab ═══════════════════════ */

function KeybindingsTab() {
  return (
    <div className="glass rounded-xl overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
        <Keyboard className="w-4 h-4 text-muted-foreground" />
        <span className="text-xs font-medium">Keyboard Shortcuts</span>
      </div>
      <div className="divide-y divide-border/10">
        {KEYBINDINGS.map((kb) => (
          <div key={kb.keys} className="flex items-center justify-between px-4 py-3">
            <span className="text-xs text-foreground">{kb.action}</span>
            <kbd className="text-[10px] font-mono px-2 py-1 rounded-md bg-muted/40 text-muted-foreground border border-border/30">
              {kb.keys}
            </kbd>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ═══════════════════════ Shared Components ═══════════════════════ */

function InfoRow({ label, value, mono, badge }: { label: string; value: string; mono?: boolean; badge?: boolean }) {
  return (
    <div className="flex items-center gap-3 text-sm">
      <span className="w-28 text-muted-foreground shrink-0 text-xs">{label}</span>
      {badge ? (
        <span className="text-xs font-mono px-2 py-0.5 rounded-md bg-primary/5 border border-primary/10 text-primary font-medium">
          {value}
        </span>
      ) : (
        <code className={cn("text-xs bg-muted/20 px-2 py-1 rounded truncate max-w-[300px]", mono ? "font-mono" : "font-sans")}>
          {value}
        </code>
      )}
    </div>
  );
}

function GitHubSection({
  ghAuthenticated, ghUser, repository,
}: {
  ghAuthenticated: boolean;
  ghUser: any;
  repository: any;
}) {
  return (
    <div className="glass rounded-xl overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border/20">
        <Globe className="w-4 h-4 text-muted-foreground" />
        <span className="text-xs font-medium">GitHub Connection</span>
      </div>
      <div className="p-4">
        {ghAuthenticated && ghUser ? (
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <img
                src={ghUser.avatar_url}
                alt={ghUser.login}
                className="w-8 h-8 rounded-full ring-1 ring-border"
              />
              <div>
                <p className="text-xs font-medium">{ghUser.name || ghUser.login}</p>
                <p className="text-[11px] text-muted-foreground">{ghUser.login}</p>
              </div>
            </div>
            {repository && (
              <>
                <div className="h-px bg-border/30" />
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <span className="text-muted-foreground">Repository:</span>
                    <p className="font-medium truncate">{repository.full_name || repository.name || "—"}</p>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Default branch:</span>
                    <p className="font-mono">{repository.default_branch || "—"}</p>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Visibility:</span>
                    <p>{repository.is_private ? "Private" : "Public"}</p>
                  </div>
                </div>
                {repository.description && (
                  <p className="text-xs text-muted-foreground mt-2">{repository.description}</p>
                )}
              </>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-3 py-1">
            <User className="w-6 h-6 text-muted-foreground" />
            <div>
              <p className="text-xs font-medium">Not connected</p>
              <p className="text-[11px] text-muted-foreground">Sign in to GitHub from the header</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
