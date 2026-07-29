import { useEffect, useState, useCallback } from "react";
import {
  FolderOpen, GitBranch, Globe, User, Sparkles, Eye, EyeOff,
  Save, X, Palette, Keyboard,
  Check, ChevronRight, Sun,
  LogOut, Github, Terminal, Sliders,
  RotateCcw,
} from "lucide-react";
import { useAppStore, type Theme, THEMES, type UserPreferences } from "@/store/app";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { ACCENT_PRESETS, isValidHex, hexToHSL } from "@/lib/theme";
import ModelSelectorPopover from "@/components/ModelSelectorPopover";

/* ─── Settings tabs (5) ─── */
const SETTINGS_TABS = [
  { id: "general",     label: "General",      Icon: Sliders },
  { id: "appearance",  label: "Appearance",    Icon: Palette },
  { id: "git",         label: "Git & GitHub",  Icon: GitBranch },
  { id: "ai",          label: "AI Integration", Icon: Sparkles },
  { id: "shortcuts",   label: "Shortcuts",     Icon: Keyboard },
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

/* ─── Theme preview colours ─── */
const THEME_PREVIEW_BG: Record<Theme, string> = {
  dark:       "#0a0d14",
  catppuccin: "#1e1e2e",
  tokyonight: "#1a1b26",
  light:      "#ffffff",
  dracula:    "#282a36",
};
const THEME_PREVIEW_ACCENT: Record<Theme, string> = {
  dark:       "#22C55E",
  catppuccin: "#cba6f7",
  tokyonight: "#7dcfff",
  light:      "#3b82f6",
  dracula:    "#ff79c6",
};

export default function SettingsPage() {
  const {
    repoPath, currentBranch, status,
    repository, fetchRepository,
    ghAuthenticated, ghUser,
    gitConfig, fetchGitConfig, setGitConfig,
    aiConfig, fetchAIConfig, setAIConfigAction,
    theme, setTheme, logoutGitHub, setLoginDialogOpen,
    userPreferences, setAccentColor, setBrightness, setKeybinding,
  } = useAppStore();

  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
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

  const tabContent = (
    <div className="flex-1 min-w-0 max-w-2xl animate-in fade-in slide-in-from-left-2 duration-200">
      {activeTab === "general" && (
        <GeneralTab
          repoName={repoName}
          repoPath={repoPath}
          currentBranch={currentBranch || status?.branch}
        />
      )}
      {activeTab === "appearance" && (
        <AppearanceTab
          theme={theme}
          onChangeTheme={setTheme}
          accentColor={userPreferences?.appearance?.accent_color || ""}
          brightness={userPreferences?.appearance?.brightness ?? 50}
          onSetAccentColor={setAccentColor}
          onSetBrightness={setBrightness}
        />
      )}
      {activeTab === "git" && (
        <GitTab
          gitConfig={gitConfig}
          editingKey={editingKey}
          editValue={editValue}
          onStartEdit={startEdit}
          onEditValueChange={setEditValue}
          onSaveEdit={saveEdit}
          onCancelEdit={() => setEditingKey(null)}
          ghAuthenticated={ghAuthenticated}
          ghUser={ghUser}
          repository={repository}
          onLogout={logoutGitHub}
          onLogin={() => setLoginDialogOpen(true)}
        />
      )}
      {activeTab === "ai" && (
        <AITab
          aiConfig={aiConfig}
          aiProvider={aiProvider}
          aiApiKey={aiApiKey}
          aiModel={aiModel}
          aiEndpoint={aiEndpoint}
          showApiKey={showApiKey}
          aiSaving={aiSaving}
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
      {activeTab === "shortcuts" && (
        <ShortcutsTab
          keybindings={userPreferences?.keybindings || {}}
          onSetKeybinding={setKeybinding}
        />
      )}
    </div>
  );

  return (
    <div className="flex gap-8 h-full">
      {/* ─── Sidebar Tabs ─── */}
      <aside className="w-48 shrink-0 space-y-1">
        {SETTINGS_TABS.map((tab) => {
          const TabIcon = tab.Icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                "sidebar-item text-xs",
                isActive && "active"
              )}
            >
              <TabIcon className="w-4 h-4 shrink-0" />
              <span className="flex-1 text-left">{tab.label}</span>
              {isActive && <ChevronRight className="w-3 h-3 text-primary" />}
            </button>
          );
        })}
      </aside>

      {/* ─── Content ─── */}
      {tabContent}
    </div>
  );
}

/* ═══════════════════════ General Tab ═══════════════════════ */

function GeneralTab({
  repoName, repoPath, currentBranch,
}: {
  repoName: string; repoPath: string | null; currentBranch?: string;
}) {
  return (
    <div className="space-y-5">
      <Section icon={<Sliders className="w-4 h-4" />} title="General">
        <p className="text-xs text-muted-foreground">
          Repository and project information.
        </p>
      </Section>

      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <FolderOpen className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Current Repository</span>
        </div>
        <div className="p-4 space-y-3">
          <InfoRow label="Name" value={repoName} />
          <InfoRow label="Path" value={repoPath || "—"} mono />
          <InfoRow
            label="Branch"
            value={currentBranch || "—"}
            badge
          />
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ Appearance Tab ═══════════════════════ */

function AppearanceTab({
  theme, onChangeTheme,
  accentColor, brightness,
  onSetAccentColor, onSetBrightness,
}: {
  theme: Theme; onChangeTheme: (t: Theme) => void;
  accentColor: string;
  brightness: number;
  onSetAccentColor: (color: string) => void;
  onSetBrightness: (level: number) => void;
}) {
  const [customHex, setCustomHex] = useState(accentColor);

  useEffect(() => {
    setCustomHex(accentColor);
  }, [accentColor]);

  /* ─── Accent helpers ─── */
  const getPresetBorderColor = (hex: string) => {
    const hsl = hexToHSL(hex);
    if (!hsl) return "transparent";
    return `hsl(${hsl.h} ${hsl.s}% ${hsl.l}%)`;
  };

  /* ─── Brightness helpers ─── */
  const brightnessLabel = brightness === 0 ? "Max Contrast"
    : brightness === 25 ? "High Contrast"
    : brightness === 50 ? "Default"
    : brightness === 75 ? "Low Contrast"
    : brightness === 100 ? "Min Contrast"
    : brightness < 50 ? "Darker" : "Lighter";

  return (
    <div className="space-y-5">
      <Section icon={<Palette className="w-4 h-4" />} title="Appearance">
        <p className="text-xs text-muted-foreground">
          Choose your colour theme, accent, and background brightness.
        </p>
      </Section>

      {/* ── Theme Selector (existing) ── */}
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
                  ? "border-primary bg-primary/5 shadow-glow"
                  : "border-border/40 hover:border-border/80 hover:bg-accent/20"
              )}
            >
              <div
                className="w-full h-16 rounded-lg border border-border/30 flex items-end p-2"
                style={{ background: THEME_PREVIEW_BG[t.id] }}
              >
                <div
                  className="w-4 h-4 rounded-full"
                  style={{ background: THEME_PREVIEW_ACCENT[t.id] }}
                />
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-medium">{t.label}</span>
                {isActive && <Check className="w-3 h-3 text-primary" />}
              </div>
            </button>
          );
        })}
      </div>

      {/* ── Accent Colour ── */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Palette className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Accent Colour</span>
          {accentColor && (
            <button
              onClick={() => onSetAccentColor("")}
              className="ml-auto text-[10px] px-2 py-0.5 rounded text-muted-foreground hover:text-foreground hover:bg-accent/30 transition-colors"
              title="Reset to theme default"
            >
              <RotateCcw className="w-3 h-3" />
            </button>
          )}
        </div>
        <div className="p-4 space-y-3">
          {/* Preset dots */}
          <div className="flex flex-wrap gap-2">
            {ACCENT_PRESETS.map((p) => (
              <button
                key={p.hex}
                onClick={() => onSetAccentColor(p.hex)}
                className={cn(
                  "w-7 h-7 rounded-full ring-1 ring-offset-1 ring-offset-background transition-all duration-150 press-scale",
                  accentColor === p.hex
                    ? "ring-2 ring-primary scale-110"
                    : "ring-border/40 hover:ring-primary/60"
                )}
                style={{ background: p.hex }}
                title={p.name}
              />
            ))}
          </div>

          {/* Custom HEX input */}
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground font-mono w-8">HEX</span>
            <div className="flex items-center gap-1.5 flex-1">
              <Input
                className="h-7 text-xs font-mono flex-1"
                placeholder="#22C55E"
                value={customHex}
                onChange={(e) => {
                  const v = e.target.value;
                  setCustomHex(v);
                  if (isValidHex(v)) {
                    onSetAccentColor(v);
                  }
                }}
                onBlur={() => {
                  if (customHex && !isValidHex(customHex)) {
                    setCustomHex(accentColor);
                  }
                }}
              />
              {customHex && isValidHex(customHex) && (
                <div
                  className="w-7 h-7 rounded-full shrink-0 border border-border/30"
                  style={{ background: customHex }}
                />
              )}
            </div>
          </div>
        </div>
      </div>

      {/* ── Brightness / Tone ── */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Sun className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Brightness / Tone</span>
        </div>
        <div className="p-4 space-y-2">
          <div className="flex items-center gap-3">
            <span className="text-[10px] text-muted-foreground w-6 text-right">0</span>
            <input
              type="range"
              min="0"
              max="100"
              value={brightness}
              onChange={(e) => onSetBrightness(Number(e.target.value))}
              className="flex-1 h-1.5 appearance-none rounded-full bg-muted/50 accent-primary cursor-pointer
                [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3.5 [&::-webkit-slider-thumb]:h-3.5
                [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-primary
                [&::-webkit-slider-thumb]:shadow-md [&::-webkit-slider-thumb]:cursor-pointer
                [&::-webkit-slider-thumb]:transition-transform [&::-webkit-slider-thumb]:active:scale-125"
            />
            <span className="text-[10px] text-muted-foreground w-6">100</span>
          </div>
          <div className="flex items-center justify-between text-[10px] text-muted-foreground">
            <span>Darker</span>
            <span className="text-xs font-medium text-foreground">{brightnessLabel}</span>
            <span>Lighter</span>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ Git & GitHub Tab ═══════════════════════ */

function GitTab({
  gitConfig, editingKey, editValue,
  onStartEdit, onEditValueChange, onSaveEdit, onCancelEdit,
  ghAuthenticated, ghUser, repository,
  onLogout, onLogin,
}: {
  gitConfig: Record<string, string>;
  editingKey: string | null;
  editValue: string;
  onStartEdit: (key: string) => void;
  onEditValueChange: (v: string) => void;
  onSaveEdit: () => void;
  onCancelEdit: () => void;
  ghAuthenticated: boolean;
  ghUser: any;
  repository: any;
  onLogout: () => void;
  onLogin: () => void;
}) {
  return (
    <div className="space-y-5">
      <Section icon={<GitBranch className="w-4 h-4" />} title="Git & GitHub">
        <p className="text-xs text-muted-foreground">
          Git configuration and GitHub authentication.
        </p>
      </Section>

      {/* Git Config */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Terminal className="w-4 h-4 text-muted-foreground" />
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
                    onKeyDown={(e) => {
                      if (e.key === "Enter") onSaveEdit();
                      if (e.key === "Escape") onCancelEdit();
                    }}
                    placeholder={placeholder}
                    autoFocus
                  />
                  <button className="p-1 text-success hover:bg-success/10 rounded transition-colors press-scale" onClick={onSaveEdit} title="Save">
                    <Save className="w-3.5 h-3.5" />
                  </button>
                  <button className="p-1 text-muted-foreground hover:text-foreground rounded transition-colors press-scale" onClick={onCancelEdit} title="Cancel">
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ) : (
                <div
                  className="flex-1 flex items-center gap-2 cursor-pointer group"
                  onClick={() => onStartEdit(key)}
                >
                  <code className="font-mono text-xs bg-muted/30 px-2 py-1 rounded border border-border/20 group-hover:border-border/60 transition-colors">
                    {gitConfig[key] || (
                      <span className="italic text-muted-foreground/50">not set</span>
                    )}
                  </code>
                  <span className="text-[10px] text-muted-foreground/40 opacity-0 group-hover:opacity-100 transition-opacity">
                    Click to edit
                  </span>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* GitHub Connection */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Github className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">GitHub Connection</span>
        </div>
        <div className="p-4">
          {ghAuthenticated && ghUser ? (
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <img
                  src={ghUser.avatar_url}
                  alt={ghUser.login}
                  className="w-9 h-9 rounded-full ring-1 ring-border"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{ghUser.name || ghUser.login}</p>
                  <p className="text-xs text-muted-foreground">{ghUser.login}</p>
                </div>
                <button
                  onClick={onLogout}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all press-scale"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  Disconnect
                </button>
              </div>
              {repository && (
                <>
                  <div className="h-px bg-border/20" />
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <span className="text-muted-foreground">Repository</span>
                      <p className="font-medium truncate">
                        {repository.full_name || repository.name || "—"}
                      </p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Default branch</span>
                      <p className="font-mono">{repository.default_branch || "—"}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Visibility</span>
                      <p>{repository.is_private ? "Private" : "Public"}</p>
                    </div>
                  </div>
                  {repository.description && (
                    <p className="text-xs text-muted-foreground mt-2">
                      {repository.description}
                    </p>
                  )}
                </>
              )}
            </div>
          ) : (
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <User className="w-6 h-6 text-muted-foreground" />
                <div>
                  <p className="text-xs font-medium">Not connected</p>
                  <p className="text-[11px] text-muted-foreground">
                    Connect to GitHub for PRs, issues & more
                  </p>
                </div>
              </div>
              <button
                onClick={onLogin}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-primary text-primary-foreground hover:brightness-110 transition-all press-scale"
              >
                <Github className="w-3.5 h-3.5" />
                Sign In
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ AI Integration Tab ═══════════════════════ */

function AITab({
  aiConfig, aiProvider, aiApiKey, aiModel, aiEndpoint,
  showApiKey, aiSaving,
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
  onProviderChange: (provider: string, model: string, endpoint: string) => void;
  onApiKeyChange: (key: string) => void;
  onModelChange: (model: string) => void;
  onEndpointChange: (endpoint: string) => void;
  onToggleShowKey: () => void;
  onSave: () => Promise<void>;
}) {
  return (
    <div className="space-y-5">
      <Section icon={<Sparkles className="w-4 h-4" />} title="AI Integration">
        <p className="text-xs text-muted-foreground">
          Configure an AI provider to generate commit messages and assist with Git workflows.
        </p>
      </Section>

      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Terminal className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Provider Configuration</span>
        </div>
        <div className="p-4 space-y-4">
          {/* Provider chips */}
          <div className="space-y-1.5">
            <Label>Provider</Label>
            <div className="flex flex-wrap gap-1.5">
              {AI_PROVIDERS.map((p) => (
                <button
                  key={p.value}
                  onClick={() => onProviderChange(p.value, p.models[0] || "", p.value === "custom" ? "https://" : "")}
                  className={cn(
                    "text-xs px-2.5 py-1.5 rounded-lg border font-medium transition-all press-scale",
                    aiProvider === p.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border/50 text-muted-foreground hover:border-primary/40 hover:text-foreground"
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          {/* API Key */}
          <div className="space-y-1.5">
            <Label>API Key</Label>
            <div className="flex items-center gap-1">
              <Input
                className="h-8 text-xs flex-1 font-mono"
                type={showApiKey ? "text" : "password"}
                value={aiApiKey}
                onChange={(e) => onApiKeyChange(e.target.value)}
                placeholder={aiProvider ? `Enter ${AI_PROVIDERS.find((p) => p.value === aiProvider)?.label} key` : "Select provider first"}
                disabled={!aiProvider}
              />
              <button
                onClick={onToggleShowKey}
                className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
              >
                {showApiKey ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Model */}
          <div className="space-y-1.5">
            <Label>Model</Label>
            <div className="flex items-center gap-2">
              <Input
                className="h-8 text-xs flex-1 font-mono"
                value={aiModel}
                onChange={(e) => onModelChange(e.target.value)}
                placeholder={aiProvider ? `e.g. ${AI_PROVIDERS.find((p) => p.value === aiProvider)?.models[0] || "gpt-4o-mini"}` : "Select provider first"}
                disabled={!aiProvider}
              />
              <ModelSelectorPopover variant="settings" />
            </div>
            <p className="text-[10px] text-muted-foreground/50 mt-0.5">
              Enter any model name or use the picker.
            </p>
          </div>

          {/* Endpoint */}
          <div className="space-y-1.5">
            <Label>API Endpoint</Label>
            <Input
              className="h-8 text-xs flex-1 font-mono"
              value={aiEndpoint}
              onChange={(e) => onEndpointChange(e.target.value)}
              placeholder={aiProvider === "custom" ? "https://api.example.com/v1/chat/completions" : "Auto-detected (override if needed)"}
              disabled={!aiProvider}
            />
          </div>

          {/* Save */}
          <div className="flex justify-end pt-1">
            <button
              disabled={aiSaving || !aiProvider || !aiApiKey}
              onClick={onSave}
              className="h-8 px-4 rounded-lg text-xs font-medium transition-all press-scale bg-primary text-primary-foreground hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {aiSaving ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                  Saving...
                </span>
              ) : (
                <span className="inline-flex items-center gap-1.5">
                  <Save className="w-3.5 h-3.5" />
                  {aiConfig?.provider === aiProvider ? "Update" : "Save"} Config
                </span>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ Shortcuts Tab ═══════════════════════ */

const KEYBINDING_DEFS: { id: string; action: string; defaultKeys: string }[] = [
  { id: "command_palette",  action: "Command Palette",         defaultKeys: "Ctrl+K" },
  { id: "toggle_ai_panel",  action: "Toggle AI Panel",         defaultKeys: "Ctrl+Shift+A" },
  { id: "fullscreen_ai",    action: "Fullscreen AI Chat",      defaultKeys: "Ctrl+Shift+F" },
  { id: "commit",           action: "Commit (in summary field)", defaultKeys: "Enter" },
  { id: "close_dialog",     action: "Close dialogs / palettes", defaultKeys: "Escape" },
];

function ShortcutsTab({
  keybindings, onSetKeybinding,
}: {
  keybindings: Record<string, string>;
  onSetKeybinding: (id: string, keys: string) => void;
}) {
  const [recording, setRecording] = useState<string | null>(null);

  const handleStartRecord = useCallback((id: string) => {
    setRecording(id);
  }, []);

  // Global keydown listener when recording — captures combo
  useEffect(() => {
    if (!recording) return;
    const handler = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      e.stopImmediatePropagation();

      if (e.key === "Control" || e.key === "Alt" || e.key === "Shift" || e.key === "Meta") return;

      const parts: string[] = [];
      if (e.ctrlKey || e.metaKey) parts.push("Ctrl");
      if (e.altKey) parts.push("Alt");
      if (e.shiftKey) parts.push("Shift");

      let keyLabel = e.key;
      if (keyLabel === " ") keyLabel = "Space";
      else if (keyLabel.length === 1) keyLabel = keyLabel.toUpperCase();
      else if (keyLabel === "Escape") keyLabel = "Esc";
      else if (keyLabel.startsWith("Arrow")) keyLabel = keyLabel.replace("Arrow", "↑");
      else if (keyLabel === "Enter") { /* keep */ }

      if (!parts.includes(keyLabel)) {
        parts.push(keyLabel);
      }

      const combo = parts.join("+");
      onSetKeybinding(recording, combo);
      setRecording(null);
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, [recording, onSetKeybinding]);

  const getDisplayKeys = (id: string): string => {
    return keybindings[id] || KEYBINDING_DEFS.find((d) => d.id === id)?.defaultKeys || "";
  };

  return (
    <div className="space-y-5">
      <Section icon={<Keyboard className="w-4 h-4" />} title="Keyboard Shortcuts">
        <p className="text-xs text-muted-foreground">
          Click a shortcut to rebind it. Press your new key combination to save.
        </p>
      </Section>

      <div className="glass rounded-xl overflow-hidden">
        <div className="divide-y divide-border/10">
          {KEYBINDING_DEFS.map((def) => {
            const isRecording = recording === def.id;
            const displayKeys = getDisplayKeys(def.id);
            return (
              <div key={def.id} className="flex items-center justify-between px-4 py-3">
                <span className="text-xs text-foreground">{def.action}</span>
                <button
                  onClick={() => handleStartRecord(def.id)}
                  className={cn(
                    "text-[10px] font-mono px-2 py-1 rounded-md border transition-all press-scale min-w-[80px] text-center",
                    isRecording
                      ? "border-primary bg-primary/10 text-primary ring-1 ring-primary animate-pulse"
                      : "bg-muted/40 text-muted-foreground border-border/30 hover:border-primary/50 hover:text-foreground"
                  )}
                  title="Click to rebind"
                >
                  {isRecording ? (
                    <span className="inline-flex items-center gap-1">
                      <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                      Press keys...
                    </span>
                  ) : (
                    displayKeys
                  )}
                </button>
              </div>
            );
          })}
        </div>
      </div>

      <p className="text-[10px] text-muted-foreground/50">
        Changes are saved automatically. Defaults restore on next app start if not customized.
      </p>
    </div>
  );
}

/* ═══════════════════════ Shared Components ═══════════════════════ */

function Section({ icon, title, children }: { icon: React.ReactNode; title: string; children?: React.ReactNode }) {
  return (
    <div>
      <div className="flex items-center gap-2 mb-1">
        <span className="text-primary">{icon}</span>
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      {children}
    </div>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return <span className="text-xs text-muted-foreground font-medium block">{children}</span>;
}

function InfoRow({ label, value, mono, badge }: { label: string; value: string; mono?: boolean; badge?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <span className="w-24 text-muted-foreground shrink-0 text-xs">{label}</span>
      {badge ? (
        <span className="text-xs font-mono px-2 py-0.5 rounded-md bg-primary/5 border border-primary/10 text-primary font-medium">
          {value}
        </span>
      ) : (
        <code className={cn(
          "text-xs bg-muted/15 px-2 py-1 rounded truncate max-w-[280px] border border-border/10",
          mono ? "font-mono" : "font-sans"
        )}>
          {value}
        </code>
      )}
    </div>
  );
}
