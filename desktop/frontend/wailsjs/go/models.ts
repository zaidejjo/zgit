export namespace ai {
	
	export class AgentActionProposal {
	    id: string;
	    type: string;
	    description: string;
	    reasoning: string;
	    diff_preview?: string;
	    status: string;
	    // Go type: time
	    created_at: any;
	    params?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new AgentActionProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.reasoning = source["reasoning"];
	        this.diff_preview = source["diff_preview"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.params = source["params"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentResponse {
	    message: string;
	    proposals?: AgentActionProposal[];
	    finished: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.proposals = this.convertValues(source["proposals"], AgentActionProposal);
	        this.finished = source["finished"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolCall {
	    id: string;
	    name: string;
	    arguments: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	    }
	}
	export class Message {
	    role: string;
	    content?: string;
	    tool_call_id?: string;
	    tool_calls?: ToolCall[];
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.tool_call_id = source["tool_call_id"];
	        this.tool_calls = this.convertValues(source["tool_calls"], ToolCall);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProposalResult {
	    proposal_id: string;
	    status: string;
	    success: boolean;
	    output?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProposalResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proposal_id = source["proposal_id"];
	        this.status = source["status"];
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class SessionSummary {
	    id: string;
	    name: string;
	    mode: string;
	    message_count: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.message_count = source["message_count"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace embed {
	
	export class FS {
	
	
	    static createFrom(source: any = {}) {
	        return new FS(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace models {
	
	export class AIConfig {
	    provider: string;
	    api_key: string;
	    model: string;
	    endpoint?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.endpoint = source["endpoint"];
	    }
	}
	export class AppearanceConfig {
	    theme: string;
	    accent_color: string;
	    brightness: number;
	
	    static createFrom(source: any = {}) {
	        return new AppearanceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.accent_color = source["accent_color"];
	        this.brightness = source["brightness"];
	    }
	}
	export class Branch {
	    name: string;
	    full_ref: string;
	    type: number;
	    is_head: boolean;
	    upstream?: string;
	    ahead?: number;
	    behind?: number;
	    latest_hash?: string;
	    latest_msg?: string;
	
	    static createFrom(source: any = {}) {
	        return new Branch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.full_ref = source["full_ref"];
	        this.type = source["type"];
	        this.is_head = source["is_head"];
	        this.upstream = source["upstream"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.latest_hash = source["latest_hash"];
	        this.latest_msg = source["latest_msg"];
	    }
	}
	export class CheckRun {
	    name: string;
	    state: string;
	    conclusion: string;
	    details_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.state = source["state"];
	        this.conclusion = source["conclusion"];
	        this.details_url = source["details_url"];
	    }
	}
	export class Commit {
	    hash: string;
	    parents: string[];
	    author: string;
	    email: string;
	    message: string;
	    // Go type: time
	    timestamp: any;
	    ref_names?: string;
	
	    static createFrom(source: any = {}) {
	        return new Commit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.parents = source["parents"];
	        this.author = source["author"];
	        this.email = source["email"];
	        this.message = source["message"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.ref_names = source["ref_names"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConflictBlock {
	    index: number;
	    ours: string;
	    theirs: string;
	    ours_start: number;
	    ours_end: number;
	    theirs_start: number;
	    theirs_end: number;
	    resolved?: string;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new ConflictBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.ours = source["ours"];
	        this.theirs = source["theirs"];
	        this.ours_start = source["ours_start"];
	        this.ours_end = source["ours_end"];
	        this.theirs_start = source["theirs_start"];
	        this.theirs_end = source["theirs_end"];
	        this.resolved = source["resolved"];
	        this.state = source["state"];
	    }
	}
	export class ConflictFile {
	    path: string;
	    ancestor_sha?: string;
	    ours_sha?: string;
	    theirs_sha?: string;
	    block_count: number;
	
	    static createFrom(source: any = {}) {
	        return new ConflictFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.ancestor_sha = source["ancestor_sha"];
	        this.ours_sha = source["ours_sha"];
	        this.theirs_sha = source["theirs_sha"];
	        this.block_count = source["block_count"];
	    }
	}
	export class DeviceFlowCode {
	    device_code: string;
	    user_code: string;
	    verification_uri: string;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowCode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_code = source["device_code"];
	        this.user_code = source["user_code"];
	        this.verification_uri = source["verification_uri"];
	        this.interval = source["interval"];
	    }
	}
	export class FileChange {
	    type: number;
	    old_path?: string;
	    new_path?: string;
	    additions: number;
	    deletions: number;
	    is_binary: boolean;
	    unified_diff?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.old_path = source["old_path"];
	        this.new_path = source["new_path"];
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	        this.is_binary = source["is_binary"];
	        this.unified_diff = source["unified_diff"];
	    }
	}
	export class Diff {
	    files: FileChange[];
	    total_additions: number;
	    total_deletions: number;
	
	    static createFrom(source: any = {}) {
	        return new Diff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], FileChange);
	        this.total_additions = source["total_additions"];
	        this.total_deletions = source["total_deletions"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FileStatus {
	    path: string;
	    old_path?: string;
	    staged: number;
	    unstaged: number;
	
	    static createFrom(source: any = {}) {
	        return new FileStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.old_path = source["old_path"];
	        this.staged = source["staged"];
	        this.unstaged = source["unstaged"];
	    }
	}
	export class Label {
	    name: string;
	    color: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new Label(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	        this.description = source["description"];
	    }
	}
	export class Issue {
	    number: number;
	    title: string;
	    state: string;
	    author: string;
	    body: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    // Go type: time
	    closed_at?: any;
	    labels?: Label[];
	    assignees?: string[];
	    comments: number;
	    is_pull_request: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.title = source["title"];
	        this.state = source["state"];
	        this.author = source["author"];
	        this.body = source["body"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.closed_at = this.convertValues(source["closed_at"], null);
	        this.labels = this.convertValues(source["labels"], Label);
	        this.assignees = source["assignees"];
	        this.comments = source["comments"];
	        this.is_pull_request = source["is_pull_request"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Step {
	    name: string;
	    status: string;
	    conclusion?: string;
	    number: number;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.conclusion = source["conclusion"];
	        this.number = source["number"];
	    }
	}
	export class Job {
	    id: number;
	    name: string;
	    status: string;
	    conclusion?: string;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    completed_at?: any;
	    runner_name?: string;
	    steps?: Step[];
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.conclusion = source["conclusion"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
	        this.runner_name = source["runner_name"];
	        this.steps = this.convertValues(source["steps"], Step);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MergeConflictDetail {
	    path: string;
	    ours: string;
	    theirs: string;
	    ancestor?: string;
	    raw_content: string;
	    blocks: ConflictBlock[];
	    has_merge: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergeConflictDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.ours = source["ours"];
	        this.theirs = source["theirs"];
	        this.ancestor = source["ancestor"];
	        this.raw_content = source["raw_content"];
	        this.blocks = this.convertValues(source["blocks"], ConflictBlock);
	        this.has_merge = source["has_merge"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Review {
	    id: number;
	    author: string;
	    state: string;
	    body: string;
	    // Go type: time
	    submitted_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Review(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.author = source["author"];
	        this.state = source["state"];
	        this.body = source["body"];
	        this.submitted_at = this.convertValues(source["submitted_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PullRequestDetail {
	    number: number;
	    title: string;
	    state: string;
	    author: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    is_draft: boolean;
	    mergeable: string;
	    head_ref: string;
	    base_ref: string;
	    status_emoji: string;
	    review_state: string;
	    labels?: string[];
	    body: string;
	    // Go type: time
	    closed_at?: any;
	    // Go type: time
	    merged_at?: any;
	    merged_by?: string;
	    additions: number;
	    deletions: number;
	    changed_files: number;
	    commits?: Commit[];
	    reviews?: Review[];
	    check_runs?: CheckRun[];
	    files?: FileChange[];
	    comments: number;
	
	    static createFrom(source: any = {}) {
	        return new PullRequestDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.title = source["title"];
	        this.state = source["state"];
	        this.author = source["author"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.is_draft = source["is_draft"];
	        this.mergeable = source["mergeable"];
	        this.head_ref = source["head_ref"];
	        this.base_ref = source["base_ref"];
	        this.status_emoji = source["status_emoji"];
	        this.review_state = source["review_state"];
	        this.labels = source["labels"];
	        this.body = source["body"];
	        this.closed_at = this.convertValues(source["closed_at"], null);
	        this.merged_at = this.convertValues(source["merged_at"], null);
	        this.merged_by = source["merged_by"];
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	        this.changed_files = source["changed_files"];
	        this.commits = this.convertValues(source["commits"], Commit);
	        this.reviews = this.convertValues(source["reviews"], Review);
	        this.check_runs = this.convertValues(source["check_runs"], CheckRun);
	        this.files = this.convertValues(source["files"], FileChange);
	        this.comments = source["comments"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PullRequestSummary {
	    number: number;
	    title: string;
	    state: string;
	    author: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    is_draft: boolean;
	    mergeable: string;
	    head_ref: string;
	    base_ref: string;
	    status_emoji: string;
	    review_state: string;
	    labels?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PullRequestSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.title = source["title"];
	        this.state = source["state"];
	        this.author = source["author"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.is_draft = source["is_draft"];
	        this.mergeable = source["mergeable"];
	        this.head_ref = source["head_ref"];
	        this.base_ref = source["base_ref"];
	        this.status_emoji = source["status_emoji"];
	        this.review_state = source["review_state"];
	        this.labels = source["labels"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RebaseResult {
	    success: boolean;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new RebaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class ReflogEntry {
	    sequence: number;
	    hash: string;
	    action: string;
	    subject: string;
	    // Go type: time
	    timestamp: any;
	    old_hash?: string;
	    undoable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReflogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sequence = source["sequence"];
	        this.hash = source["hash"];
	        this.action = source["action"];
	        this.subject = source["subject"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.old_hash = source["old_hash"];
	        this.undoable = source["undoable"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Remote {
	    name: string;
	    url: string;
	    push_url?: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Remote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.push_url = source["push_url"];
	        this.type = source["type"];
	    }
	}
	export class Repo {
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
	    // Go type: time
	    created_at?: any;
	    // Go type: time
	    updated_at?: any;
	    html_url?: string;
	    ssh_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.is_bare = source["is_bare"];
	        this.owner = source["owner"];
	        this.name = source["name"];
	        this.full_name = source["full_name"];
	        this.default_branch = source["default_branch"];
	        this.description = source["description"];
	        this.language = source["language"];
	        this.is_private = source["is_private"];
	        this.is_fork = source["is_fork"];
	        this.stars = source["stars"];
	        this.forks = source["forks"];
	        this.open_issues = source["open_issues"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.html_url = source["html_url"];
	        this.ssh_url = source["ssh_url"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Stash {
	    index: number;
	    message: string;
	    hash: string;
	    // Go type: time
	    time: any;
	
	    static createFrom(source: any = {}) {
	        return new Stash(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.message = source["message"];
	        this.hash = source["hash"];
	        this.time = this.convertValues(source["time"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Status {
	    branch: string;
	    upstream?: string;
	    ahead: number;
	    behind: number;
	    files: FileStatus[];
	    staged_count: number;
	    unstaged_count: number;
	    untracked_count: number;
	    is_clean: boolean;
	    is_merging: boolean;
	    is_rebasing: boolean;
	    is_cherry_pick: boolean;
	    is_reverting: boolean;
	    is_bisecting: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.upstream = source["upstream"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.files = this.convertValues(source["files"], FileStatus);
	        this.staged_count = source["staged_count"];
	        this.unstaged_count = source["unstaged_count"];
	        this.untracked_count = source["untracked_count"];
	        this.is_clean = source["is_clean"];
	        this.is_merging = source["is_merging"];
	        this.is_rebasing = source["is_rebasing"];
	        this.is_cherry_pick = source["is_cherry_pick"];
	        this.is_reverting = source["is_reverting"];
	        this.is_bisecting = source["is_bisecting"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class User {
	    login: string;
	    name?: string;
	    email?: string;
	    avatar_url?: string;
	    bio?: string;
	    company?: string;
	    location?: string;
	    plan?: string;
	    followers: number;
	    following: number;
	    public_repos: number;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.name = source["name"];
	        this.email = source["email"];
	        this.avatar_url = source["avatar_url"];
	        this.bio = source["bio"];
	        this.company = source["company"];
	        this.location = source["location"];
	        this.plan = source["plan"];
	        this.followers = source["followers"];
	        this.following = source["following"];
	        this.public_repos = source["public_repos"];
	    }
	}
	export class UserPreferences {
	    appearance: AppearanceConfig;
	    keybindings: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UserPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appearance = this.convertValues(source["appearance"], AppearanceConfig);
	        this.keybindings = source["keybindings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkflowRun {
	    id: number;
	    workflow_name: string;
	    event: string;
	    status: string;
	    conclusion?: string;
	    branch: string;
	    head_sha: string;
	    run_number: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    html_url: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkflowRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workflow_name = source["workflow_name"];
	        this.event = source["event"];
	        this.status = source["status"];
	        this.conclusion = source["conclusion"];
	        this.branch = source["branch"];
	        this.head_sha = source["head_sha"];
	        this.run_number = source["run_number"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.html_url = source["html_url"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

