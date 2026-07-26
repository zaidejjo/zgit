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
  status_emoji?: string;
  review_state?: string;
  labels?: string[];
}

export interface Review {
  id: number;
  author: string;
  state: string;
  body: string;
  submitted_at: string;
}

export interface Remote {
  name: string;
  url: string;
  push_url?: string;
  type: string; // "fetch" or "push"
}

export interface CheckRun {
  name: string;
  state: string;
  conclusion: string;
  details_url?: string;
}

export interface PullRequestDetail {
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
  body: string;
  closed_at?: string;
  merged_at?: string;
  merged_by?: string;
  additions: number;
  deletions: number;
  changed_files: number;
  commits?: Commit[];
  reviews?: Review[];
  check_runs?: CheckRun[];
  comments: number;
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

export interface IssueComment {
  id: number;
  author: string;
  body: string;
  created_at: string;
  updated_at: string;
  is_minimized: boolean;
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

export interface Step {
  name: string;
  status: string;
  conclusion?: string;
  number: number;
}

export interface Job {
  id: number;
  name: string;
  status: string;
  conclusion?: string;
  started_at: string;
  completed_at?: string;
  runner_name?: string;
  steps?: Step[];
}

export interface WorkflowRun {
  id: number;
  workflow_name: string;
  event: string;
  status: string;
  conclusion?: string;
  branch: string;
  head_sha: string;
  run_number: number;
  created_at: string;
  updated_at: string;
  html_url: string;
}

export interface StashEntry {
  index: number;
  hash: string;
  message: string;
  time: string; // ISO timestamp from Go time.Time
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
          GitRenameBranch(oldName: string, newName: string): Promise<void>;
          GetDiff(pathspec: string): Promise<Diff>;
          StageFile(file: string): Promise<void>;
          UnstageFile(file: string): Promise<void>;
          StageAll(): Promise<void>;
          UnstageAll(): Promise<void>;
          Commit(message: string, body: string): Promise<string>;
          CheckoutBranch(name: string): Promise<void>;
          CreateBranch(name: string): Promise<void>;
          DeleteBranch(name: string, force: boolean): Promise<void>;
          CurrentBranch(): Promise<string>;
          GitMerge(branch: string): Promise<string>;
          IsGitHubAuthenticated(): Promise<boolean>;
          GetPullRequests(): Promise<PRSummary[]>;
          GetIssues(): Promise<Issue[]>;
          GetRepository(): Promise<Repo>;
          GetCurrentRepoPath(): Promise<string>;
          GetGitHubUser(): Promise<GitHubUser>;
          AuthenticateGitHub(token: string): Promise<void>;
          GetRemotes(): Promise<Remote[]>;
          AddRemote(name: string, url: string): Promise<void>;
          RemoveRemote(name: string): Promise<void>;
          GetAheadCommits(): Promise<Commit[]>;
          StartDeviceFlow(): Promise<DeviceFlowCode>;
          PollDeviceFlow(deviceCode: string): Promise<string>;
          GitPush(): Promise<void>;
          GetRepoPath(): Promise<string>;
          ListWorkflowRuns(): Promise<WorkflowRun[]>;
          ReRunWorkflow(runID: number): Promise<void>;
          CancelWorkflowRun(runID: number): Promise<void>;
          ListWorkflowJobs(runID: number): Promise<Job[]>;
          GetWorkflowJobLogs(jobID: number): Promise<string>;
          // Repo management
          OpenRepo(path: string): Promise<void>;
          ResolveGitRoot(path: string): Promise<string>;
          ListRecentRepos(): Promise<string[]>;
          PickDirectory(): Promise<string>;
          GetRepoName(): Promise<string>;
          // Sprint 6: Fetch / Pull / Push force
          GitFetch(): Promise<void>;
          GitPull(rebase: boolean): Promise<void>;
          GitPushForce(): Promise<void>;
          // Sprint 6: Stash
          StashList(): Promise<StashEntry[]>;
          StashPush(message: string): Promise<void>;
          StashPop(index: number): Promise<void>;
          StashApply(index: number): Promise<void>;
          StashDrop(index: number): Promise<void>;
          // Sprint 6: Discard
          DiscardFile(file: string): Promise<void>;
          DiscardAllFiles(): Promise<void>;
          // PR management
          GetPullRequestDetail(number: number): Promise<PullRequestDetail>;
          CreatePullRequest(title: string, body: string, head: string, base: string, draft: boolean): Promise<PRSummary>;
          MergePullRequest(number: number, method: string): Promise<void>;
          // Line-level staging
          StagePatch(patch: string): Promise<void>;
          // Issue management
          GetIssueDetail(number: number): Promise<Issue>;
          CreateIssue(title: string, body: string): Promise<Issue>;
          CloseIssue(number: number): Promise<void>;
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
  workflowRuns: WorkflowRun[];
  selectedRunJobs: Job[];
  selectedJobLogs: string;
  stashes: StashEntry[];

  // Repo management
  repoPath: string;
  recentRepos: string[];

  // PR detail
  selectedPRDetail: PullRequestDetail | null;
  // Issue detail
  selectedIssueDetail: Issue | null;

  // Remote management
  remotes: Remote[];

  // Push confirmation
  pendingPush: boolean;
  pushCommits: Commit[];
  showPushDialog: boolean;

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
  commit: (message: string, body?: string) => Promise<string | null>;
  gitPush: () => Promise<void>;
  commitAndPush: (message: string, body?: string) => Promise<string | null>;
  checkoutBranch: (name: string) => Promise<void>;
  createBranch: (name: string) => Promise<void>;
  deleteBranch: (name: string, force?: boolean) => Promise<void>;
  renameBranch: (oldName: string, newName: string) => Promise<void>;
  gitMerge: (branch: string) => Promise<string | null>;
  fetchPullRequests: () => Promise<void>;
  fetchPRDetail: (number: number) => Promise<void>;
  clearPRDetail: () => void;
  createPullRequest: (title: string, body: string, head: string, base: string, draft?: boolean) => Promise<PRSummary | null>;
  mergePullRequest: (number: number, method: string) => Promise<boolean>;
  fetchIssueDetail: (number: number) => Promise<void>;
  clearIssueDetail: () => void;
  createIssue: (title: string, body: string) => Promise<Issue | null>;
  closeIssue: (number: number) => Promise<boolean>;
  fetchIssues: () => Promise<void>;
  fetchRepository: () => Promise<void>;
  checkGitHubAuth: () => Promise<void>;
  authenticateGitHub: (token: string) => Promise<boolean>;
  fetchWorkflowRuns: () => Promise<void>;
  reRunWorkflow: (runID: number) => Promise<void>;
  cancelWorkflowRun: (runID: number) => Promise<void>;
  fetchWorkflowJobs: (runID: number) => Promise<void>;
  fetchJobLogs: (jobID: number) => Promise<void>;
  clearJobLogs: () => void;
  startDeviceFlow: () => Promise<DeviceFlowCode | null>;
  pollDeviceFlow: (deviceCode: string) => Promise<string>;
  setLoginDialogOpen: (open: boolean) => void;
  setError: (err: string | null) => void;
  clearDiff: () => void;

  // Repo management
  fetchRecentRepos: () => Promise<void>;
  openRepo: (path: string) => Promise<string | null>;
  selectAndOpenRepo: () => Promise<string | null>;
  refreshAll: () => Promise<void>;

  // Sprint 6: Git sync
  gitFetch: () => Promise<void>;
  gitPull: (rebase?: boolean) => Promise<void>;
  gitPushForce: () => Promise<void>;

  // Sprint 6: Stash
  fetchStashes: () => Promise<void>;
  stashPush: (message?: string) => Promise<void>;
  stashPop: (index: number) => Promise<void>;
  stashApply: (index: number) => Promise<void>;
  stashDrop: (index: number) => Promise<void>;

  // Diff viewer: hunk staging
  stagePatch: (patch: string) => Promise<void>;

  // Remote management
  fetchRemotes: () => Promise<void>;
  addRemote: (name: string, url: string) => Promise<void>;
  removeRemote: (name: string) => Promise<void>;

  // Push confirmation
  requestPush: () => Promise<void>;
  confirmPush: () => Promise<void>;
  cancelPush: () => void;

  // Sprint 6: Discard
  discardFile: (file: string) => Promise<void>;
  discardAllFiles: () => Promise<void>;
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
  selectedPRDetail: null,
  selectedIssueDetail: null,
  remotes: [],
  pendingPush: false,
  pushCommits: [],
  showPushDialog: false,
  ghAuthenticated: false,
  ghUser: null,
  pullRequests: [],
  issues: [],
  repository: null,
  workflowRuns: [],
  selectedRunJobs: [],
  selectedJobLogs: "",
  stashes: [],
  repoPath: "",
  recentRepos: [],
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

  stagePatch: async (patch) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StagePatch(patch);
      await get().fetchStatus();
      // Refresh the diff to reflect staged changes
      await get().fetchDiff();
    } catch (e: any) {
      set({ error: e.message || "Failed to stage hunk" });
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

  commit: async (message, body) => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const hash = await backend.Commit(message, body || "");
      await get().fetchStatus();
      await get().fetchLog();
      return hash;
    } catch (e: any) {
      set({ error: e.message || "Failed to commit" });
      return null;
    }
  },

  gitPush: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.GitPush();
    } catch (e: any) {
      set({ error: e.message || "Failed to push" });
    }
  },

  commitAndPush: async (message, body) => {
    const hash = await get().commit(message, body);
    if (hash) {
      // Show push confirmation dialog
      await get().requestPush();
    }
    return hash;
  },

  // Remote management
  fetchRemotes: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const remotes = await backend.GetRemotes();
      set({ remotes });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch remotes" });
    }
  },

  addRemote: async (name, url) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.AddRemote(name, url);
      await get().fetchRemotes();
    } catch (e: any) {
      set({ error: e.message || `Failed to add remote ${name}` });
    }
  },

  removeRemote: async (name) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.RemoveRemote(name);
      await get().fetchRemotes();
    } catch (e: any) {
      set({ error: e.message || `Failed to remove remote ${name}` });
    }
  },

  // Push confirmation
  requestPush: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const commits = await backend.GetAheadCommits();
      set({ pushCommits: commits, pendingPush: true, showPushDialog: true });
    } catch (e: any) {
      // If we can't get ahead commits, just push directly
      set({ showPushDialog: false });
      await get().gitPush();
    }
  },

  confirmPush: async () => {
    set({ pendingPush: false, showPushDialog: false });
    await get().gitPush();
  },

  cancelPush: () => {
    set({ pendingPush: false, showPushDialog: false, pushCommits: [] });
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

  renameBranch: async (oldName, newName) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.GitRenameBranch(oldName, newName);
      await get().fetchBranches();
    } catch (e: any) {
      set({ error: e.message || `Failed to rename branch` });
    }
  },

  gitMerge: async (branch) => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const result = await backend.GitMerge(branch);
      await get().refreshAll();
      return result;
    } catch (e: any) {
      set({ error: e.message || `Failed to merge ${branch}` });
      return null;
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

  fetchPRDetail: async (number) => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, prDetail: true }, error: null }));
    try {
      const detail = await backend.GetPullRequestDetail(number);
      set({ selectedPRDetail: detail, loading: { ...get().loading, prDetail: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch PR detail", loading: { ...get().loading, prDetail: false } });
    }
  },

  clearPRDetail: () => set({ selectedPRDetail: null }),

  createPullRequest: async (title, body, head, base, draft = false) => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const pr = await backend.CreatePullRequest(title, body, head, base, draft);
      await get().fetchPullRequests();
      return pr;
    } catch (e: any) {
      set({ error: e.message || "Failed to create PR" });
      return null;
    }
  },

  mergePullRequest: async (number, method) => {
    const backend = getBackend();
    if (!backend) return false;
    try {
      await backend.MergePullRequest(number, method);
      await get().fetchPullRequests();
      set({ selectedPRDetail: null });
      return true;
    } catch (e: any) {
      set({ error: e.message || "Failed to merge PR" });
      return false;
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

  fetchIssueDetail: async (number) => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, issueDetail: true }, error: null }));
    try {
      const detail = await backend.GetIssueDetail(number);
      set({ selectedIssueDetail: detail, loading: { ...get().loading, issueDetail: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch issue detail", loading: { ...get().loading, issueDetail: false } });
    }
  },

  clearIssueDetail: () => set({ selectedIssueDetail: null }),

  createIssue: async (title, body) => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const issue = await backend.CreateIssue(title, body);
      await get().fetchIssues();
      return issue;
    } catch (e: any) {
      set({ error: e.message || "Failed to create issue" });
      return null;
    }
  },

  closeIssue: async (number) => {
    const backend = getBackend();
    if (!backend) return false;
    try {
      await backend.CloseIssue(number);
      await get().fetchIssues();
      set({ selectedIssueDetail: null });
      return true;
    } catch (e: any) {
      set({ error: e.message || "Failed to close issue" });
      return false;
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

  fetchWorkflowRuns: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, workflowRuns: true } }));
    try {
      const runs = (await backend.ListWorkflowRuns()) || [];
      set({ workflowRuns: runs, loading: { ...get().loading, workflowRuns: false } });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch workflow runs", loading: { ...get().loading, workflowRuns: false } });
    }
  },

  reRunWorkflow: async (runID) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.ReRunWorkflow(runID);
      get().fetchWorkflowRuns();
    } catch (e: any) {
      set({ error: e.message || "Failed to re-run workflow" });
    }
  },

  cancelWorkflowRun: async (runID) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.CancelWorkflowRun(runID);
      get().fetchWorkflowRuns();
    } catch (e: any) {
      set({ error: e.message || "Failed to cancel workflow" });
    }
  },

  fetchWorkflowJobs: async (runID) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const jobs = (await backend.ListWorkflowJobs(runID)) || [];
      set({ selectedRunJobs: jobs, selectedJobLogs: "" });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch jobs" });
    }
  },

  fetchJobLogs: async (jobID) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const logs = (await backend.GetWorkflowJobLogs(jobID)) || "";
      set({ selectedJobLogs: logs });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch job logs" });
    }
  },

  clearJobLogs: () => set({ selectedRunJobs: [], selectedJobLogs: "" }),

  setLoginDialogOpen: (open) => set({ loginDialogOpen: open }),

  setError: (err) => set({ error: err }),
  clearDiff: () => set({ diff: null }),

  // --- Repo management ---

  fetchRecentRepos: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const repos = await backend.ListRecentRepos();
      set({ recentRepos: repos || [] });
      const path = await backend.GetRepoPath();
      set({ repoPath: path || "" });
    } catch (_) {
      // Non-critical
    }
  },

  openRepo: async (path) => {
    const backend = getBackend();
    if (!backend) return null;
    set((s) => ({ loading: { ...s.loading, repo: true }, error: null }));
    try {
      await backend.OpenRepo(path);
      const repoPath = await backend.GetRepoPath();
      set({ repoPath, loading: { ...get().loading, repo: false } });
      // Refresh all data
      await get().refreshAll();
      // Refresh recent list
      get().fetchRecentRepos().catch(() => {});
      return repoPath;
    } catch (e: any) {
      set({
        error: e.message || "Failed to open repository",
        loading: { ...get().loading, repo: false },
      });
      return null;
    }
  },

  selectAndOpenRepo: async () => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      const dir = await backend.PickDirectory();
      if (!dir) return null; // User cancelled
      return await get().openRepo(dir);
    } catch (e: any) {
      set({ error: e.message || "Failed to pick directory" });
      return null;
    }
  },

  refreshAll: async () => {
    const s = get();
    await Promise.allSettled([
      s.fetchStatus(),
      s.fetchLog(),
      s.fetchBranches(),
      s.fetchPullRequests(),
      s.fetchIssues(),
      s.fetchWorkflowRuns(),
      s.fetchStashes(),
    ]);
  },

  // --- Sprint 6: Git sync ---

  gitFetch: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, fetch: true }, error: null }));
    try {
      await backend.GitFetch();
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || "Fetch failed", loading: { ...get().loading, fetch: false } });
      return;
    }
    set((s) => ({ loading: { ...s.loading, fetch: false } }));
  },

  gitPull: async (rebase = false) => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, pull: true }, error: null }));
    try {
      await backend.GitPull(rebase);
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || "Pull failed", loading: { ...get().loading, pull: false } });
      return;
    }
    set((s) => ({ loading: { ...s.loading, pull: false } }));
  },

  gitPushForce: async () => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, push: true }, error: null }));
    try {
      await backend.GitPushForce();
    } catch (e: any) {
      set({ error: e.message || "Force push failed" });
    }
    set((s) => ({ loading: { ...s.loading, push: false } }));
  },

  // --- Sprint 6: Stash ---

  fetchStashes: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const list = await backend.StashList();
      set({ stashes: list || [] });
    } catch (_) {
      // Non-critical
    }
  },

  stashPush: async (message = "") => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StashPush(message);
      await get().fetchStashes();
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Stash failed" });
    }
  },

  stashPop: async (index) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StashPop(index);
      await get().fetchStashes();
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Stash pop failed" });
    }
  },

  stashApply: async (index) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StashApply(index);
      await get().fetchStashes();
      await get().fetchStatus();
    } catch (e: any) {
      set({ error: e.message || "Stash apply failed" });
    }
  },

  stashDrop: async (index) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.StashDrop(index);
      await get().fetchStashes();
    } catch (e: any) {
      set({ error: e.message || "Stash drop failed" });
    }
  },

  // --- Sprint 6: Discard ---

  discardFile: async (file) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.DiscardFile(file);
      await get().fetchStatus();
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || `Failed to discard ${file}` });
    }
  },

  discardAllFiles: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.DiscardAllFiles();
      await get().fetchStatus();
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || "Failed to discard all changes" });
    }
  },
}));
