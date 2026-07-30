import { useEffect, useState, useCallback } from "react";
import { useLocation } from "@tanstack/react-router";
import {
  FolderOpen, GitBranch, User, Sparkles, Eye, EyeOff,
  Save, X, Palette, Keyboard,
  Check, ChevronRight, Sun,
  LogOut, Github, Terminal, Sliders,
  RotateCcw, Monitor, ExternalLink, RefreshCw,
  Users, GitFork, BookOpen, Crown,
} from "lucide-react";
import { useAppStore, type Theme, THEMES, type GitHubUser } from "@/store/app";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { ACCENT_PRESETS, isValidHex, hexToHSL } from "@/lib/theme";

import { OpenURL } from "../../wailsjs/go/main/App";

/* ─── Settings tabs (6) ─── */
const SETTINGS_TABS = [
  { id: "profile",     label: "Profile",       Icon: User },
  { id: "general",     label: "General",       Icon: Sliders },
  { id: "appearance",  label: "Appearance",    Icon: Palette },
  { id: "git",         label: "Git & GitHub",  Icon: GitBranch },
  { id: "ai",          label: "AI Integration", Icon: Sparkles },
  { id: "shortcuts",   label: "Shortcuts",     Icon: Keyboard },
] as const;

type SettingsTab = (typeof SETTINGS_TABS)[number]["id"];

/* ─── AI providers (matches Go backend ProviderKind) ─── */
const AI_PROVIDERS = [
  { value: "openai", label: "OpenAI", modelHint: "gpt-4o-mini", fetchModels: true },
  { value: "anthropic", label: "Anthropic", modelHint: "claude-sonnet-4-20250514", fetchModels: false },
  { value: "groq", label: "Groq", modelHint: "llama-3.1-8b-instant", fetchModels: true },
  { value: "deepseek", label: "DeepSeek", modelHint: "deepseek-chat", fetchModels: false },
  { value: "openrouter", label: "OpenRouter", modelHint: "openai/gpt-4o-mini", fetchModels: true },
  { value: "ollama", label: "Ollama", modelHint: "llama3.2", fetchModels: true },
  { value: "custom", label: "Custom / Ollama", modelHint: "llama3.2", fetchModels: false },
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
  const location = useLocation();
  const initialTab = (location.state as { settingsTab?: SettingsTab })?.settingsTab;

  const {
    repoPath, currentBranch, status,
    repository, fetchRepository,
    ghAuthenticated, ghUser,
    gitConfig, fetchGitConfig, setGitConfig,
    aiConfig, fetchAIConfig, setAIConfigAction,
    theme, setTheme, logoutGitHub, setLoginDialogOpen,
    userPreferences, setAccentColor, setBrightness, setKeybinding,
    validateGitHubToken,
  } = useAppStore();

  const [activeTab, setActiveTab] = useState<SettingsTab>(initialTab || "general");

  useEffect(() => {
    fetchRepository();
    fetchGitConfig();
    fetchAIConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const repoName = repoPath ? repoPath.split("/").pop() || repoPath : "No repository";

  const tabContent = (
    <div className="flex-1 min-w-0 max-w-2xl animate-in fade-in slide-in-from-left-2 duration-200">
      {activeTab === "profile" && (
        <ProfileTab
          ghUser={ghUser}
          ghAuthenticated={ghAuthenticated}
          gitConfig={gitConfig}
          onSetConfig={setGitConfig}
          onRefresh={validateGitHubToken}
          onLogin={() => setLoginDialogOpen(true)}
          onLogout={logoutGitHub}
        />
      )}
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
          onSetConfig={setGitConfig}
        />
      )}
      {activeTab === "ai" && <AITab />}
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

const AUTOCRLF_OPTIONS = [
  { value: "input", label: "input", desc: "LF → LF (Recommended for Linux/macOS)" },
  { value: "true", label: "true", desc: "CRLF → LF on commit, LF → CRLF on checkout (Windows)" },
  { value: "false", label: "false", desc: "No line-ending conversion" },
] as const;

const EDITOR_PRESETS = [
  { label: "VS Code", value: "code --wait" },
  { label: "Zed", value: "zed --wait" },
  { label: "Neovim", value: "nvim" },
  { label: "Vim", value: "vim" },
  { label: "Nano", value: "nano" },
] as const;

function GitTab({
  gitConfig, onSetConfig,
}: {
  gitConfig: Record<string, string>;
  onSetConfig: (key: string, value: string) => Promise<void>;
}) {
  const [saving, setSaving] = useState<string | null>(null);

  const handleSet = async (key: string, value: string) => {
    setSaving(key);
    try {
      await onSetConfig(key, value);
    } finally {
      setSaving(null);
    }
  };

  const autocrlf = gitConfig["core.autocrlf"] || "";
  const editor = gitConfig["core.editor"] || "";
  const defaultBranch = gitConfig["init.defaultBranch"] || "";
  const isWindows = typeof navigator !== "undefined" && navigator.userAgent.includes("Windows");

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
        <div className="p-4 space-y-5">
          {/* core.autocrlf — OS-aware dropdown */}
          <div>
            <span className="text-xs text-muted-foreground font-medium block mb-1.5">
              Auto CRLF
            </span>
            <div className="flex flex-wrap gap-1.5">
              {AUTOCRLF_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => handleSet("core.autocrlf", autocrlf === opt.value ? "" : opt.value)}
                  className={cn(
                    "text-xs px-2.5 py-1.5 rounded-lg border font-medium transition-all press-scale",
                    saving === "core.autocrlf" && "opacity-50 pointer-events-none",
                    autocrlf === opt.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border/50 text-muted-foreground hover:border-primary/40 hover:text-foreground"
                  )}
                  title={opt.desc}
                >
                  {opt.label}
                  {autocrlf === opt.value && <Check className="w-3 h-3 inline ml-1" />}
                </button>
              ))}
              {autocrlf && !AUTOCRLF_OPTIONS.some((o) => o.value === autocrlf) && (
                <code className="text-xs px-2 py-1 rounded bg-muted/30 border border-border text-muted-foreground self-center">
                  {autocrlf}
                </code>
              )}
              <button
                onClick={() => handleSet("core.autocrlf", isWindows ? "true" : "input")}
                className="text-[10px] px-2 py-1 rounded border border-dashed border-border/40 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-colors press-scale"
                title={`Detected OS: ${isWindows ? "Windows" : "Linux/macOS"}`}
              >
                <Monitor className="w-3 h-3 inline mr-1" />
                Recommended for {isWindows ? "Windows" : "Linux/macOS"}
              </button>
            </div>
            <p className="text-[10px] text-muted-foreground/40 mt-1">
              {autocrlf ? `Current: ${autocrlf}` : "Not set — click a value above to set"}
            </p>
          </div>

          {/* core.editor — Editor Picker */}
          <div>
            <span className="text-xs text-muted-foreground font-medium block mb-1.5">
              Default Editor
            </span>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {EDITOR_PRESETS.map((preset) => (
                <button
                  key={preset.value}
                  onClick={() => handleSet("core.editor", editor === preset.value ? "" : preset.value)}
                  className={cn(
                    "text-xs px-2.5 py-1.5 rounded-lg border font-medium transition-all press-scale",
                    saving === "core.editor" && "opacity-50 pointer-events-none",
                    editor === preset.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border/50 text-muted-foreground hover:border-primary/40 hover:text-foreground"
                  )}
                >
                  {preset.label}
                  {editor === preset.value && <Check className="w-3 h-3 inline ml-1" />}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-1.5">
              <Input
                className="h-7 text-xs font-mono flex-1"
                value={editor}
                onChange={(e) => handleSet("core.editor", e.target.value)}
                placeholder="Custom editor command (e.g. /usr/local/bin/nvim)"
              />
              {editor && (
                <button
                  onClick={() => handleSet("core.editor", "")}
                  className="h-7 px-2 text-[10px] rounded border border-border/30 text-muted-foreground hover:text-foreground transition-colors"
                  title="Clear"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </div>
          </div>

          {/* init.defaultBranch */}
          <div>
            <span className="text-xs text-muted-foreground font-medium block mb-1.5">
              Default Branch Name
            </span>
            <div className="flex items-center gap-1.5">
              <Input
                className="h-7 text-xs font-mono flex-1"
                value={defaultBranch}
                onChange={(e) => handleSet("init.defaultBranch", e.target.value)}
                placeholder="main"
              />
              <button
                onClick={() => handleSet("init.defaultBranch", "main")}
                className="h-7 px-2.5 text-[10px] rounded border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-colors press-scale shrink-0"
                title='Reset default branch to "main"'
              >
                Reset to "main"
              </button>
            </div>
            <p className="text-[10px] text-muted-foreground/40 mt-1">
              {defaultBranch
                ? `Current: ${defaultBranch}`
                : "Not set — new repos will default to main"}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════ AI Integration Tab ═══════════════════════ */

function AITab() {
  const {
    aiConfig, fetchAIConfig,
    setAIConfigAction, setProviderAIConfigAction,
    deleteProviderAIConfigAction, fetchProviderModelsAction,
  } = useAppStore();

  const [selectedProvider, setSelectedProvider] = useState("");
  const [keyInput, setKeyInput] = useState("");
  const [showKeyInput, setShowKeyInput] = useState(false);
  const [aiSaving, setAiSaving] = useState(false);
  const [model, setModel] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [fetchedModels, setFetchedModels] = useState<string[]>([]);
  const [fetchingModels, setFetchingModels] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);

  // Reset local state when provider changes
  useEffect(() => {
    setKeyInput("");
    setShowKeyInput(false);
    setShowApiKey(false);
    setFetchedModels([]);

    if (!aiConfig || !selectedProvider) {
      setModel("");
      setEndpoint("");
      return;
    }

    // Load model and endpoint from per-provider or top-level config
    const providerStatus = aiConfig.providers?.find((p) => p.provider === selectedProvider);
    if (providerStatus) {
      setModel(providerStatus.model || aiConfig.model || "");
      setEndpoint(providerStatus.endpoint || aiConfig.endpoint || "");
    } else {
      setModel(aiConfig.model || "");
      setEndpoint(aiConfig.endpoint || "");
    }

    // Auto-fetch models if supported and key exists
    const prov = AI_PROVIDERS.find((p) => p.value === selectedProvider);
    if (prov?.fetchModels && providerStatus?.has_key) {
      fetchModels(selectedProvider);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProvider, aiConfig?.providers, aiConfig?.provider]);

  // Init selectedProvider from stored config on mount
  useEffect(() => {
    if (aiConfig?.provider) {
      setSelectedProvider(aiConfig.provider);
    }
  }, [aiConfig?.provider]);

  const getProviderStatus = (): { has_key: boolean; key_masked: string } | null => {
    if (!aiConfig?.providers) return null;
    const ps = aiConfig.providers.find((p) => p.provider === selectedProvider);
    return ps || null;
  };

  const fetchModels = async (provider: string, overrideKey?: string) => {
    setFetchingModels(true);
    try {
      const models = await fetchProviderModelsAction(provider, overrideKey || "");
      setFetchedModels(models || []);
    } finally {
      setFetchingModels(false);
    }
  };

  const handleSaveKey = async () => {
    if (!selectedProvider || !keyInput) return;
    setAiSaving(true);
    try {
      await setProviderAIConfigAction(selectedProvider, keyInput, model, endpoint);
      setKeyInput("");
      setShowKeyInput(false);
      // Fetch models after saving the key
      const prov = AI_PROVIDERS.find((p) => p.value === selectedProvider);
      if (prov?.fetchModels) {
        fetchModels(selectedProvider, keyInput);
      }
    } finally {
      setAiSaving(false);
    }
  };

  const handleClearKey = async () => {
    if (!selectedProvider) return;
    setAiSaving(true);
    try {
      await deleteProviderAIConfigAction(selectedProvider);
      setKeyInput("");
      setShowKeyInput(false);
      setFetchedModels([]);
    } finally {
      setAiSaving(false);
    }
  };

  const handleSetActiveProvider = async () => {
    if (!selectedProvider) return;
    setAiSaving(true);
    try {
      const status = getProviderStatus();
      await setAIConfigAction(selectedProvider, status?.has_key ? "" : "", model, endpoint);
    } finally {
      setAiSaving(false);
    }
  };

  const handleFetchModels = () => {
    if (!selectedProvider) return;
    fetchModels(selectedProvider);
  };

  const status = getProviderStatus();
  const provInfo = AI_PROVIDERS.find((p) => p.value === selectedProvider);
  const canFetchModels = provInfo?.fetchModels ?? false;

  return (
    <div className="space-y-5">
      <Section icon={<Sparkles className="w-4 h-4" />} title="AI Integration">
        <p className="text-xs text-muted-foreground">
          Each provider stores its own encrypted API key. Keys are encrypted at rest and never
          sent to the frontend in plaintext.
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
              {AI_PROVIDERS.map((p) => {
                const isActive = selectedProvider === p.value;
                const provStatus = aiConfig?.providers?.find((ps) => ps.provider === p.value);
                return (
                  <button
                    key={p.value}
                    onClick={() => {
                      setSelectedProvider(p.value);
                      if (!isActive) {
                        setShowKeyInput(false);
                        setKeyInput("");
                      }
                    }}
                    className={cn(
                      "text-xs px-2.5 py-1.5 rounded-lg border font-medium transition-all press-scale flex items-center gap-1.5",
                      isActive
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border/50 text-muted-foreground hover:border-primary/40 hover:text-foreground"
                    )}
                  >
                    <span className={cn(
                      "w-1.5 h-1.5 rounded-full shrink-0",
                      provStatus?.has_key ? "bg-success" : "bg-muted-foreground/30"
                    )} />
                    {p.label}
                  </button>
                );
              })}
            </div>
          </div>

          {selectedProvider ? (
            <>
              {/* API Key status + actions */}
              <div className="space-y-1.5">
                <Label>API Key</Label>
                {status?.has_key && !showKeyInput ? (
                  <div className="flex items-center gap-2">
                    <div className="flex-1 h-8 flex items-center px-3 rounded-lg border border-border/30 bg-muted/20 text-xs font-mono text-muted-foreground">
                      <span className="inline-flex items-center gap-2">
                        <span className="w-1.5 h-1.5 rounded-full bg-success" />
                        {status.key_masked || "••••••••••••"}
                      </span>
                    </div>
                    <button
                      onClick={() => setShowKeyInput(true)}
                      className="h-8 px-3 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-all press-scale"
                    >
                      Update Key
                    </button>
                    <button
                      onClick={handleClearKey}
                      disabled={aiSaving}
                      className="h-8 px-3 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-destructive hover:border-destructive/30 transition-all press-scale disabled:opacity-50"
                    >
                      <X className="w-3.5 h-3.5 inline mr-1" />
                      Clear
                    </button>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <p className="text-xs text-muted-foreground/60 italic flex items-center gap-1.5">
                      {showKeyInput ? "Enter a new API key below" : "No API key saved for this provider"}
                    </p>
                    <div className="flex items-center gap-1">
                      <Input
                        className="h-8 text-xs flex-1 font-mono"
                        type={showApiKey ? "text" : "password"}
                        value={keyInput}
                        onChange={(e) => setKeyInput(e.target.value)}
                        placeholder={`Enter ${provInfo?.label || ""} API key`}
                      />
                      <button
                        onClick={() => setShowApiKey(!showApiKey)}
                        className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {showApiKey ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                    <div className="flex gap-1.5">
                      <button
                        onClick={handleSaveKey}
                        disabled={aiSaving || !keyInput}
                        className="h-7 px-3 rounded-lg text-xs font-medium bg-primary text-primary-foreground hover:brightness-110 transition-all press-scale disabled:opacity-40 disabled:cursor-not-allowed"
                      >
                        {aiSaving ? (
                          <span className="inline-flex items-center gap-1.5">
                            <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                            Saving...
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5">
                            <Save className="w-3.5 h-3.5" />
                            Save Key
                          </span>
                        )}
                      </button>
                      {showKeyInput && (
                        <button
                          onClick={() => { setShowKeyInput(false); setKeyInput(""); }}
                          className="h-7 px-3 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-foreground transition-all press-scale"
                        >
                          Cancel
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>

              {/* Model */}
              <div className="space-y-1.5">
                <Label>Model</Label>
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Input
                      className="h-8 text-xs flex-1 font-mono"
                      value={model}
                      onChange={(e) => setModel(e.target.value)}
                      placeholder={provInfo?.modelHint || "Model name"}
                      list="ai-model-list"
                    />
                    {fetchedModels.length > 0 && (
                      <datalist id="ai-model-list">
                        {fetchedModels.map((m) => (
                          <option key={m} value={m} />
                        ))}
                      </datalist>
                    )}
                  </div>
                  {canFetchModels && (
                    <button
                      onClick={handleFetchModels}
                      disabled={fetchingModels || (!status?.has_key && !keyInput)}
                      className="h-8 px-2.5 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-all press-scale disabled:opacity-40"
                      title="Fetch models from provider"
                    >
                      <RefreshCw className={cn("w-3.5 h-3.5", fetchingModels && "animate-spin")} />
                    </button>
                  )}
                </div>
                {fetchedModels.length > 0 && (
                  <div className="max-h-32 overflow-y-auto flex flex-wrap gap-1">
                    {fetchedModels.slice(0, 30).map((m) => (
                      <button
                        key={m}
                        onClick={() => setModel(m)}
                        className={cn(
                          "text-[10px] px-1.5 py-0.5 rounded border transition-colors",
                          model === m
                            ? "border-primary bg-primary/10 text-primary"
                            : "border-border/30 text-muted-foreground hover:text-foreground"
                        )}
                      >
                        {m}
                      </button>
                    ))}
                    {fetchedModels.length > 30 && (
                      <span className="text-[10px] text-muted-foreground/50 self-center">
                        +{fetchedModels.length - 30} more
                      </span>
                    )}
                  </div>
                )}
                <p className="text-[10px] text-muted-foreground/50 mt-0.5">
                  {fetchedModels.length > 0
                    ? `Fetched ${fetchedModels.length} models. Click a model or type custom.`
                    : canFetchModels
                      ? "Save an API key and click refresh to fetch models."
                      : "Enter any model name supported by this provider."}
                </p>
              </div>

              {/* Endpoint */}
              <div className="space-y-1.5">
                <Label>API Endpoint</Label>
                <Input
                  className="h-8 text-xs flex-1 font-mono"
                  value={endpoint}
                  onChange={(e) => setEndpoint(e.target.value)}
                  placeholder={
                    selectedProvider === "custom"
                      ? "https://api.example.com/v1/chat/completions"
                      : "Auto-detected (override if needed)"
                  }
                />
              </div>

              {/* Set Active Provider */}
              <div className="flex justify-end pt-1">
                <button
                  onClick={handleSetActiveProvider}
                  disabled={aiSaving || !selectedProvider}
                  className="h-8 px-4 rounded-lg text-xs font-medium transition-all press-scale bg-primary text-primary-foreground hover:brightness-110 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {aiSaving ? (
                    <span className="inline-flex items-center gap-1.5">
                      <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" />
                      Saving...
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5">
                      {aiConfig?.provider === selectedProvider ? (
                        <><Check className="w-3.5 h-3.5" /> Active Provider</>
                      ) : (
                        <><Save className="w-3.5 h-3.5" /> Set as Active</>
                      )}
                    </span>
                  )}
                </button>
              </div>
            </>
          ) : (
            <div className="flex items-center justify-center py-8">
              <p className="text-xs text-muted-foreground italic">
                Select a provider above to configure
              </p>
            </div>
          )}
        </div>
      </div>

      <p className="text-[10px] text-muted-foreground/50">
        API keys are encrypted with AES-256-GCM and stored in{" "}
        <code className="text-[10px] font-mono bg-muted/30 px-1 rounded">~/.config/zgit/config.yaml</code>.
        Plaintext keys are never sent to the frontend.
      </p>
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

/* ═══════════════════════ Profile Tab ═══════════════════════ */

function ProfileTab({
  ghUser, ghAuthenticated, gitConfig,
  onSetConfig, onRefresh, onLogin, onLogout,
}: {
  ghUser: GitHubUser | null;
  ghAuthenticated: boolean;
  gitConfig: Record<string, string>;
  onSetConfig: (key: string, value: string) => Promise<void>;
  onRefresh: () => Promise<GitHubUser | null>;
  onLogin: () => void;
  onLogout: () => void;
}) {
  const [refreshing, setRefreshing] = useState(false);
  const [displayName, setDisplayName] = useState("");

  // Sync display name from ghUser or git config on mount
  useEffect(() => {
    setDisplayName(ghUser?.name || gitConfig["user.name"] || "");
  }, [ghUser?.name, gitConfig["user.name"]]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await onRefresh();
    setRefreshing(false);
  };

  const handleDisplayNameChange = async (value: string) => {
    setDisplayName(value);
    await onSetConfig("user.name", value);
  };

  // Not connected state
  if (!ghAuthenticated || !ghUser) {
    return (
      <div className="space-y-5">
        <Section icon={<User className="w-4 h-4" />} title="Profile">
          <p className="text-xs text-muted-foreground">
            Your GitHub profile and account settings.
          </p>
        </Section>
        <div className="glass rounded-xl overflow-hidden">
          <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
            <Github className="w-4 h-4 text-muted-foreground" />
            <span className="text-xs font-medium">GitHub Account</span>
          </div>
          <div className="p-8 flex flex-col items-center gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-muted/30 flex items-center justify-center">
              <User className="w-8 h-8 text-muted-foreground/50" />
            </div>
            <div>
              <p className="text-sm font-medium mb-1">Not connected</p>
              <p className="text-xs text-muted-foreground max-w-xs">
                Connect your GitHub account to view your profile, stats, and manage
                your identity.
              </p>
            </div>
            <button
              onClick={onLogin}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium bg-primary text-primary-foreground hover:brightness-110 transition-all press-scale"
            >
              <Github className="w-4 h-4" />
              Sign in with GitHub
            </button>
          </div>
        </div>
      </div>
    );
  }

  const planLabel = ghUser.plan === "pro" || ghUser.plan === "business"
    ? "Pro" : "Free";
  const isPro = ghUser.plan === "pro" || ghUser.plan === "business";
  const profileURL = `https://github.com/${ghUser.login}`;

  return (
    <div className="space-y-5 max-w-2xl">
      <Section icon={<User className="w-4 h-4" />} title="Profile">
        <p className="text-xs text-muted-foreground">
          Your GitHub profile and account settings.
        </p>
      </Section>

      {/* ── Profile Header ── */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="p-6 flex items-center gap-5">
          <img
            src={ghUser.avatar_url}
            alt={ghUser.login}
            className="w-16 h-16 rounded-full ring-2 ring-border/50 shrink-0"
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h2 className="text-lg font-semibold truncate">
                {ghUser.name || ghUser.login}
              </h2>
              <span
                className={cn(
                  "inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full font-medium",
                  isPro
                    ? "bg-amber-500/15 text-amber-500 border border-amber-500/20"
                    : "bg-muted/40 text-muted-foreground border border-border/30"
                )}
              >
                {isPro ? <Crown className="w-3 h-3" /> : null}
                {planLabel}
              </span>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">
              @{ghUser.login}
            </p>
            {ghUser.bio && (
              <p className="text-xs text-foreground/70 mt-1 line-clamp-2">
                {ghUser.bio}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-1.5 shrink-0">
            <a
              href={profileURL}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => { e.preventDefault(); OpenURL(profileURL); }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-all press-scale"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">View on GitHub</span>
            </a>
            <button
              onClick={onLogout}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs border border-border/30 text-muted-foreground hover:text-destructive hover:border-destructive/30 hover:bg-destructive/10 transition-all press-scale"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Disconnect</span>
            </button>
          </div>
        </div>
      </div>

      {/* ── Stats Grid ── */}
      <div className="grid grid-cols-3 gap-3">
        <StatCard
          icon={<Users className="w-4 h-4" />}
          label="Followers"
          value={ghUser.followers ?? 0}
        />
        <StatCard
          icon={<GitFork className="w-4 h-4" />}
          label="Following"
          value={ghUser.following ?? 0}
        />
        <StatCard
          icon={<BookOpen className="w-4 h-4" />}
          label="Public Repos"
          value={ghUser.public_repos ?? 0}
        />
      </div>

      {/* ── Identity Cards ── */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <User className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Identity</span>
        </div>
        <div className="p-4 space-y-4">
          {/* Display Name → git user.name */}
          <div>
            <span className="text-xs text-muted-foreground font-medium block mb-1.5">
              Display Name
            </span>
            <Input
              className="h-7 text-xs flex-1"
              value={displayName}
              onChange={(e) => handleDisplayNameChange(e.target.value)}
              placeholder="Your Name"
            />
            <p className="text-[10px] text-muted-foreground/50 mt-1">
              Syncs with Git config{" "}
              <code className="text-[10px] font-mono bg-muted/30 px-1 rounded">user.name</code>
            </p>
          </div>

          {/* Bio — read-only from ghUser */}
          <div>
            <span className="text-xs text-muted-foreground font-medium block mb-1.5">
              Bio
            </span>
            <p className={cn(
              "text-xs px-3 py-2 rounded-lg border min-h-[2.5rem]",
              ghUser.bio
                ? "border-border/30 text-foreground/80"
                : "border-dashed border-border/20 text-muted-foreground/40 italic"
            )}>
              {ghUser.bio || "No bio set on GitHub"}
            </p>
            {ghUser.company && (
              <p className="text-[10px] text-muted-foreground/50 mt-1">
                Works at {ghUser.company}
                {ghUser.location ? ` · ${ghUser.location}` : ""}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* ── Quick Actions ── */}
      <div className="glass rounded-xl overflow-hidden">
        <div className="px-4 py-2.5 border-b border-border/20 flex items-center gap-2">
          <Terminal className="w-4 h-4 text-muted-foreground" />
          <span className="text-xs font-medium">Quick Actions</span>
        </div>
        <div className="p-4 flex flex-wrap gap-2">
          <button
            onClick={() => OpenURL(profileURL)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border/30 text-foreground hover:bg-accent/30 hover:border-primary/40 transition-all press-scale"
          >
            <ExternalLink className="w-3.5 h-3.5" />
            View on GitHub
          </button>
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-all press-scale disabled:opacity-50"
          >
            <RefreshCw className={cn("w-3.5 h-3.5", refreshing && "animate-spin")} />
            {refreshing ? "Refreshing..." : "Refresh Data"}
          </button>
          <button
            onClick={() => OpenURL(`${profileURL}?tab=repositories`)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border/30 text-muted-foreground hover:text-foreground hover:border-primary/40 transition-all press-scale"
          >
            <BookOpen className="w-3.5 h-3.5" />
            Repositories
          </button>
        </div>
      </div>
    </div>
  );
}

/* ── Stat Card sub-component ── */
function StatCard({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="glass rounded-xl p-4 flex flex-col items-center gap-1.5 text-center">
      <span className="text-muted-foreground">{icon}</span>
      <span className="text-lg font-bold tabular-nums">{value.toLocaleString()}</span>
      <span className="text-[10px] text-muted-foreground">{label}</span>
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
