import { create } from "zustand";

// Types matching the Go backend models (snake_case — matches Wails JSON serialization)
export interface StatusFile {
  path: string;
  old_path?: string;
  staged: number;   // StatusType enum
  unstaged: number; // StatusType enum
}

export interface Status {
  branch: string;
  upstream?: string;
  ahead: number;
  behind: number;
  files: StatusFile[];
  staged_count: number;
  unstaged_count: number;
  untracked_count: number;
  is_clean: boolean;
  is_merging: boolean;
  is_rebasing: boolean;
  is_cherry_pick: boolean;
  is_reverting: boolean;
  is_bisecting: boolean;
}

export interface Commit {
  hash: string;
  author: string;
  email: string;
  message: string;
  timestamp: string;
  ref_names?: string;
}

export interface Branch {
  name: string;
  full_ref: string;
  type: number;
  is_head: boolean;
  upstream?: string;
  ahead?: number;
  behind?: number;
  latest_hash?: string;
  latest_msg?: string;
}

export interface FileChange {
  type: number;
  old_path?: string;
  new_path?: string;
  additions: number;
  deletions: number;
  is_binary: boolean;
  unified_diff?: string;
}

export interface Diff {
  files: FileChange[];
  total_additions: number;
  total_deletions: number;
}

export interface PRSummary {
  number: number;
  title: string;
  state: string;
  author: string;
  created_at: string;
  updated_at: string;
  is_draft: boolean;
  mergeable: string;
  head_ref: string;
  base_ref: string;
  labels?: string[];
}

export interface Issue {
  number: number;
  title: string;
  state: string;
  author: string;
  body: string;
  created_at: string;
  updated_at: string;
  closed_at?: string;
  labels?: { name: string; color: string }[];
  assignees?: string[];
  comments: number;
  is_pull_request: boolean;
}

export interface Repo {
  path: string;
  is_bare: boolean;
  owner?: string;
  name?: string;
  full_name?: string;
  default_branch?: string;
  description?: string;
  language?: string;
  is_private: boolean;
  is_fork: boolean;
  stars: number;
  forks: number;
  open_issues: number;
  created_at?: string;
  updated_at?: string;
  html_url?: string;
  ssh_url?: string;
}

export interface GitHubUser {
  login: string;
  name?: string;
  email?: string;
  avatar_url?: string;
  bio?: string;
  company?: string;
  location?: string;
  followers: number;
  following: number;
  public_repos: number;
}

export interface DeviceFlowCode {
  device_code: string;
  user_code: string;
  verification_uri: string;
  interval: number;
}

// Wails runtime injects window.go.main.App during dev/build.
// The generated wrappers at wailsjs/go/main/App.js call the same runtime.
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetStatus(): Promise<Status>;
          GetLog(count: number): Promise<Commit[]>;
          GetBranches(): Promise<Branch[]>;
          GetDiff(pathspec: string): Promise<Diff>;
          StageFile(file: string): Promise<void>;
          UnstageFile(file: string): Promise<void>;
          StageAll(): Promise<void>;
          UnstageAll(): Promise<void>;
          Commit(message: string): Promise<string>;
          CheckoutBranch(name: string): Promise<void>;
          CreateBranch(name: string): Promise<void>;
          DeleteBranch(name: string, force: boolean): Promise<void>;
          CurrentBranch(): Promise<string>;
          IsGitHubAuthenticated(): Promise<boolean>;
          GetPullRequests(): Promise<PRSummary[]>;
          GetIssues(): Promise<Issue[]>;
          GetRepository(): Promise<Repo>;
          GetCurrentRepoPath(): Promise<string>;
          GetGitHubUser(): Promise<GitHubUser>;
          AuthenticateGitHub(token: string): Promise<void>;
          StartDeviceFlow(): Promise<DeviceFlowCode>;
          PollDeviceFlow(deviceCode: string): Promise<string>;
        };
      };
    };
    Go?: Window["go"];
  }
}

// App state store
interface AppState {
  // Git state
  status: Status | null;
  log: Commit[];
  branches: Branch[];
  diff: Diff | null;
  currentBranch: string;

  // GitHub state
  ghAuthenticated: boolean;
  ghUser: GitHubUser | null;
  pullRequests: PRSummary[];
  issues: Issue[];
  repository: Repo | null;

  // UI state
  loading: Record<string, boolean>;
  error: string | null;
  activeTab: string;
  darkMode: boolean;
  loginDialogOpen: boolean;

  // Actions
  setActiveTab: (tab: string) => void;
  toggleDarkMode: () => void;
  fetchStatus: () => Promise<void>;
  fetchLog: (count?: number) => Promise<void>;
  fetchBranches: () => Promise<void>;
  fetchDiff: (pathspec?: string) => Promise<void>;
  stageFile: (file: string) => Promise<void>;
  unstageFile: (file: string) => Promise<void>;
  stageAll: () => Promise<void>;
  unstageAll: () => Promise<void>;
  commit: (message: string) => Promise<string | null>;
  checkoutBranch: (name: string) => Promise<void>;
  createBranch: (name: string) => Promise<void>;
  deleteBranch: (name: string, force?: boolean) => Promise<void>;
  fetchPullRequests: () => Promise<void>;
  fetchIssues: () => Promise<void>;
  fetchRepository: () => Promise<void>;
  checkGitHubAuth: () => Promise<void>;
  authenticateGitHub: (token: string) => Promise<boolean>;
  startDeviceFlow: () => Promise<DeviceFlowCode | null>;
  pollDeviceFlow: (deviceCode: string) => Promise<string>;
  setLoginDialogOpen: (open: boolean) => void;
  setError: (err: string | null) => void;
  clearDiff: () => void;
}

const app = window as any;

function getBackend() {
  // During development with wails dev, the backend is at window.go.main.App
  // During standalone, we construct from window.Go.main.App
  return app.go?.main?.App || app.Go?.main?.App || null;
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  status: null,
  log: [],
  branches: [],
  diff: null,
  currentBranch: "",
  ghAuthenticated: false,
  ghUser: null,
  pullRequests: [],
  issues: [],
  repository: null,
  loading: {},
  error: null,
  activeTab: "status",
  darkMode: true,
  loginDialogOpen: false,

  setActiveTab: (tab) => set({ activeTab: tab }),

  toggleDarkMode: () => {
    const next = !get().darkMode;
    document.documentElement.classList.toggle("dark", next);
    set({ darkMode: next });
  },

  fetchStatus: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, status: true }, error: null }));
    try {
      const status = await backend.GetStatus();
      set({ status, loading: { ...get().loading, status: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch status", loading: { ...get().loading, status: false } });
    }
  },

  fetchLog: async (count = 50) => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, log: true } }));
    try {
      const log = await backend.GetLog(count);
      set({ log, loading: { ...get().loading, log: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch log", loading: { ...get().loading, log: false } });
    }
  },

  fetchBranches: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, branches: true } }));
    try {
      const [branches, currentBranch] = await Promise.all([
        backend.GetBranches(),
        backend.CurrentBranch(),
      ]);
      set({ branches, currentBranch, loading: { ...get().loading, branches: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch branches", loading: { ...get().loading, branches: false } });
    }
  },

  fetchDiff: async (pathspec = "") => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, diff: true } }));
    try {
      const diff = await backend.GetDiff(pathspec);
      set({ diff, loading: { ...get().loading, diff: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch diff", loading: { ...get().loading, diff: false } });
    }
  },

  stageFile: async (file) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StageFile(file);
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Failed to stage file" });
    }
  },

  unstageFile: async (file) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.UnstageFile(file);
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Failed to unstage file" });
    }
  },

  stageAll: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StageAll();
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Failed to stage all" });
    }
  },

  unstageAll: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.UnstageAll();
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Failed to unstage all" });
    }
  },

  commit: async (message) => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const hash = await backend.Commit(message);
      await get().fetchStatus();
      await get().fetchLog();
      return hash;
    } catch (e: any) {
      set({ error: e.message || "Failed to commit" });
      return null;
    }
  },

  checkoutBranch: async (name) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.CheckoutBranch(name);
      await Promise.all([get().fetchStatus(), get().fetchBranches()]);
      set({ diff: null });
    } catch (e: any) {
      set({ error: e.message || `Failed to checkout ${name}` });
    }
  },

  createBranch: async (name) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.CreateBranch(name);
      await get().fetchBranches();
    } catch (e: any) {
      set({ error: e.message || `Failed to create branch ${name}` });
    }
  },

  deleteBranch: async (name, force = false) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.DeleteBranch(name, force);
      await get().fetchBranches();
    } catch (e: any) {
      set({ error: e.message || `Failed to delete branch ${name}` });
    }
  },

  fetchPullRequests: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, prs: true } }));
    try {
      const prs = await backend.GetPullRequests();
      set({ pullRequests: prs, loading: { ...get().loading, prs: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch PRs", loading: { ...get().loading, prs: false } });
    }
  },

  fetchIssues: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, issues: true } }));
    try {
      const issues = await backend.GetIssues();
      set({ issues, loading: { ...get().loading, issues: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch issues", loading: { ...get().loading, issues: false } });
    }
  },

  fetchRepository: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const repo = await backend.GetRepository();
      set({ repository: repo });
    } catch (_) {
      // Non-critical
    }
  },

  checkGitHubAuth: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const authed = await backend.IsGitHubAuthenticated();
      set({ ghAuthenticated: authed });
      if (authed) {
        const user = await backend.GetGitHubUser();
        set({ ghUser: user });
        get().fetchPullRequests().catch(() => {});
        get().fetchIssues().catch(() => {});
      }
    } catch (_) {
      set({ ghAuthenticated: false });
    }
  },

  authenticateGitHub: async (token) => {
    const backend = getBackend();
    if (!backend) return false;
    set((s) => ({ loading: { ...s.loading, auth: true }, error: null }));
    try {
      await backend.AuthenticateGitHub(token);
      set({ ghAuthenticated: true, loginDialogOpen: false, loading: { ...get().loading, auth: false } });
      // Refresh user + data
      try {
        const user = await backend.GetGitHubUser();
        set({ ghUser: user });
      } catch (_) { /* user fetch non-critical */ }
      get().fetchPullRequests().catch(() => {});
      get().fetchIssues().catch(() => {});
      return true;
    } catch (e: any) {
      set({
        error: e.message || "Authentication failed",
        loading: { ...get().loading, auth: false },
      });
      return false;
    }
  },

  startDeviceFlow: async () => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const code = await backend.StartDeviceFlow();
      return code;
    } catch (e: any) {
      set({ error: e.message || "Failed to start device flow" });
      return null;
    }
  },

  pollDeviceFlow: async (deviceCode) => {
    const backend = getBackend();
    if (!backend) return "";
    try {
      const token = await backend.PollDeviceFlow(deviceCode);
      return token;
    } catch (e: any) {
      set({ error: e.message || "Device flow polling failed" });
      return "";
    }
  },

  setLoginDialogOpen: (open) => set({ loginDialogOpen: open }),

  setError: (err) => set({ error: err }),
  clearDiff: () => set({ diff: null }),
}));
