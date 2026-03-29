export namespace app {
	
	export class AgentCard {
	    id: string;
	    name: string;
	    tagline: string;
	    description: string;
	    mode: string;
	    built_in: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tagline = source["tagline"];
	        this.description = source["description"];
	        this.mode = source["mode"];
	        this.built_in = source["built_in"];
	    }
	}
	export class ArtifactSummary {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtifactSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class ChatLiveState {
	    turn: number;
	    artifacts?: Record<string, string>;
	    artifact_previews?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ChatLiveState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turn = source["turn"];
	        this.artifacts = source["artifacts"];
	        this.artifact_previews = source["artifact_previews"];
	    }
	}
	export class ChatMessage {
	    turn: number;
	    role: string;
	    content: string;
	    created_at: string;
	    artifacts?: Record<string, string>;
	    artifact_previews?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turn = source["turn"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	        this.artifacts = source["artifacts"];
	        this.artifact_previews = source["artifact_previews"];
	    }
	}
	export class ChatSummary {
	    id: string;
	    provider: string;
	    mode: string;
	    status: string;
	    created_at: string;
	    updated_at: string;
	    provider_session_id?: string;
	    last_user_message?: string;
	    last_assistant_message?: string;
	    message_count: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.provider_session_id = source["provider_session_id"];
	        this.last_user_message = source["last_user_message"];
	        this.last_assistant_message = source["last_assistant_message"];
	        this.message_count = source["message_count"];
	    }
	}
	export class ChatDetail {
	    session: ChatSummary;
	    messages: ChatMessage[];
	    metadata?: Record<string, string>;
	    live?: ChatLiveState;
	
	    static createFrom(source: any = {}) {
	        return new ChatDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], ChatSummary);
	        this.messages = this.convertValues(source["messages"], ChatMessage);
	        this.metadata = source["metadata"];
	        this.live = this.convertValues(source["live"], ChatLiveState);
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
	
	
	
	export class DocsStatus {
	    ready: boolean;
	    generated_maps: string[];
	
	    static createFrom(source: any = {}) {
	        return new DocsStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.generated_maps = source["generated_maps"];
	    }
	}
	export class PresetCard {
	    id: string;
	    name: string;
	    tagline: string;
	    goal: string;
	    adapter: string;
	    category: string;
	    version: string;
	    author_name: string;
	    trust: string;
	    files?: string[];
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tagline = source["tagline"];
	        this.goal = source["goal"];
	        this.adapter = source["adapter"];
	        this.category = source["category"];
	        this.version = source["version"];
	        this.author_name = source["author_name"];
	        this.trust = source["trust"];
	        this.files = source["files"];
	        this.path = source["path"];
	    }
	}
	export class ProviderHealth {
	    name: string;
	    installed: boolean;
	    binary_path?: string;
	    capabilities?: string[];
	    notes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProviderHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.installed = source["installed"];
	        this.binary_path = source["binary_path"];
	        this.capabilities = source["capabilities"];
	        this.notes = source["notes"];
	    }
	}
	export class InstalledPresetSummary {
	    install_id: string;
	    preset_id: string;
	    version: string;
	    status: string;
	    installed_at: string;
	    report_path?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstalledPresetSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.install_id = source["install_id"];
	        this.preset_id = source["preset_id"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.installed_at = source["installed_at"];
	        this.report_path = source["report_path"];
	    }
	}
	export class QuestionEntry {
	    id: string;
	    summary: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestionEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.status = source["status"];
	    }
	}
	export class RunSummary {
	    id: string;
	    task: string;
	    status: string;
	    state: string;
	    mode: string;
	    provider: string;
	    started_at: string;
	    updated_at: string;
	    dry_run: boolean;
	    artifact_count: number;
	
	    static createFrom(source: any = {}) {
	        return new RunSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task = source["task"];
	        this.status = source["status"];
	        this.state = source["state"];
	        this.mode = source["mode"];
	        this.provider = source["provider"];
	        this.started_at = source["started_at"];
	        this.updated_at = source["updated_at"];
	        this.dry_run = source["dry_run"];
	        this.artifact_count = source["artifact_count"];
	    }
	}
	export class IndexStatus {
	    ready: boolean;
	    files: number;
	    symbols: number;
	    dependencies: number;
	    docs: number;
	
	    static createFrom(source: any = {}) {
	        return new IndexStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.files = source["files"];
	        this.symbols = source["symbols"];
	        this.dependencies = source["dependencies"];
	        this.docs = source["docs"];
	    }
	}
	export class WorkspaceSummary {
	    root: string;
	    name: string;
	    arc_dir: string;
	    default_provider: string;
	    enabled_providers: string[];
	    mode: string;
	    autonomy: string;
	    index: IndexStatus;
	    memory: memory.Summary;
	    docs: DocsStatus;
	    last_run?: RunSummary;
	    questions?: QuestionEntry[];
	    installed_presets?: InstalledPresetSummary[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.name = source["name"];
	        this.arc_dir = source["arc_dir"];
	        this.default_provider = source["default_provider"];
	        this.enabled_providers = source["enabled_providers"];
	        this.mode = source["mode"];
	        this.autonomy = source["autonomy"];
	        this.index = this.convertValues(source["index"], IndexStatus);
	        this.memory = this.convertValues(source["memory"], memory.Summary);
	        this.docs = this.convertValues(source["docs"], DocsStatus);
	        this.last_run = this.convertValues(source["last_run"], RunSummary);
	        this.questions = this.convertValues(source["questions"], QuestionEntry);
	        this.installed_presets = this.convertValues(source["installed_presets"], InstalledPresetSummary);
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
	export class HomeSnapshot {
	    workspace?: WorkspaceSummary;
	    providers: ProviderHealth[];
	    runs: RunSummary[];
	    presets: PresetCard[];
	    installed?: InstalledPresetSummary[];
	    chats?: ChatSummary[];
	
	    static createFrom(source: any = {}) {
	        return new HomeSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], WorkspaceSummary);
	        this.providers = this.convertValues(source["providers"], ProviderHealth);
	        this.runs = this.convertValues(source["runs"], RunSummary);
	        this.presets = this.convertValues(source["presets"], PresetCard);
	        this.installed = this.convertValues(source["installed"], InstalledPresetSummary);
	        this.chats = this.convertValues(source["chats"], ChatSummary);
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
	
	
	export class LiveAppDetail {
	    id: string;
	    session_id?: string;
	    title: string;
	    origin?: string;
	    type: string;
	    status: string;
	    port?: number;
	    preview_url?: string;
	    started_at?: string;
	    updated_at?: string;
	    auto_stop_policy?: string;
	    stop_reason?: string;
	    source_path?: string;
	    stdout_path?: string;
	    stderr_path?: string;
	    command?: string[];
	
	    static createFrom(source: any = {}) {
	        return new LiveAppDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.session_id = source["session_id"];
	        this.title = source["title"];
	        this.origin = source["origin"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.port = source["port"];
	        this.preview_url = source["preview_url"];
	        this.started_at = source["started_at"];
	        this.updated_at = source["updated_at"];
	        this.auto_stop_policy = source["auto_stop_policy"];
	        this.stop_reason = source["stop_reason"];
	        this.source_path = source["source_path"];
	        this.stdout_path = source["stdout_path"];
	        this.stderr_path = source["stderr_path"];
	        this.command = source["command"];
	    }
	}
	export class LiveAppSummary {
	    id: string;
	    session_id?: string;
	    title: string;
	    origin?: string;
	    type: string;
	    status: string;
	    port?: number;
	    preview_url?: string;
	    started_at?: string;
	    updated_at?: string;
	    auto_stop_policy?: string;
	    stop_reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new LiveAppSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.session_id = source["session_id"];
	        this.title = source["title"];
	        this.origin = source["origin"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.port = source["port"];
	        this.preview_url = source["preview_url"];
	        this.started_at = source["started_at"];
	        this.updated_at = source["updated_at"];
	        this.auto_stop_policy = source["auto_stop_policy"];
	        this.stop_reason = source["stop_reason"];
	    }
	}
	
	export class PresetPreview {
	    manifest: PresetCard;
	    files: string[];
	    readme?: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifest = this.convertValues(source["manifest"], PresetCard);
	        this.files = source["files"];
	        this.readme = source["readme"];
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
	export class ProjectState {
	    path: string;
	    name: string;
	    state: string;
	    message?: string;
	    workspace?: WorkspaceSummary;
	
	    static createFrom(source: any = {}) {
	        return new ProjectState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.message = source["message"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceSummary);
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
	
	
	export class RunDetail {
	    run: RunSummary;
	    artifacts: ArtifactSummary[];
	    changed_files?: string[];
	    previews?: Record<string, string>;
	    metadata?: Record<string, string>;
	    docs?: Record<string, string>;
	    review?: Record<string, string>;
	    verify?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RunDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run = this.convertValues(source["run"], RunSummary);
	        this.artifacts = this.convertValues(source["artifacts"], ArtifactSummary);
	        this.changed_files = source["changed_files"];
	        this.previews = source["previews"];
	        this.metadata = source["metadata"];
	        this.docs = source["docs"];
	        this.review = source["review"];
	        this.verify = source["verify"];
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
	
	export class SessionMaterialCard {
	    id: string;
	    type: string;
	    title: string;
	    summary: string;
	    source: string;
	    preview?: string;
	    path?: string;
	    url?: string;
	    files?: string[];
	    open_label?: string;
	    launchable?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionMaterialCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.source = source["source"];
	        this.preview = source["preview"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.files = source["files"];
	        this.open_label = source["open_label"];
	        this.launchable = source["launchable"];
	    }
	}
	export class SessionSummary {
	    id: string;
	    title: string;
	    summary: string;
	    agent_id: string;
	    agent_name: string;
	    mode: string;
	    status: string;
	    created_at: string;
	    updated_at: string;
	    last_user_message?: string;
	    last_assistant_message?: string;
	    related_run_ids?: string[];
	    material_count: number;
	
	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.agent_id = source["agent_id"];
	        this.agent_name = source["agent_name"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.last_user_message = source["last_user_message"];
	        this.last_assistant_message = source["last_assistant_message"];
	        this.related_run_ids = source["related_run_ids"];
	        this.material_count = source["material_count"];
	    }
	}
	export class SessionDetail {
	    session: SessionSummary;
	    messages: ChatMessage[];
	    runs?: RunSummary[];
	    materials?: SessionMaterialCard[];
	    live_apps?: LiveAppSummary[];
	    metadata?: Record<string, string>;
	    live?: ChatLiveState;
	    next_action?: string;
	    project_root?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], SessionSummary);
	        this.messages = this.convertValues(source["messages"], ChatMessage);
	        this.runs = this.convertValues(source["runs"], RunSummary);
	        this.materials = this.convertValues(source["materials"], SessionMaterialCard);
	        this.live_apps = this.convertValues(source["live_apps"], LiveAppSummary);
	        this.metadata = source["metadata"];
	        this.live = this.convertValues(source["live"], ChatLiveState);
	        this.next_action = source["next_action"];
	        this.project_root = source["project_root"];
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
	
	
	export class WorkspaceChange {
	    hash: string;
	    date: string;
	    author: string;
	    subject: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.date = source["date"];
	        this.author = source["author"];
	        this.subject = source["subject"];
	    }
	}
	export class WorkspaceGitChange {
	    path: string;
	    status: string;
	    diff_preview?: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceGitChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.diff_preview = source["diff_preview"];
	    }
	}
	export class WorkspaceFileEntry {
	    path: string;
	    kind: string;
	    size: number;
	    mod_time: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceFileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.mod_time = source["mod_time"];
	    }
	}
	export class WorkspaceExplorer {
	    files: WorkspaceFileEntry[];
	    recent_changes?: WorkspaceChange[];
	    dirty_files?: WorkspaceGitChange[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceExplorer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], WorkspaceFileEntry);
	        this.recent_changes = this.convertValues(source["recent_changes"], WorkspaceChange);
	        this.dirty_files = this.convertValues(source["dirty_files"], WorkspaceGitChange);
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
	export class WorkspaceSymbol {
	    name: string;
	    kind: string;
	    language: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSymbol(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.language = source["language"];
	        this.line = source["line"];
	    }
	}
	export class WorkspaceFileDetail {
	    path: string;
	    kind: string;
	    size: number;
	    mod_time: string;
	    content?: string;
	    truncated: boolean;
	    editable: boolean;
	    symbols?: WorkspaceSymbol[];
	    doc_title?: string;
	    doc_headings?: string[];
	    recent_change?: WorkspaceChange;
	    git_change?: WorkspaceGitChange;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceFileDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.mod_time = source["mod_time"];
	        this.content = source["content"];
	        this.truncated = source["truncated"];
	        this.editable = source["editable"];
	        this.symbols = this.convertValues(source["symbols"], WorkspaceSymbol);
	        this.doc_title = source["doc_title"];
	        this.doc_headings = source["doc_headings"];
	        this.recent_change = this.convertValues(source["recent_change"], WorkspaceChange);
	        this.git_change = this.convertValues(source["git_change"], WorkspaceGitChange);
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

export namespace desktop {
	
	export class ChatSendRequest {
	    path: string;
	    session_id: string;
	    model: string;
	    prompt: string;
	    dry_run: boolean;
	    async: boolean;
	    attach_session_ids?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ChatSendRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.session_id = source["session_id"];
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.dry_run = source["dry_run"];
	        this.async = source["async"];
	        this.attach_session_ids = source["attach_session_ids"];
	    }
	}
	export class ChatStartRequest {
	    path: string;
	    provider: string;
	    mode: string;
	    model: string;
	    prompt: string;
	    dry_run: boolean;
	    async: boolean;
	    attach_session_ids?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ChatStartRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.provider = source["provider"];
	        this.mode = source["mode"];
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.dry_run = source["dry_run"];
	        this.async = source["async"];
	        this.attach_session_ids = source["attach_session_ids"];
	    }
	}
	export class LiveAppStartRequest {
	    path: string;
	    session_id?: string;
	    material_id?: string;
	    lesson_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new LiveAppStartRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.session_id = source["session_id"];
	        this.material_id = source["material_id"];
	        this.lesson_id = source["lesson_id"];
	    }
	}
	export class LiveAppStopRequest {
	    path: string;
	    app_id: string;
	
	    static createFrom(source: any = {}) {
	        return new LiveAppStopRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.app_id = source["app_id"];
	    }
	}
	export class PresetInstallRequest {
	    path: string;
	    id: string;
	    allow_overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PresetInstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.id = source["id"];
	        this.allow_overwrite = source["allow_overwrite"];
	    }
	}
	export class PresetRollbackRequest {
	    path: string;
	    install_id: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetRollbackRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.install_id = source["install_id"];
	    }
	}
	export class TaskPlanRequest {
	    path: string;
	    task: string;
	    mode: string;
	    provider: string;
	    session_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskPlanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.task = source["task"];
	        this.mode = source["mode"];
	        this.provider = source["provider"];
	        this.session_id = source["session_id"];
	    }
	}
	export class TaskReviewRequest {
	    path: string;
	    run_id: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskReviewRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.run_id = source["run_id"];
	    }
	}
	export class TaskRunRequest {
	    path: string;
	    task: string;
	    mode: string;
	    provider: string;
	    dry_run: boolean;
	    run_checks: boolean;
	    session_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskRunRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.task = source["task"];
	        this.mode = source["mode"];
	        this.provider = source["provider"];
	        this.dry_run = source["dry_run"];
	        this.run_checks = source["run_checks"];
	        this.session_id = source["session_id"];
	    }
	}
	export class WorkspaceFileSaveRequest {
	    path: string;
	    file: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceFileSaveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.file = source["file"];
	        this.content = source["content"];
	    }
	}
	export class WorkspaceInitRequest {
	    path: string;
	    provider: string;
	    mode: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceInitRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.provider = source["provider"];
	        this.mode = source["mode"];
	    }
	}

}

export namespace memory {
	
	export class Summary {
	    total: number;
	    by_status: Record<string, number>;
	    by_kind: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.by_status = source["by_status"];
	        this.by_kind = source["by_kind"];
	    }
	}

}

export namespace presets {
	
	export class Author {
	    name: string;
	    handle: string;
	
	    static createFrom(source: any = {}) {
	        return new Author(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.handle = source["handle"];
	    }
	}
	export class FileOperation {
	    target_path: string;
	    source_path: string;
	    action: string;
	    collision: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_path = source["target_path"];
	        this.source_path = source["source_path"];
	        this.action = source["action"];
	        this.collision = source["collision"];
	    }
	}
	export class Manifest {
	    id: string;
	    name: string;
	    tagline: string;
	    goal: string;
	    adapter: string;
	    category: string;
	    persona: string;
	    version: string;
	    files: string[];
	    safety_notes: string[];
	    author: Author;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tagline = source["tagline"];
	        this.goal = source["goal"];
	        this.adapter = source["adapter"];
	        this.category = source["category"];
	        this.persona = source["persona"];
	        this.version = source["version"];
	        this.files = source["files"];
	        this.safety_notes = source["safety_notes"];
	        this.author = this.convertValues(source["author"], Author);
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
	export class InstallPreview {
	    manifest: Manifest;
	    target: string;
	    operations: FileOperation[];
	    has_conflicts: boolean;
	    conflicts?: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifest = this.convertValues(source["manifest"], Manifest);
	        this.target = source["target"];
	        this.operations = this.convertValues(source["operations"], FileOperation);
	        this.has_conflicts = source["has_conflicts"];
	        this.conflicts = source["conflicts"];
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
	export class InstalledRecord {
	    install_id: string;
	    preset_id: string;
	    version: string;
	    target: string;
	    installed_at: string;
	    status: string;
	    allow_overwrite: boolean;
	    operations: FileOperation[];
	    report_path: string;
	    backup_dir?: string;
	    rolled_back_at?: string;
	    rollback_summary?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstalledRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.install_id = source["install_id"];
	        this.preset_id = source["preset_id"];
	        this.version = source["version"];
	        this.target = source["target"];
	        this.installed_at = source["installed_at"];
	        this.status = source["status"];
	        this.allow_overwrite = source["allow_overwrite"];
	        this.operations = this.convertValues(source["operations"], FileOperation);
	        this.report_path = source["report_path"];
	        this.backup_dir = source["backup_dir"];
	        this.rolled_back_at = source["rolled_back_at"];
	        this.rollback_summary = source["rollback_summary"];
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
	export class InstallResult {
	    preview: InstallPreview;
	    record: InstalledRecord;
	    report: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preview = this.convertValues(source["preview"], InstallPreview);
	        this.record = this.convertValues(source["record"], InstalledRecord);
	        this.report = source["report"];
	        this.warnings = source["warnings"];
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

