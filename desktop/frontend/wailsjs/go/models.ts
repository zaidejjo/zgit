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
	export class Commit {
	    hash: string;
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
	        this.followers = source["followers"];
	        this.following = source["following"];
	        this.public_repos = source["public_repos"];
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

