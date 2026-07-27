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
  parents: string[];
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

export interface RebaseCommitOp {
  sha: string;
  action: "pick" | "reword" | "squash" | "fixup" | "drop";
  new_message?: string;
}

export interface RebaseResult {
  success: boolean;
  message?: string;
}

export interface AIConfig {
  provider: string;
  api_key: string;
  model: string;
  endpoint?: string;
}

export interface ReflogEntry {
  sequence: number;
  hash: string;
  action: string;
  subject: string;
  timestamp: string;
  old_hash?: string;
  undoable: boolean;
}

export interface ConflictFile {
  path: string;
  ancestor_sha?: string;
  ours_sha?: string;
  theirs_sha?: string;
  block_count: number;
}

export interface ConflictBlock {
  index: number;
  ours: string;
  theirs: string;
  ours_start: number;
  ours_end: number;
  theirs_start: number;
  theirs_end: number;
  resolved?: string;
  state: "unresolved" | "use-ours" | "use-theirs" | "edited";
}

export interface MergeConflictDetail {
  path: string;
  ours: string;
  theirs: string;
  ancestor?: string;
  raw_content: string;
  blocks: ConflictBlock[];
  has_merge: boolean;
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
          // Commit actions
          CherryPick(sha: string): Promise<void>;
          RevertCommit(sha: string): Promise<void>;
          ResetCommit(sha: string, mode: string): Promise<void>;
          // Tags
          TagList(): Promise<string[]>;
          TagCreate(name: string, target: string, message: string): Promise<void>;
          TagDelete(name: string): Promise<void>;
          // Config
          ConfigGet(key: string): Promise<string>;
          ConfigSet(key: string, value: string, global: boolean): Promise<void>;
          // Conflict resolution
          CheckoutOurs(file: string): Promise<void>;
          CheckoutTheirs(file: string): Promise<void>;
          // Reflog / Undo
          GetReflog(count: number): Promise<ReflogEntry[]>;
          UndoLastAction(): Promise<string>;
          // 3-Way Merge Editor
          GetConflictFiles(): Promise<ConflictFile[]>;
          GetMergeConflictDetail(file: string): Promise<MergeConflictDetail>;
          StageResolvedFile(file: string, content: string): Promise<void>;
          // Rebase
          RebaseSequence(onto: string, commitsJSON: string): Promise<RebaseResult>;
          // AI Commit Message Generator
          GetAIConfig(): Promise<AIConfig>;
          SetAIConfig(provider: string, apiKey: string, model: string, endpoint: string): Promise<void>;
          GenerateCommitMessage(): Promise<string>;
          // Agentic AI
          AgentStart(): Promise<void>;
          AgentChat(message: string): Promise<AgentResponse>;
          AgentApproveProposal(proposalID: string): Promise<ProposalResult>;
          AgentRejectProposal(proposalID: string, feedback: string): Promise<void>;
          AgentReset(): Promise<void>;
          AgentGetProposals(): Promise<AgentActionProposal[]>;
          AgentGetHistory(): Promise<AgentMessage[]>;
          AgentCancel(): Promise<void>;
          // Ask mode
          AskChat(message: string): Promise<string>;
          AskChatStream(message: string): Promise<void>;
          AskChatCancel(): Promise<void>;
          // Sessions
          SessionCreate(name: string, mode: string): Promise<ChatSession>;
          SessionList(): Promise<ChatSession[]>;
          SessionRename(id: string, name: string): Promise<void>;
          SessionDelete(id: string): Promise<void>;
          SessionSwitch(id: string): Promise<ChatSession>;
          SessionGetMessages(id: string): Promise<AgentMessage[]>;
          SessionClearMessages(id: string): Promise<void>;
          SessionActiveID(): Promise<string>;
        };
      };
    };
    Go?: Window["go"];
  }
}

// --- AI Agent types ---

export interface AgentToolCall {
  id: string;
  name: string;
  arguments: string;
}

export interface AgentMessage {
  role: string;           // "user" | "assistant" | "tool" | "system"
  content?: string;
  tool_call_id?: string;
  tool_calls?: AgentToolCall[];
}

export interface AgentActionProposal {
  id: string;
  type: string;
  description: string;
  reasoning: string;
  diff_preview?: string;
  status: "pending" | "approved" | "rejected" | "executed" | "failed";
  created_at: string;
  params?: Record<string, any>;
}

export interface AgentResponse {
  message: string;
  proposals?: AgentActionProposal[];
  finished: boolean;
}

export interface ProposalResult {
  proposal_id: string;
  status: string;
  success: boolean;
  output?: string;
  error?: string;
}

// Chat session (used for both Ask and Agent mode)
export interface ChatSession {
  id: string;
  name: string;
  mode: "ask" | "agent";
  message_count: number;
  created_at: string;
  updated_at: string;
}

// App state store
interface AppState {
  // Git state
  status: Status | null;
  log: Commit[];
  graphLog: Commit[];
  graphView: boolean;
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

  // Tags
  tags: string[];

  // Git config
  gitConfig: Record<string, string>;

  // Merge conflict resolution
  conflictResolutions: Record<string, "ours" | "theirs" | "both">;
  conflictFiles: ConflictFile[];
  mergeEditorOpen: boolean;
  mergeEditorFile: string | null;
  mergeConflictDetail: MergeConflictDetail | null;

  // Interactive Rebase
  rebaseMode: boolean;
  rebaseCommits: RebaseCommitOp[];
  rebaseOnto: string;

  // DnD Merge/Rebase dialog
  mergeRebaseDialog: { branch: string; targetHash: string; targetMsg: string } | null;

  // AI Commit Message
  aiConfig: AIConfig | null;
  aiGenerating: boolean;

  // Agentic AI Assistant (shared for Ask + Agent modes)
  aiPanelOpen: boolean;
  aiSessionActive: boolean;
  aiMessages: AgentMessage[];
  aiProposals: AgentActionProposal[];
  aiThinking: boolean;
  aiError: string | null;
  aiMode: "ask" | "agent";
  aiSessions: ChatSession[];
  aiActiveSessionId: string | null;
  aiFullscreen: boolean;

  // Reflog / Undo
  reflog: ReflogEntry[];
  undoDescription: string | null;

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
  fetchGraphLog: (count?: number) => Promise<void>;
  toggleGraphView: () => void;
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

  // Commit actions
  cherryPick: (sha: string) => Promise<void>;
  revertCommit: (sha: string) => Promise<void>;
  resetCommit: (sha: string, mode: string) => Promise<void>;

  // Tags
  fetchTags: () => Promise<void>;
  createTag: (name: string, target: string, message: string) => Promise<void>;
  deleteTag: (name: string) => Promise<void>;

  // Config
  fetchGitConfig: () => Promise<void>;
  setGitConfig: (key: string, value: string, global?: boolean) => Promise<void>;

  // Merge conflict resolution
  resolveConflict: (file: string, side: "ours" | "theirs") => Promise<void>;
  fetchConflictFiles: () => Promise<void>;
  openMergeEditor: (file: string) => Promise<void>;
  closeMergeEditor: () => void;
  resolveMergeBlock: (blockIndex: number, state: "use-ours" | "use-theirs" | "edited", content?: string) => void;
  saveResolvedFile: () => Promise<void>;

  // Interactive Rebase
  enterRebaseMode: () => void;
  exitRebaseMode: () => void;
  reorderCommits: (fromIndex: number, toIndex: number) => void;
  setCommitAction: (index: number, action: "pick" | "reword" | "squash" | "fixup" | "drop") => void;
  setCommitMessage: (index: number, message: string) => void;
  applyRebase: () => Promise<void>;
  // DnD Dialog
  showMergeRebaseDialog: (branch: string, targetHash: string, targetMsg: string) => void;
  closeMergeRebaseDialog: () => void;
  executeMerge: (branch: string) => Promise<void>;
  executeRebaseOnto: (branch: string, target: string) => Promise<void>;

  // AI Commit Message
  fetchAIConfig: () => Promise<void>;
  setAIConfigAction: (provider: string, apiKey: string, model: string, endpoint?: string) => Promise<void>;
  generateCommitMessage: () => Promise<string | null>;

  // Agentic AI Assistant — Dual Mode
  toggleAIPanel: () => void;
  setAIMode: (mode: "ask" | "agent") => void;
  startAgentSession: () => Promise<boolean>;
  sendAgentMessage: (message: string) => Promise<void>;
  sendAskMessage: (message: string) => Promise<void>;
  askChatCancel: () => void;
  approveProposal: (proposalID: string) => Promise<void>;
  rejectProposal: (proposalID: string, feedback?: string) => Promise<void>;
  resetAgentSession: () => Promise<void>;
  toggleAIFullscreen: () => void;
  // Session management
  fetchSessions: () => Promise<void>;
  createSession: (name: string, mode: "ask" | "agent") => Promise<void>;
  renameSession: (id: string, name: string) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  switchSession: (id: string) => Promise<void>;
  loadSessionMessages: (id: string) => Promise<void>;
  clearSessionMessages: (id: string) => Promise<void>;

  // Reflog / Undo
  fetchReflog: (count?: number) => Promise<void>;
  undoLastAction: () => Promise<void>;
  clearUndoDescription: () => void;

  // Sprint 6: Discard
  discardFile: (file: string) => Promise<void>;
  discardAllFiles: () => Promise<void>;
}

// Reconstruct resolved file content from raw (marker-containing) content + resolved blocks.
function applyResolvedBlocks(rawContent: string, blocks: ConflictBlock[]): string {
  const lines = rawContent.split("\n");
  const result: string[] = [];
  let i = 0;

  for (const block of blocks) {
    // Copy lines before this conflict block
    while (i < lines.length && !lines[i].startsWith("<<<<<<< ")) {
      result.push(lines[i]);
      i++;
    }
    if (i >= lines.length) break;

    // Skip: <<<<<<< ours-ref
    i++; // skip <<<<<<< line
    // Skip: ours lines (until =======)
    while (i < lines.length && !lines[i].startsWith("=======")) {
      i++;
    }
    if (i < lines.length) i++; // skip =======
    // Skip: theirs lines (until >>>>>>>)
    while (i < lines.length && !lines[i].startsWith(">>>>>>> ")) {
      i++;
    }
    if (i < lines.length) i++; // skip >>>>>>> line

    // Insert resolved content
    if (block.resolved !== undefined && block.resolved !== "") {
      result.push(block.resolved);
    }
  }

  // Copy remaining lines after last block
  while (i < lines.length) {
    result.push(lines[i]);
    i++;
  }

  return result.join("\n");
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
  graphLog: [],
  graphView: false,
  branches: [],
  diff: null,
  currentBranch: "",
  selectedPRDetail: null,
  selectedIssueDetail: null,
  remotes: [],
  tags: [],
  gitConfig: {},
  conflictResolutions: {},
  conflictFiles: [],
  mergeEditorOpen: false,
  mergeEditorFile: null,
  mergeConflictDetail: null,
  rebaseMode: false,
  rebaseCommits: [],
  rebaseOnto: "",
  mergeRebaseDialog: null,
  aiConfig: null,
  aiGenerating: false,
  aiPanelOpen: false,
  aiSessionActive: false,
  aiMessages: [],
  aiProposals: [],
  aiThinking: false,
  aiError: null,
  aiMode: "ask",
  aiSessions: [],
  aiActiveSessionId: null,
  aiFullscreen: false,
  reflog: [],
  undoDescription: null,
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
      const msg = e.message || "";
      // Empty repo: git log exits with 128 + "does not have any commits yet"
      if (msg.includes("does not have any commits yet") || msg.includes("does not have any commits")) {
        set({ log: [], loading: { ...get().loading, log: false } });
        return;
      }
      set({ error: msg || "Failed to fetch log", loading: { ...get().loading, log: false } });
    }
  },

  fetchGraphLog: async (count = 500) => {
    const backend = getBackend();
    if (!backend) return;
    set((s) => ({ loading: { ...s.loading, graphLog: true } }));
    try {
      const graphLog = await backend.GetGraphLog(count);
      set({ graphLog, loading: { ...get().loading, graphLog: false } });
    } catch (e: any) {
      const msg = e.message || "";
      if (msg.includes("does not have any commits yet") || msg.includes("does not have any commits")) {
        set({ graphLog: [], loading: { ...get().loading, graphLog: false } });
        return;
      }
      set({ error: msg || "Failed to fetch graph log", loading: { ...get().loading, graphLog: false } });
    }
  },

  toggleGraphView: () => {
    set((s) => ({ graphView: !s.graphView }));
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
      set({ branches: branches || [], currentBranch: currentBranch || "", loading: { ...get().loading, branches: false } });
    } catch (e: any) {
      const msg = e.message || "";
      // Empty repo: git branch commands exit with 128 / unborn HEAD
      if (msg.includes("does not have any commits yet") || msg.includes("not a git repository") || msg.includes("ambiguous argument")) {
        set({ branches: [], currentBranch: "HEAD", loading: { ...get().loading, branches: false } });
        return;
      }
      set({ error: msg || "Failed to fetch branches", loading: { ...get().loading, branches: false } });
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
      set({ remotes: remotes || [] });
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
      // Clear all repo-scoped state before refresh to prevent stale data from previous repo
      set({
        repoPath,
        status: null,
        log: [],
        graphLog: [],
        graphView: false,
        branches: [],
        diff: null,
        currentBranch: "",
        tags: [],
        reflog: [],
        undoDescription: null,
        conflictFiles: [],
        mergeEditorOpen: false,
        mergeEditorFile: null,
        mergeConflictDetail: null,
        rebaseMode: false,
        rebaseCommits: [],
        stashes: [],
        remotes: [],
        error: null,
        loading: { ...get().loading, repo: false },
      });
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
      s.fetchGraphLog(),
      s.fetchBranches(),
      s.fetchPullRequests(),
      s.fetchIssues(),
      s.fetchWorkflowRuns(),
      s.fetchStashes(),
      s.fetchConflictFiles(),
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

  // Commit actions
  cherryPick: async (sha) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.CherryPick(sha);
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || `Failed to cherry-pick ${sha.slice(0,7)}` });
    }
  },

  revertCommit: async (sha) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.RevertCommit(sha);
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || `Failed to revert ${sha.slice(0,7)}` });
    }
  },

  resetCommit: async (sha, mode) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.ResetCommit(sha, mode);
      await get().refreshAll();
    } catch (e: any) {
      set({ error: e.message || `Failed to reset to ${sha.slice(0,7)}` });
    }
  },

  // Tags
  fetchTags: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const tags = await backend.TagList();
      set({ tags });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch tags" });
    }
  },

  createTag: async (name, target, message) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.TagCreate(name, target, message);
      await get().fetchTags();
    } catch (e: any) {
      set({ error: e.message || `Failed to create tag ${name}` });
    }
  },

  deleteTag: async (name) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.TagDelete(name);
      await get().fetchTags();
    } catch (e: any) {
      set({ error: e.message || `Failed to delete tag ${name}` });
    }
  },

  // Config
  fetchGitConfig: async () => {
    const backend = getBackend();
    if (!backend) return;
    const keys = ["user.name", "user.email", "core.autocrlf", "core.editor", "init.defaultBranch"];
    const config: Record<string, string> = {};
    for (const key of keys) {
      try {
        config[key] = await backend.ConfigGet(key);
      } catch { config[key] = ""; }
    }
    set({ gitConfig: config });
  },

  setGitConfig: async (key, value, global = true) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.ConfigSet(key, value, global);
      const cfg = { ...get().gitConfig, [key]: value };
      set({ gitConfig: cfg });
    } catch (e: any) {
      set({ error: e.message || `Failed to set config ${key}` });
    }
  },

  // Merge conflict resolution
  resolveConflict: async (file, side) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      if (side === "ours") {
        await backend.CheckoutOurs(file);
      } else {
        await backend.CheckoutTheirs(file);
      }
      await get().refreshAll();
      const resolutions = { ...get().conflictResolutions, [file]: side };
      set({ conflictResolutions: resolutions });
    } catch (e: any) {
      set({ error: e.message || `Failed to resolve conflict in ${file}` });
    }
  },

  // 3-Way Merge Editor
  fetchConflictFiles: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const files: ConflictFile[] = await backend.GetConflictFiles();
      set({ conflictFiles: files });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch conflict files" });
    }
  },

  openMergeEditor: async (file) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const detail: MergeConflictDetail = await backend.GetMergeConflictDetail(file);
      set({ mergeEditorOpen: true, mergeEditorFile: file, mergeConflictDetail: detail });
    } catch (e: any) {
      set({ error: e.message || `Failed to open merge editor for ${file}` });
    }
  },

  closeMergeEditor: () => {
    set({ mergeEditorOpen: false, mergeEditorFile: null, mergeConflictDetail: null });
  },

  resolveMergeBlock: (blockIndex, state, content) => {
    const detail = get().mergeConflictDetail;
    if (!detail) return;
    const blocks = [...detail.blocks];
    const block = { ...blocks[blockIndex] };
    block.state = state;

    if (state === "use-ours") {
      block.resolved = block.ours;
    } else if (state === "use-theirs") {
      block.resolved = block.theirs;
    } else if (state === "edited" && content !== undefined) {
      block.resolved = content;
    }

    blocks[blockIndex] = block;
    set({ mergeConflictDetail: { ...detail, blocks } });
  },

  saveResolvedFile: async () => {
    const backend = getBackend();
    const detail = get().mergeConflictDetail;
    const file = get().mergeEditorFile;
    if (!backend || !detail || !file) return;
    try {
      // Reconstruct file: replace each conflict block with its resolved content
      const content = applyResolvedBlocks(detail.raw_content, detail.blocks);
      await backend.StageResolvedFile(file, content);
      set({ mergeEditorOpen: false, mergeEditorFile: null, mergeConflictDetail: null });
      await get().refreshAll();
      // Refresh conflict files list
      await get().fetchConflictFiles();
    } catch (e: any) {
      set({ error: e.message || `Failed to save resolved file ${file}` });
    }
  },

  // Interactive Rebase
  enterRebaseMode: () => {
    const log = get().log;
    const commits: RebaseCommitOp[] = log.map((c) => ({
      sha: c.hash,
      action: "pick" as const,
    }));
    set({ rebaseMode: true, rebaseCommits: commits, rebaseOnto: "" });
  },

  exitRebaseMode: () => {
    set({ rebaseMode: false, rebaseCommits: [], rebaseOnto: "" });
  },

  reorderCommits: (fromIndex, toIndex) => {
    const commits = [...get().rebaseCommits];
    const [moved] = commits.splice(fromIndex, 1);
    commits.splice(toIndex, 0, moved);
    set({ rebaseCommits: commits });
  },

  setCommitAction: (index, action) => {
    const commits = [...get().rebaseCommits];
    commits[index] = { ...commits[index], action };
    set({ rebaseCommits: commits });
  },

  setCommitMessage: (index, message) => {
    const commits = [...get().rebaseCommits];
    commits[index] = { ...commits[index], new_message: message };
    set({ rebaseCommits: commits });
  },

  applyRebase: async () => {
    const backend = getBackend();
    const { rebaseCommits, rebaseOnto, currentBranch } = get();
    if (!backend || rebaseCommits.length === 0) return;
    try {
      set((s) => ({ loading: { ...s.loading, rebase: true }, error: null }));
      // Determine onto: parent of first commit being rebased
      const onto = rebaseOnto || `${rebaseCommits[0].sha}^`;
      const commitsJSON = JSON.stringify(rebaseCommits);
      const result: RebaseResult = await backend.RebaseSequence(onto, commitsJSON);
      set((s) => ({ loading: { ...s.loading, rebase: false } }));
      if (result.success) {
        set({ rebaseMode: false, rebaseCommits: [] });
        await get().refreshAll();
        await get().fetchLog(100);
      } else {
        set({ error: result.message || "Rebase failed" });
      }
    } catch (e: any) {
      set((s) => ({ loading: { ...s.loading, rebase: false } }));
      set({ error: e.message || "Rebase failed" });
    }
  },

  showMergeRebaseDialog: (branch, targetHash, targetMsg) => {
    set({ mergeRebaseDialog: { branch, targetHash, targetMsg } });
  },

  closeMergeRebaseDialog: () => {
    set({ mergeRebaseDialog: null });
  },

  executeMerge: async (branch) => {
    try {
      await get().gitMerge(branch);
      set({ mergeRebaseDialog: null });
    } catch (e: any) {
      set({ error: e.message || `Merge failed: ${branch}` });
      set({ mergeRebaseDialog: null });
    }
  },

  executeRebaseOnto: async (branch, target) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const commitsJSON = JSON.stringify([{ sha: target, action: "pick" }]);
      const result: RebaseResult = await backend.RebaseSequence(branch, commitsJSON);
      if (result.success) {
        set({ mergeRebaseDialog: null });
        await get().refreshAll();
        await get().fetchLog(100);
      } else {
        set({ error: result.message || "Rebase failed" });
        set({ mergeRebaseDialog: null });
      }
    } catch (e: any) {
      set({ error: e.message || `Rebase failed: ${e.message}` });
      set({ mergeRebaseDialog: null });
    }
  },

  // AI Commit Message
  fetchAIConfig: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const config: AIConfig = await backend.GetAIConfig();
      set({ aiConfig: config });
    } catch (e: any) {
      // Silently fail — AI is optional
    }
  },

  setAIConfigAction: async (provider, apiKey, model, endpoint) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.SetAIConfig(provider, apiKey, model, endpoint || "");
      set({ aiConfig: { provider, api_key: apiKey, model, endpoint } });
    } catch (e: any) {
      set({ error: e.message || "Failed to save AI config" });
    }
  },

  generateCommitMessage: async () => {
    const backend = getBackend();
    if (!backend) return null;
    try {
      set({ aiGenerating: true });
      const message: string = await backend.GenerateCommitMessage();
      set({ aiGenerating: false });
      return message;
    } catch (e: any) {
      set({ error: e.message || "Failed to generate commit message" });
      set({ aiGenerating: false });
      return null;
    }
  },

  // --- Agentic AI Assistant — Dual Mode ---

  setAIMode: (mode) => {
    const prev = get().aiMode;
    set({ aiMode: mode });
    // Switching to Agent: auto-start session
    if (mode === "agent" && prev !== "agent") {
      if (!get().aiSessionActive) {
        get().startAgentSession();
      }
    }
    // Switching to Ask: cancel any agent session, clear proposals
    if (mode === "ask" && prev !== "ask") {
      set({ aiProposals: [] });
    }
  },

  toggleAIPanel: () => {
    const next = !get().aiPanelOpen;
    set({ aiPanelOpen: next, aiFullscreen: false });
    // Auto-start based on current mode
    if (next && !get().aiSessionActive) {
      if (get().aiMode === "agent") {
        get().startAgentSession();
      }
    }
  },

  toggleAIFullscreen: () => {
    set((s) => ({ aiFullscreen: !s.aiFullscreen }));
  },

  startAgentSession: async () => {
    const backend = getBackend();
    if (!backend) return false;
    try {
      await backend.AgentStart();
      // Create or ensure an agent session
      const sessions = await backend.SessionList();
      const existingAgent = sessions.find((s: ChatSession) => s.mode === "agent");
      if (!existingAgent) {
        const session = await backend.SessionCreate("Agent Session", "agent");
        set({ aiActiveSessionId: session.id });
      } else {
        set({ aiActiveSessionId: existingAgent.id });
      }
      set({
        aiSessionActive: true,
        aiMessages: [],
        aiProposals: [],
        aiThinking: false,
        aiError: null,
        aiMode: "agent",
      });
      // Refresh session list
      get().fetchSessions();
      return true;
    } catch (e: any) {
      set({ aiError: e.message || "Failed to start agent session", aiSessionActive: false });
      return false;
    }
  },

  sendAgentMessage: async (message: string) => {
    const backend = getBackend();
    if (!backend) return;
    const msg: AgentMessage = { role: "user", content: message };
    set((s) => ({ aiMessages: [...s.aiMessages, msg], aiThinking: true, aiError: null }));
    try {
      const resp: AgentResponse = await backend.AgentChat(message);
      const assistantMsg: AgentMessage = {
        role: "assistant",
        content: resp.message || "",
      };
      set((s) => ({
        aiMessages: [...s.aiMessages, assistantMsg],
        aiProposals: resp.proposals || [],
        aiThinking: false,
      }));
    } catch (e: any) {
      set({ aiError: e.message || "Agent error", aiThinking: false });
    }
  },

  sendAskMessage: async (message: string) => {
    const backend = getBackend();
    if (!backend) return;
    const msg: AgentMessage = { role: "user", content: message };
    set((s) => ({ aiMessages: [...s.aiMessages, msg], aiThinking: true, aiError: null, aiSessionActive: true }));
    try {
      const responseText: string = await backend.AskChat(message);
      const assistantMsg: AgentMessage = {
        role: "assistant",
        content: responseText || "",
      };
      set((s) => ({
        aiMessages: [...s.aiMessages, assistantMsg],
        aiThinking: false,
      }));
      // Refresh sessions to get updated message count
      get().fetchSessions();
    } catch (e: any) {
      set({ aiError: e.message || "Ask error", aiThinking: false });
    }
  },

  askChatCancel: () => {
    const backend = getBackend();
    if (!backend) return;
    backend.AskChatCancel().catch(() => {});
    set({ aiThinking: false });
  },

  approveProposal: async (proposalID: string) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      set((s) => ({
        aiProposals: s.aiProposals.map((p) =>
          p.id === proposalID ? { ...p, status: "approved" as const } : p
        ),
      }));
      const result: ProposalResult = await backend.AgentApproveProposal(proposalID);

      // Add result to chat
      const resultMsg: AgentMessage = {
        role: "tool",
        tool_call_id: proposalID,
        content: result.success
          ? `✅ ${result.output || "Action completed successfully"}`
          : `❌ ${result.error || "Action failed"}`,
      };
      set((s) => ({ aiMessages: [...s.aiMessages, resultMsg] }));

      if (result.success) {
        set((s) => ({
          aiProposals: s.aiProposals.map((p) =>
            p.id === proposalID ? { ...p, status: "executed" as const } : p
          ),
        }));
        // Continue agent loop: send empty message to let agent check context
        set({ aiThinking: true });
        try {
          const followUp: AgentResponse = await backend.AgentChat("_continue");
          const followMsg: AgentMessage = {
            role: "assistant",
            content: followUp.message || "",
          };
          set((s) => ({
            aiMessages: [...s.aiMessages, followMsg],
            aiProposals: followUp.proposals || [],
            aiThinking: false,
          }));
        } catch {
          set({ aiThinking: false });
        }
      } else {
        set((s) => ({
          aiProposals: s.aiProposals.map((p) =>
            p.id === proposalID ? { ...p, status: "failed" as const } : p
          ),
        }));
      }
    } catch (e: any) {
      set({ aiError: e.message || "Failed to execute proposal" });
    }
  },

  rejectProposal: async (proposalID: string, feedback?: string) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.AgentRejectProposal(proposalID, feedback || "");
      set((s) => ({
        aiProposals: s.aiProposals.map((p) =>
          p.id === proposalID ? { ...p, status: "rejected" as const } : p
        ),
      }));

      // If feedback provided, send it to agent for alternative suggestion
      if (feedback) {
        const feedbackMsg: AgentMessage = { role: "user", content: feedback };
        set((s) => ({ aiMessages: [...s.aiMessages, feedbackMsg], aiThinking: true }));
        try {
          const resp: AgentResponse = await backend.AgentChat(feedback);
          const assistantMsg: AgentMessage = {
            role: "assistant",
            content: resp.message || "",
          };
          set((s) => ({
            aiMessages: [...s.aiMessages, assistantMsg],
            aiProposals: resp.proposals || [],
            aiThinking: false,
          }));
        } catch (e: any) {
          set({ aiError: e.message || "Agent error", aiThinking: false });
        }
      }
    } catch (e: any) {
      set({ aiError: e.message || "Failed to reject proposal" });
    }
  },

  resetAgentSession: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.AgentReset();
    } catch {
      // ignore
    }
    set({
      aiSessionActive: false,
      aiMessages: [],
      aiProposals: [],
      aiThinking: false,
      aiError: null,
      aiPanelOpen: false,
      aiFullscreen: false,
    });
  },

  // --- Session Management ---

  fetchSessions: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const sessions: ChatSession[] = await backend.SessionList();
      set({ aiSessions: sessions });
      // Update active session ID if not set
      const activeID: string = await backend.SessionActiveID();
      set({ aiActiveSessionId: activeID || null });
    } catch {
      // Silent
    }
  },

  createSession: async (name, mode) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const session: ChatSession = await backend.SessionCreate(name, mode);
      set({
        aiActiveSessionId: session.id,
        aiMessages: [],
        aiProposals: [],
        aiSessionActive: true,
      });
      get().fetchSessions();
    } catch (e: any) {
      set({ error: e.message || "Failed to create session" });
    }
  },

  renameSession: async (id, name) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.SessionRename(id, name);
      get().fetchSessions();
    } catch (e: any) {
      set({ error: e.message || "Failed to rename session" });
    }
  },

  deleteSession: async (id) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.SessionDelete(id);
      const state = get();
      if (state.aiActiveSessionId === id) {
        set({ aiActiveSessionId: null, aiMessages: [], aiProposals: [], aiSessionActive: false });
      }
      get().fetchSessions();
    } catch (e: any) {
      set({ error: e.message || "Failed to delete session" });
    }
  },

  switchSession: async (id) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const session: ChatSession = await backend.SessionSwitch(id);
      const messages: AgentMessage[] = await backend.SessionGetMessages(id);
      set({
        aiActiveSessionId: session.id,
        aiMode: session.mode as "ask" | "agent",
        aiMessages: messages,
        aiProposals: [],
        aiSessionActive: true,
        aiThinking: false,
        aiError: null,
      });
    } catch (e: any) {
      set({ error: e.message || "Failed to switch session" });
    }
  },

  loadSessionMessages: async (id) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const messages: AgentMessage[] = await backend.SessionGetMessages(id);
      set({ aiMessages: messages });
    } catch {
      // Silent
    }
  },

  clearSessionMessages: async (id) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      await backend.SessionClearMessages(id);
      set({ aiMessages: [] });
    } catch (e: any) {
      set({ error: e.message || "Failed to clear session" });
    }
  },

  // Reflog / Undo
  fetchReflog: async (count = 10) => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const entries: ReflogEntry[] = await backend.GetReflog(count);
      set({ reflog: entries });
    } catch (e: any) {
      set({ error: e.message || "Failed to fetch reflog" });
    }
  },

  undoLastAction: async () => {
    const backend = getBackend();
    if (!backend) return;
    try {
      const description: string = await backend.UndoLastAction();
      set({ undoDescription: description });
      await get().refreshAll();
      // Refresh reflog to show the undo entry
      await get().fetchReflog(5);
    } catch (e: any) {
      set({ error: e.message || "Failed to undo last action" });
    }
  },

  clearUndoDescription: () => {
    set({ undoDescription: null });
  },
}));
