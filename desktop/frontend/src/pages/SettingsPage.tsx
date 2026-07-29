import { useEffect, useState } from "react";
import {
  FolderOpen, GitBranch, Globe, GitCommitHorizontal, Settings, User, Calendar,
  Save, X, Sparkles, Eye, EyeOff,
} from "lucide-react";
import { useAppStore } from "@/store/app";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import ModelSelectorPopover from "@/components/ModelSelectorPopover";

const CONFIG_KEYS = [
  { key: "user.name", label: "User Name", placeholder: "Your Name" },
  { key: "user.email", label: "User Email", placeholder: "user@example.com" },
  { key: "core.autocrlf", label: "Auto CRLF", placeholder: "true / false / input" },
  { key: "core.editor", label: "Editor", placeholder: "code --wait" },
  { key: "init.defaultBranch", label: "Default Branch", placeholder: "main" },
];

const AI_PROVIDERS = [
  { value: "openai", label: "OpenAI", models: ["gpt-4o-mini", "gpt-4o", "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"] },
  { value: "anthropic", label: "Anthropic", models: ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-haiku-20240307"] },
  { value: "deepseek", label: "DeepSeek", models: ["deepseek-chat", "deepseek-coder", "deepseek-reasoner"] },
  { value: "openrouter", label: "OpenRouter", models: ["openai/gpt-4o-mini", "openai/gpt-4o", "anthropic/claude-3.5-sonnet", "deepseek/deepseek-chat", "deepseek/deepseek-r1", "google/gemini-2.0-flash-exp"] },
  { value: "custom", label: "Custom / Ollama", models: ["llama3.2", "qwen2.5-coder", "mistral", "codestral"] },
];

export default function SettingsPage() {
  const {
    repoPath, status, currentBranch,
    repository, fetchRepository,
    ghAuthenticated, ghUser,
    gitConfig, fetchGitConfig, setGitConfig,
    aiConfig, fetchAIConfig, setAIConfigAction,
  } = useAppStore();

  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  // AI form state
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
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center gap-2">
        <Settings className="w-5 h-5" />
        <h2 className="text-xl font-bold">Repository Settings</h2>
      </div>

      {/* Repository info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <FolderOpen className="w-4 h-4 text-muted-foreground" />
            Repository
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Name:</span>
            <span className="font-medium">{repoName}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Path:</span>
            <code className="font-mono text-xs bg-muted/30 px-2 py-1 rounded truncate">
              {repoPath || "—"}
            </code>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="w-24 text-muted-foreground shrink-0">Current branch:</span>
            <Badge variant="outline" className="font-mono">
              {currentBranch || status?.branch || "—"}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Git Config */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <GitBranch className="w-4 h-4 text-muted-foreground" />
            Git Configuration
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {CONFIG_KEYS.map(({ key, label, placeholder }) => (
            <div key={key} className="flex items-center gap-3 text-sm">
              <span className="w-28 text-muted-foreground shrink-0">{label}:</span>
              {editingKey === key ? (
                <div className="flex-1 flex items-center gap-1">
                  <Input
                    className="h-7 text-xs flex-1"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(); if (e.key === "Escape") setEditingKey(null); }}
                    placeholder={placeholder}
                    autoFocus
                  />
                  <button className="p-1 text-success hover:bg-success/10 rounded" onClick={saveEdit} title="Save">
                    <Save className="w-3.5 h-3.5" />
                  </button>
                  <button className="p-1 text-muted-foreground hover:text-foreground rounded" onClick={() => setEditingKey(null)} title="Cancel">
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ) : (
                <div
                  className="flex-1 flex items-center gap-2 cursor-pointer group"
                  onClick={() => startEdit(key)}
                >
                  <code className="font-mono text-xs bg-muted/30 px-2 py-1 rounded">
                    {gitConfig[key] || <span className="italic text-muted-foreground/60">not set</span>}
                  </code>
                  <span className="text-xs text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity">
                    Click to edit
                  </span>
                </div>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      {/* AI Commit Message Generator */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-muted-foreground" />
            AI Commit Messages
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
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
                  onClick={() => {
                    setAiProvider(p.value);
                    setAiModel(p.models[0] || "");
                    setAiEndpoint(p.value === "custom" ? "https://" : "");
                  }}
                  className={cn(
                    "text-xs px-2.5 py-1.5 rounded-md border font-medium transition-colors",
                    aiProvider === p.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border text-muted-foreground hover:border-primary/50 hover:text-foreground"
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
                onChange={(e) => setAiApiKey(e.target.value)}
                placeholder={aiProvider ? `Enter your ${AI_PROVIDERS.find((p) => p.value === aiProvider)?.label || ""} API key` : "Select a provider first"}
                disabled={!aiProvider}
              />
              <button
                onClick={() => setShowApiKey(!showApiKey)}
                className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
                title={showApiKey ? "Hide API key" : "Show API key"}
              >
                {showApiKey ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Model selector */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">Model</label>
            <div className="flex items-center gap-2">
              <Input
                className="h-8 text-xs flex-1 font-mono"
                value={aiModel}
                onChange={(e) => setAiModel(e.target.value)}
                placeholder={
                  aiProvider
                    ? `e.g. ${(AI_PROVIDERS.find((p) => p.value === aiProvider)?.models[0] || "gpt-4o-mini")}`
                    : "Select a provider first"
                }
                disabled={!aiProvider}
              />
              <ModelSelectorPopover variant="settings" />
            </div>
            <p className="text-[10px] text-muted-foreground/50 mt-0.5">
              Enter any model name or use the picker for recommendations
            </p>
          </div>

          {/* Endpoint — shown for all providers, pre-filled for non-custom */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground font-medium">API Endpoint</label>
            <Input
              className="h-8 text-xs flex-1 font-mono"
              value={aiEndpoint}
              onChange={(e) => setAiEndpoint(e.target.value)}
              placeholder={
                aiProvider === "custom"
                  ? "https://api.example.com/v1/chat/completions"
                  : "Auto-detected from provider (override if needed)"
              }
              disabled={!aiProvider}
            />
          </div>

          {/* Save button */}
          <div className="flex justify-end pt-1">
            <Button
              size="sm"
              className="h-8 text-xs"
              disabled={aiSaving || !aiProvider || !aiApiKey}
              onClick={async () => {
                setAiSaving(true);
                await setAIConfigAction(aiProvider, aiApiKey, aiModel, aiEndpoint);
                setAiSaving(false);
              }}
            >
              {aiSaving ? (
                <span className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin mr-1.5" />
              ) : (
                <Save className="w-3.5 h-3.5 mr-1.5" />
              )}
              {aiConfig?.provider === aiProvider ? "Update" : "Save"} AI Config
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* GitHub info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Globe className="w-4 h-4 text-muted-foreground" />
            GitHub Connection
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {ghAuthenticated && ghUser ? (
            <>
              <div className="flex items-center gap-3">
                <img
                  src={ghUser.avatar_url}
                  alt={ghUser.login}
                  className="w-10 h-10 rounded-full"
                />
                <div>
                  <p className="font-medium">{ghUser.name || ghUser.login}</p>
                  <p className="text-sm text-muted-foreground">{ghUser.login}</p>
                </div>
              </div>
              {repository && (
                <>
                  <Separator />
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-28 text-muted-foreground shrink-0">Repository:</span>
                    <span>{repository.full_name || repository.name || "—"}</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-28 text-muted-foreground shrink-0">Default branch:</span>
                    <Badge variant="outline" className="font-mono text-xs">
                      {repository.default_branch || "—"}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <span className="w-28 text-muted-foreground shrink-0">Visibility:</span>
                    <span>{repository.is_private ? "Private" : "Public"}</span>
                  </div>
                  {repository.description && (
                    <p className="text-sm text-muted-foreground mt-2">
                      {repository.description}
                    </p>
                  )}
                </>
              )}
            </>
          ) : (
            <div className="flex items-center gap-3 py-2">
              <User className="w-8 h-8 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">Not connected</p>
                <p className="text-xs text-muted-foreground">Sign in to GitHub from the header</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* About */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <GitCommitHorizontal className="w-4 h-4 text-muted-foreground" />
            About
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>zgit — Git & GitHub desktop client</p>
          <div className="flex items-center gap-2">
            <Calendar className="w-3.5 h-3.5" />
            <span>Built with Go + React + Wails</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
