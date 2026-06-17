export namespace agentctx {

	export class Engine {


	    static createFrom(source: any = {}) {
	        return new Engine(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class TimelineCollector {


	    static createFrom(source: any = {}) {
	        return new TimelineCollector(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}

}

export namespace config {

	export class RuntimeLSPServerConfig {
	    language: string;
	    executable?: string;
	    command?: string[];
	    extensions?: string[];
	    disabled?: boolean;

	    static createFrom(source: any = {}) {
	        return new RuntimeLSPServerConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.executable = source["executable"];
	        this.command = source["command"];
	        this.extensions = source["extensions"];
	        this.disabled = source["disabled"];
	    }
	}
	export class RuntimeLSPConfig {
	    enabled?: boolean;
	    requestTimeoutMs?: number;
	    maxResultBytes?: number;
	    servers?: RuntimeLSPServerConfig[];

	    static createFrom(source: any = {}) {
	        return new RuntimeLSPConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.requestTimeoutMs = source["requestTimeoutMs"];
	        this.maxResultBytes = source["maxResultBytes"];
	        this.servers = this.convertValues(source["servers"], RuntimeLSPServerConfig);
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
	export class WebSearchProviderConfig {
	    name: string;
	    type?: string;
	    endpoint?: string;
	    method?: string;
	    headers?: Record<string, string>;
	    apiKeyEnv?: string;
	    queryParam?: string;
	    limitParam?: string;
	    bodyTemplate?: string;
	    resultsPath?: string;
	    titlePath?: string;
	    urlPath?: string;
	    snippetPath?: string;
	    location?: string;
	    language?: string;
	    page?: number;
	    timeoutMs?: number;
	    maxResults?: number;
	    disabled?: boolean;

	    static createFrom(source: any = {}) {
	        return new WebSearchProviderConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.endpoint = source["endpoint"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.queryParam = source["queryParam"];
	        this.limitParam = source["limitParam"];
	        this.bodyTemplate = source["bodyTemplate"];
	        this.resultsPath = source["resultsPath"];
	        this.titlePath = source["titlePath"];
	        this.urlPath = source["urlPath"];
	        this.snippetPath = source["snippetPath"];
	        this.location = source["location"];
	        this.language = source["language"];
	        this.page = source["page"];
	        this.timeoutMs = source["timeoutMs"];
	        this.maxResults = source["maxResults"];
	        this.disabled = source["disabled"];
	    }
	}
	export class WebSearchConfig {
	    enabled?: boolean;
	    defaultProvider?: string;
	    providers?: WebSearchProviderConfig[];

	    static createFrom(source: any = {}) {
	        return new WebSearchConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.defaultProvider = source["defaultProvider"];
	        this.providers = this.convertValues(source["providers"], WebSearchProviderConfig);
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
	export class SubagentDefinitionConfig {
	    name: string;
	    description: string;
	    instruction?: string;
	    allowedTools?: string[];
	    defaultIsolation?: string;
	    backgroundAllowed?: boolean;

	    static createFrom(source: any = {}) {
	        return new SubagentDefinitionConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.instruction = source["instruction"];
	        this.allowedTools = source["allowedTools"];
	        this.defaultIsolation = source["defaultIsolation"];
	        this.backgroundAllowed = source["backgroundAllowed"];
	    }
	}
	export class AgentRuntimeConfig {
	    engine: string;
	    toolSearchMode: string;
	    agenticProtocol: string;
	    enableBuiltinDeepTransferFallback: boolean;
	    subagents?: SubagentDefinitionConfig[];

	    static createFrom(source: any = {}) {
	        return new AgentRuntimeConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.toolSearchMode = source["toolSearchMode"];
	        this.agenticProtocol = source["agenticProtocol"];
	        this.enableBuiltinDeepTransferFallback = source["enableBuiltinDeepTransferFallback"];
	        this.subagents = this.convertValues(source["subagents"], SubagentDefinitionConfig);
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
	export class AgentConfig {
	    maxIterations: number;
	    runtime: AgentRuntimeConfig;
	    webSearch: WebSearchConfig;
	    lsp: RuntimeLSPConfig;

	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxIterations = source["maxIterations"];
	        this.runtime = this.convertValues(source["runtime"], AgentRuntimeConfig);
	        this.webSearch = this.convertValues(source["webSearch"], WebSearchConfig);
	        this.lsp = this.convertValues(source["lsp"], RuntimeLSPConfig);
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

	export class MCPServerConfig {
	    name: string;
	    transport: string;
	    command?: string;
	    args?: string[];
	    url?: string;
	    env?: Record<string, string>;
	    enabled: boolean;

	    static createFrom(source: any = {}) {
	        return new MCPServerConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.url = source["url"];
	        this.env = source["env"];
	        this.enabled = source["enabled"];
	    }
	}
	export class MCPConfig {
	    servers: MCPServerConfig[];

	    static createFrom(source: any = {}) {
	        return new MCPConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], MCPServerConfig);
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
	export class LLMConfig {
	    type: string;
	    baseURL: string;
	    apiKey: string;
	    model: string;
	    headers?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new LLMConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.baseURL = source["baseURL"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.headers = source["headers"];
	    }
	}
	export class DockerConfig {
	    image: string;
	    memoryLimit: number;
	    cpuLimit: number;
	    workDir: string;
	    network: boolean;

	    static createFrom(source: any = {}) {
	        return new DockerConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.memoryLimit = source["memoryLimit"];
	        this.cpuLimit = source["cpuLimit"];
	        this.workDir = source["workDir"];
	        this.network = source["network"];
	    }
	}
	export class SandboxConfig {
	    runtime: string;
	    rootDir: string;
	    workDirName: string;
	    network: boolean;
	    memoryLimitMB: number;
	    commandTimeoutSec: number;
	    bootstrapPython: boolean;
	    pythonPackages: string[];

	    static createFrom(source: any = {}) {
	        return new SandboxConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtime = source["runtime"];
	        this.rootDir = source["rootDir"];
	        this.workDirName = source["workDirName"];
	        this.network = source["network"];
	        this.memoryLimitMB = source["memoryLimitMB"];
	        this.commandTimeoutSec = source["commandTimeoutSec"];
	        this.bootstrapPython = source["bootstrapPython"];
	        this.pythonPackages = source["pythonPackages"];
	    }
	}
	export class SSHConfig {
	    host: string;
	    port: number;
	    user: string;
	    password?: string;
	    privateKey?: string;

	    static createFrom(source: any = {}) {
	        return new SSHConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.privateKey = source["privateKey"];
	    }
	}
	export class AppConfig {
	    ssh: SSHConfig;
	    sandbox: SandboxConfig;
	    docker?: DockerConfig;
	    llm: LLMConfig;
	    mcp: MCPConfig;
	    agent: AgentConfig;

	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ssh = this.convertValues(source["ssh"], SSHConfig);
	        this.sandbox = this.convertValues(source["sandbox"], SandboxConfig);
	        this.docker = this.convertValues(source["docker"], DockerConfig);
	        this.llm = this.convertValues(source["llm"], LLMConfig);
	        this.mcp = this.convertValues(source["mcp"], MCPConfig);
	        this.agent = this.convertValues(source["agent"], AgentConfig);
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

export namespace model {

	export class Container {
	    id: string;
	    runtimeID?: string;
	    runtime?: string;
	    workspacePath?: string;
	    dockerID?: string;
	    name: string;
	    image?: string;
	    sshHost: string;
	    sshPort: number;
	    status: string;
	    setupComplete: boolean;
	    sessionID: string;
	    createdAt: number;
	    lastUsedAt: number;

	    static createFrom(source: any = {}) {
	        return new Container(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.runtimeID = source["runtimeID"];
	        this.runtime = source["runtime"];
	        this.workspacePath = source["workspacePath"];
	        this.dockerID = source["dockerID"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.sshHost = source["sshHost"];
	        this.sshPort = source["sshPort"];
	        this.status = source["status"];
	        this.setupComplete = source["setupComplete"];
	        this.sessionID = source["sessionID"];
	        this.createdAt = source["createdAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	    }
	}
	export class DeferredAnnouncementState {
	    announcedSearchableCanonicalNames: string[];

	    static createFrom(source: any = {}) {
	        return new DeferredAnnouncementState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.announcedSearchableCanonicalNames = source["announcedSearchableCanonicalNames"];
	    }
	}
	export class DiscoveredToolRecord {
	    canonicalName: string;
	    server: string;
	    kind: string;
	    discoveredAt: number;

	    static createFrom(source: any = {}) {
	        return new DiscoveredToolRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canonicalName = source["canonicalName"];
	        this.server = source["server"];
	        this.kind = source["kind"];
	        this.discoveredAt = source["discoveredAt"];
	    }
	}
	export class DisplayEvent {
	    id: string;
	    type: string;
	    agent?: string;
	    content?: string;
	    toolName?: string;
	    toolArgs?: string;
	    toolId?: string;
	    toolResult?: string;
	    timestamp: number;
	    isStreaming?: boolean;

	    static createFrom(source: any = {}) {
	        return new DisplayEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.agent = source["agent"];
	        this.content = source["content"];
	        this.toolName = source["toolName"];
	        this.toolArgs = source["toolArgs"];
	        this.toolId = source["toolId"];
	        this.toolResult = source["toolResult"];
	        this.timestamp = source["timestamp"];
	        this.isStreaming = source["isStreaming"];
	    }
	}
	export class DisplayTurn {
	    id: string;
	    role: string;
	    content: string;
	    agent?: string;
	    timestamp: number;
	    events: DisplayEvent[];

	    static createFrom(source: any = {}) {
	        return new DisplayTurn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.agent = source["agent"];
	        this.timestamp = source["timestamp"];
	        this.events = this.convertValues(source["events"], DisplayEvent);
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
	export class MCPInstructionsDeltaState {
	    lastAnnouncedSearchableServers: string[];
	    lastAnnouncedPendingServers: string[];
	    lastAnnouncedUnavailableServers: string[];
	    lastInstructionsFingerprint: string;

	    static createFrom(source: any = {}) {
	        return new MCPInstructionsDeltaState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastAnnouncedSearchableServers = source["lastAnnouncedSearchableServers"];
	        this.lastAnnouncedPendingServers = source["lastAnnouncedPendingServers"];
	        this.lastAnnouncedUnavailableServers = source["lastAnnouncedUnavailableServers"];
	        this.lastInstructionsFingerprint = source["lastInstructionsFingerprint"];
	    }
	}
	export class PendingPlanApproval {
	    requestedAt: number;

	    static createFrom(source: any = {}) {
	        return new PendingPlanApproval(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestedAt = source["requestedAt"];
	    }
	}
	export class PendingPlanAttachment {
	    kind: string;
	    markdown: string;
	    feedback?: string;
	    createdAt: number;

	    static createFrom(source: any = {}) {
	        return new PendingPlanAttachment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.markdown = source["markdown"];
	        this.feedback = source["feedback"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class PersistedToolCallFunction {
	    name: string;
	    arguments: string;

	    static createFrom(source: any = {}) {
	        return new PersistedToolCallFunction(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	    }
	}
	export class PersistedToolCall {
	    id: string;
	    function: PersistedToolCallFunction;

	    static createFrom(source: any = {}) {
	        return new PersistedToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.function = this.convertValues(source["function"], PersistedToolCallFunction);
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
	export class PersistedMessage {
	    role: string;
	    content: string;
	    name?: string;
	    toolCallId?: string;
	    toolCalls?: PersistedToolCall[];

	    static createFrom(source: any = {}) {
	        return new PersistedMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.name = source["name"];
	        this.toolCallId = source["toolCallId"];
	        this.toolCalls = this.convertValues(source["toolCalls"], PersistedToolCall);
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


	export class PlanDocument {
	    markdown: string;
	    updatedAt: number;

	    static createFrom(source: any = {}) {
	        return new PlanDocument(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.markdown = source["markdown"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class RunObjective {
	    id: string;
	    runId: string;
	    userMessageId: string;
	    scope: string;
	    objective: string;
	    acceptance?: string;
	    createdAt: number;
	    historyStartIndex?: number;

	    static createFrom(source: any = {}) {
	        return new RunObjective(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.runId = source["runId"];
	        this.userMessageId = source["userMessageId"];
	        this.scope = source["scope"];
	        this.objective = source["objective"];
	        this.acceptance = source["acceptance"];
	        this.createdAt = source["createdAt"];
	        this.historyStartIndex = source["historyStartIndex"];
	    }
	}
	export class RuntimeWorkspaceCompact {
	    active: boolean;
	    workspacePath?: string;
	    originalWorkspace?: string;
	    worktreePath?: string;
	    worktreeBranch?: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeWorkspaceCompact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.workspacePath = source["workspacePath"];
	        this.originalWorkspace = source["originalWorkspace"];
	        this.worktreePath = source["worktreePath"];
	        this.worktreeBranch = source["worktreeBranch"];
	    }
	}
	export class RuntimeTodoItem {
	    id: string;
	    title: string;
	    status: string;
	    depends_on?: string[];

	    static createFrom(source: any = {}) {
	        return new RuntimeTodoItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.depends_on = source["depends_on"];
	    }
	}
	export class RuntimeDiffSummary {
	    filePath: string;
	    toolName?: string;
	    linesAdded?: number;
	    linesRemoved?: number;
	    replacements?: number;
	    patch?: string;
	    summary?: string;
	    updatedAt?: number;

	    static createFrom(source: any = {}) {
	        return new RuntimeDiffSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.toolName = source["toolName"];
	        this.linesAdded = source["linesAdded"];
	        this.linesRemoved = source["linesRemoved"];
	        this.replacements = source["replacements"];
	        this.patch = source["patch"];
	        this.summary = source["summary"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class RuntimeFileReadState {
	    filePath: string;
	    startLine?: number;
	    numLines?: number;
	    totalLines?: number;
	    contentHash?: string;
	    lastReadAt?: number;

	    static createFrom(source: any = {}) {
	        return new RuntimeFileReadState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.startLine = source["startLine"];
	        this.numLines = source["numLines"];
	        this.totalLines = source["totalLines"];
	        this.contentHash = source["contentHash"];
	        this.lastReadAt = source["lastReadAt"];
	    }
	}
	export class RuntimeTaskItemCompact {
	    id: string;
	    sessionId?: string;
	    title: string;
	    description?: string;
	    status: string;
	    owner?: string;
	    priority?: string;
	    depends_on?: string[];
	    createdAt?: number;
	    updatedAt?: number;
	    completedAt?: number;

	    static createFrom(source: any = {}) {
	        return new RuntimeTaskItemCompact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.owner = source["owner"];
	        this.priority = source["priority"];
	        this.depends_on = source["depends_on"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.completedAt = source["completedAt"];
	    }
	}
	export class RuntimeTaskCompact {
	    id: string;
	    sessionId?: string;
	    type?: string;
	    status?: string;
	    description?: string;
	    command?: string;
	    outputPath?: string;
	    outputSize?: number;
	    startedAt?: number;
	    finishedAt?: number;
	    durationMs?: number;
	    exitCode?: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeTaskCompact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.description = source["description"];
	        this.command = source["command"];
	        this.outputPath = source["outputPath"];
	        this.outputSize = source["outputSize"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.durationMs = source["durationMs"];
	        this.exitCode = source["exitCode"];
	        this.error = source["error"];
	    }
	}
	export class RuntimePermissionAudit {
	    requestId?: string;
	    sessionId?: string;
	    toolName: string;
	    toolClass?: string;
	    source?: string;
	    risk?: string;
	    mode?: string;
	    decision: string;
	    reason?: string;
	    input?: string;
	    createdAt?: number;
	    resolvedAt?: number;

	    static createFrom(source: any = {}) {
	        return new RuntimePermissionAudit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.sessionId = source["sessionId"];
	        this.toolName = source["toolName"];
	        this.toolClass = source["toolClass"];
	        this.source = source["source"];
	        this.risk = source["risk"];
	        this.mode = source["mode"];
	        this.decision = source["decision"];
	        this.reason = source["reason"];
	        this.input = source["input"];
	        this.createdAt = source["createdAt"];
	        this.resolvedAt = source["resolvedAt"];
	    }
	}
	export class RuntimePermissionGrant {
	    toolName: string;
	    toolClass?: string;
	    source?: string;
	    decision: string;
	    createdAt: number;

	    static createFrom(source: any = {}) {
	        return new RuntimePermissionGrant(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolName = source["toolName"];
	        this.toolClass = source["toolClass"];
	        this.source = source["source"];
	        this.decision = source["decision"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class RuntimeToolSearchCompact {
	    discoveredTools?: DiscoveredToolRecord[];
	    deferredAnnouncementState?: DeferredAnnouncementState;
	    mcpInstructionsDeltaState?: MCPInstructionsDeltaState;

	    static createFrom(source: any = {}) {
	        return new RuntimeToolSearchCompact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.discoveredTools = this.convertValues(source["discoveredTools"], DiscoveredToolRecord);
	        this.deferredAnnouncementState = this.convertValues(source["deferredAnnouncementState"], DeferredAnnouncementState);
	        this.mcpInstructionsDeltaState = this.convertValues(source["mcpInstructionsDeltaState"], MCPInstructionsDeltaState);
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
	export class RuntimeContextCompact {
	    version: number;
	    createdAt: number;
	    updatedAt: number;
	    originalMessageCount: number;
	    omittedMessageCount: number;
	    tokenEstimate: number;
	    summary: string;
	    toolSearch?: RuntimeToolSearchCompact;
	    permissionGrants?: RuntimePermissionGrant[];
	    permissionAudit?: RuntimePermissionAudit[];
	    tasks?: RuntimeTaskCompact[];
	    taskItems?: RuntimeTaskItemCompact[];
	    fileReadState?: RuntimeFileReadState[];
	    diffSummaries?: RuntimeDiffSummary[];
	    todos?: RuntimeTodoItem[];
	    planDocument?: PlanDocument;
	    workspace?: RuntimeWorkspaceCompact;
	    activeObjective?: RunObjective;

	    static createFrom(source: any = {}) {
	        return new RuntimeContextCompact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.originalMessageCount = source["originalMessageCount"];
	        this.omittedMessageCount = source["omittedMessageCount"];
	        this.tokenEstimate = source["tokenEstimate"];
	        this.summary = source["summary"];
	        this.toolSearch = this.convertValues(source["toolSearch"], RuntimeToolSearchCompact);
	        this.permissionGrants = this.convertValues(source["permissionGrants"], RuntimePermissionGrant);
	        this.permissionAudit = this.convertValues(source["permissionAudit"], RuntimePermissionAudit);
	        this.tasks = this.convertValues(source["tasks"], RuntimeTaskCompact);
	        this.taskItems = this.convertValues(source["taskItems"], RuntimeTaskItemCompact);
	        this.fileReadState = this.convertValues(source["fileReadState"], RuntimeFileReadState);
	        this.diffSummaries = this.convertValues(source["diffSummaries"], RuntimeDiffSummary);
	        this.todos = this.convertValues(source["todos"], RuntimeTodoItem);
	        this.planDocument = this.convertValues(source["planDocument"], PlanDocument);
	        this.workspace = this.convertValues(source["workspace"], RuntimeWorkspaceCompact);
	        this.activeObjective = this.convertValues(source["activeObjective"], RunObjective);
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









	export class Session {
	    id: string;
	    title: string;
	    containers: string[];
	    activeContainerID?: string;
	    workspacePath?: string;
	    createdAt: number;
	    updatedAt: number;
	    messageCount: number;

	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.containers = source["containers"];
	        this.activeContainerID = source["activeContainerID"];
	        this.workspacePath = source["workspacePath"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.messageCount = source["messageCount"];
	    }
	}
	export class StreamingState {
	    partialContent?: string;
	    agentName?: string;

	    static createFrom(source: any = {}) {
	        return new StreamingState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.partialContent = source["partialContent"];
	        this.agentName = source["agentName"];
	    }
	}
	export class SessionData {
	    version: number;
	    messages: PersistedMessage[];
	    display: DisplayTurn[];
	    streaming?: StreamingState;
	    discoveredTools?: DiscoveredToolRecord[];
	    permissionGrants?: RuntimePermissionGrant[];
	    permissionAudit?: RuntimePermissionAudit[];
	    deferredAnnouncementState?: DeferredAnnouncementState;
	    mcpInstructionsDeltaState?: MCPInstructionsDeltaState;
	    runtimeContextCompact?: RuntimeContextCompact;
	    mode?: string;
	    planDocument?: PlanDocument;
	    pendingPlanApproval?: PendingPlanApproval;
	    pendingPlanAttachment?: PendingPlanAttachment;

	    static createFrom(source: any = {}) {
	        return new SessionData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.messages = this.convertValues(source["messages"], PersistedMessage);
	        this.display = this.convertValues(source["display"], DisplayTurn);
	        this.streaming = this.convertValues(source["streaming"], StreamingState);
	        this.discoveredTools = this.convertValues(source["discoveredTools"], DiscoveredToolRecord);
	        this.permissionGrants = this.convertValues(source["permissionGrants"], RuntimePermissionGrant);
	        this.permissionAudit = this.convertValues(source["permissionAudit"], RuntimePermissionAudit);
	        this.deferredAnnouncementState = this.convertValues(source["deferredAnnouncementState"], DeferredAnnouncementState);
	        this.mcpInstructionsDeltaState = this.convertValues(source["mcpInstructionsDeltaState"], MCPInstructionsDeltaState);
	        this.runtimeContextCompact = this.convertValues(source["runtimeContextCompact"], RuntimeContextCompact);
	        this.mode = source["mode"];
	        this.planDocument = this.convertValues(source["planDocument"], PlanDocument);
	        this.pendingPlanApproval = this.convertValues(source["pendingPlanApproval"], PendingPlanApproval);
	        this.pendingPlanAttachment = this.convertValues(source["pendingPlanAttachment"], PendingPlanAttachment);
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

export namespace sandbox {

	export class RuntimeCheckResult {
	    runtime: string;
	    os: string;
	    available: boolean;
	    installable: boolean;
	    missing: string[];
	    message: string;
	    installCommand?: string;
	    workspaceRoot?: string;
	    commandTimeoutSec: number;
	    memoryLimitMB: number;
	    networkEnabled: boolean;
	    pythonBootstrap: boolean;

	    static createFrom(source: any = {}) {
	        return new RuntimeCheckResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtime = source["runtime"];
	        this.os = source["os"];
	        this.available = source["available"];
	        this.installable = source["installable"];
	        this.missing = source["missing"];
	        this.message = source["message"];
	        this.installCommand = source["installCommand"];
	        this.workspaceRoot = source["workspaceRoot"];
	        this.commandTimeoutSec = source["commandTimeoutSec"];
	        this.memoryLimitMB = source["memoryLimitMB"];
	        this.networkEnabled = source["networkEnabled"];
	        this.pythonBootstrap = source["pythonBootstrap"];
	    }
	}
	export class RuntimeInstallResult {
	    runtime: string;
	    os: string;
	    installed: boolean;
	    stdout?: string;
	    stderr?: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeInstallResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtime = source["runtime"];
	        this.os = source["os"];
	        this.installed = source["installed"];
	        this.stdout = source["stdout"];
	        this.stderr = source["stderr"];
	        this.message = source["message"];
	    }
	}
	export class SandboxDiagnosticCheck {
	    id: string;
	    label: string;
	    status: string;
	    message: string;
	    details?: string;
	    command?: string;
	    output?: string;
	    fixIDs?: string[];

	    static createFrom(source: any = {}) {
	        return new SandboxDiagnosticCheck(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.details = source["details"];
	        this.command = source["command"];
	        this.output = source["output"];
	        this.fixIDs = source["fixIDs"];
	    }
	}
	export class SandboxFixSuggestion {
	    id: string;
	    title: string;
	    description: string;
	    risk: string;
	    platform?: string;
	    commands?: string[];
	    copyOnly: boolean;
	    autoRunnable: boolean;

	    static createFrom(source: any = {}) {
	        return new SandboxFixSuggestion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.risk = source["risk"];
	        this.platform = source["platform"];
	        this.commands = source["commands"];
	        this.copyOnly = source["copyOnly"];
	        this.autoRunnable = source["autoRunnable"];
	    }
	}
	export class SandboxDiagnosticsResult {
	    runtime: string;
	    os: string;
	    available: boolean;
	    summary: string;
	    checks: SandboxDiagnosticCheck[];
	    fixes: SandboxFixSuggestion[];
	    workspaceRoot?: string;
	    commandTimeoutSec: number;
	    memoryLimitMB: number;
	    networkEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new SandboxDiagnosticsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtime = source["runtime"];
	        this.os = source["os"];
	        this.available = source["available"];
	        this.summary = source["summary"];
	        this.checks = this.convertValues(source["checks"], SandboxDiagnosticCheck);
	        this.fixes = this.convertValues(source["fixes"], SandboxFixSuggestion);
	        this.workspaceRoot = source["workspaceRoot"];
	        this.commandTimeoutSec = source["commandTimeoutSec"];
	        this.memoryLimitMB = source["memoryLimitMB"];
	        this.networkEnabled = source["networkEnabled"];
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

	export class SandboxManager {


	    static createFrom(source: any = {}) {
	        return new SandboxManager(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}

}

export namespace service {

	export class ChatService {


	    static createFrom(source: any = {}) {
	        return new ChatService(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class DeferredAnnouncementPreview {
	    Mode: string;
	    Added: string[];
	    Removed: string[];
	    WillEmit: boolean;

	    static createFrom(source: any = {}) {
	        return new DeferredAnnouncementPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Mode = source["Mode"];
	        this.Added = source["Added"];
	        this.Removed = source["Removed"];
	        this.WillEmit = source["WillEmit"];
	    }
	}
	export class DeferredInstructionsSummary {
	    SearchableServers: string[];
	    PendingServers: string[];
	    UnavailableServers: string[];
	    Fingerprint: string;
	    WillEmit: boolean;

	    static createFrom(source: any = {}) {
	        return new DeferredInstructionsSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SearchableServers = source["SearchableServers"];
	        this.PendingServers = source["PendingServers"];
	        this.UnavailableServers = source["UnavailableServers"];
	        this.Fingerprint = source["Fingerprint"];
	        this.WillEmit = source["WillEmit"];
	    }
	}
	export class DeferredSurfaceDebug {
	    CurrentConfigDigest: string;
	    BundleConfigDigest: string;
	    BundleGeneration: number;
	    SearchablePoolCanonicalNames: string[];
	    LoadablePoolCanonicalNames: string[];
	    EffectiveDiscoveredCanonicalNames: string[];
	    CurrentLoadedCanonicalNames: string[];
	    ToolSearchCurrentLoadedCanonicalNames: string[];
	    PendingMCPServers: string[];
	    ToolSearchVisible: boolean;
	    AnnouncementState: model.DeferredAnnouncementState;
	    AnnouncementPreview: DeferredAnnouncementPreview;
	    InstructionsState: model.MCPInstructionsDeltaState;
	    InstructionsSummary: DeferredInstructionsSummary;
	    ConfigSnapshotError: string;
	    BuildWarnings: string[];

	    static createFrom(source: any = {}) {
	        return new DeferredSurfaceDebug(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CurrentConfigDigest = source["CurrentConfigDigest"];
	        this.BundleConfigDigest = source["BundleConfigDigest"];
	        this.BundleGeneration = source["BundleGeneration"];
	        this.SearchablePoolCanonicalNames = source["SearchablePoolCanonicalNames"];
	        this.LoadablePoolCanonicalNames = source["LoadablePoolCanonicalNames"];
	        this.EffectiveDiscoveredCanonicalNames = source["EffectiveDiscoveredCanonicalNames"];
	        this.CurrentLoadedCanonicalNames = source["CurrentLoadedCanonicalNames"];
	        this.ToolSearchCurrentLoadedCanonicalNames = source["ToolSearchCurrentLoadedCanonicalNames"];
	        this.PendingMCPServers = source["PendingMCPServers"];
	        this.ToolSearchVisible = source["ToolSearchVisible"];
	        this.AnnouncementState = this.convertValues(source["AnnouncementState"], model.DeferredAnnouncementState);
	        this.AnnouncementPreview = this.convertValues(source["AnnouncementPreview"], DeferredAnnouncementPreview);
	        this.InstructionsState = this.convertValues(source["InstructionsState"], model.MCPInstructionsDeltaState);
	        this.InstructionsSummary = this.convertValues(source["InstructionsSummary"], DeferredInstructionsSummary);
	        this.ConfigSnapshotError = source["ConfigSnapshotError"];
	        this.BuildWarnings = source["BuildWarnings"];
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
	export class EnrichedSession {
	    id: string;
	    title: string;
	    containers: string[];
	    activeContainerID?: string;
	    workspacePath?: string;
	    createdAt: number;
	    updatedAt: number;
	    messageCount: number;
	    containerStatus: string;
	    containerName: string;
	    containerSSH: string;

	    static createFrom(source: any = {}) {
	        return new EnrichedSession(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.containers = source["containers"];
	        this.activeContainerID = source["activeContainerID"];
	        this.workspacePath = source["workspacePath"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.messageCount = source["messageCount"];
	        this.containerStatus = source["containerStatus"];
	        this.containerName = source["containerName"];
	        this.containerSSH = source["containerSSH"];
	    }
	}
	export class FileInfoDTO {
	    name: string;
	    path: string;
	    size: number;
	    modified?: string;
	    isOutput: boolean;

	    static createFrom(source: any = {}) {
	        return new FileInfoDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	        this.isOutput = source["isOutput"];
	    }
	}
	export class InterruptOption {
	    label: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new InterruptOption(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.description = source["description"];
	    }
	}
	export class InterruptEvent {
	    type: string;
	    interruptId: string;
	    checkpointId: string;
	    questions?: string[];
	    options?: InterruptOption[];
	    question?: string;
	    sessionId?: string;

	    static createFrom(source: any = {}) {
	        return new InterruptEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.interruptId = source["interruptId"];
	        this.checkpointId = source["checkpointId"];
	        this.questions = source["questions"];
	        this.options = this.convertValues(source["options"], InterruptOption);
	        this.question = source["question"];
	        this.sessionId = source["sessionId"];
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

	export class MacLocalNetworkCheckResult {
	    platform: string;
	    host: string;
	    port: number;
	    isMac: boolean;
	    isLocalNetworkHost: boolean;
	    appDialOK: boolean;
	    appDialError?: string;
	    likelyPermissionIssue: boolean;
	    summary: string;
	    fixSteps: string[];

	    static createFrom(source: any = {}) {
	        return new MacLocalNetworkCheckResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.isMac = source["isMac"];
	        this.isLocalNetworkHost = source["isLocalNetworkHost"];
	        this.appDialOK = source["appDialOK"];
	        this.appDialError = source["appDialError"];
	        this.likelyPermissionIssue = source["likelyPermissionIssue"];
	        this.summary = source["summary"];
	        this.fixSteps = source["fixSteps"];
	    }
	}
	export class PlatformUIInfo {
	    platform: string;
	    goos: string;
	    appearance: string;
	    supportsTranslucency: boolean;
	    supportsMica: boolean;

	    static createFrom(source: any = {}) {
	        return new PlatformUIInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.goos = source["goos"];
	        this.appearance = source["appearance"];
	        this.supportsTranslucency = source["supportsTranslucency"];
	        this.supportsMica = source["supportsMica"];
	    }
	}
	export class RuntimeLSPConfigured {
	    language: string;
	    executable: string;
	    command: string[];
	    extensions?: string[];
	    disabled?: boolean;

	    static createFrom(source: any = {}) {
	        return new RuntimeLSPConfigured(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.executable = source["executable"];
	        this.command = source["command"];
	        this.extensions = source["extensions"];
	        this.disabled = source["disabled"];
	    }
	}
	export class RuntimeLSPServerStatus {
	    sessionId: string;
	    workspace: string;
	    language: string;
	    executable: string;
	    command: string;
	    alive: boolean;
	    openDocs: number;
	    requestCount: number;
	    startedAt: number;
	    lastUsedAt?: number;
	    lastError?: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeLSPServerStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.workspace = source["workspace"];
	        this.language = source["language"];
	        this.executable = source["executable"];
	        this.command = source["command"];
	        this.alive = source["alive"];
	        this.openDocs = source["openDocs"];
	        this.requestCount = source["requestCount"];
	        this.startedAt = source["startedAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.lastError = source["lastError"];
	    }
	}
	export class RuntimeLSPStatus {
	    enabled: boolean;
	    requestTimeoutMs: number;
	    maxResultBytes: number;
	    configured: RuntimeLSPConfigured[];
	    servers: RuntimeLSPServerStatus[];

	    static createFrom(source: any = {}) {
	        return new RuntimeLSPStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.requestTimeoutMs = source["requestTimeoutMs"];
	        this.maxResultBytes = source["maxResultBytes"];
	        this.configured = this.convertValues(source["configured"], RuntimeLSPConfigured);
	        this.servers = this.convertValues(source["servers"], RuntimeLSPServerStatus);
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
	export class RuntimeWorktreeStateDTO {
	    sessionId: string;
	    active: boolean;
	    workspacePath?: string;
	    currentPath?: string;
	    worktreePath?: string;
	    worktreeBranch?: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeWorktreeStateDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.active = source["active"];
	        this.workspacePath = source["workspacePath"];
	        this.currentPath = source["currentPath"];
	        this.worktreePath = source["worktreePath"];
	        this.worktreeBranch = source["worktreeBranch"];
	        this.message = source["message"];
	    }
	}
	export class SandboxStatusDTO {
	    sshConnected: boolean;
	    runtimeAvailable: boolean;
	    sandboxActive: boolean;
	    activeSandboxID: string;
	    activeSandboxName: string;
	    dockerRunning: boolean;
	    containerID: string;
	    activeContainerID: string;
	    activeContainerName: string;
	    dockerAvailable: boolean;

	    static createFrom(source: any = {}) {
	        return new SandboxStatusDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sshConnected = source["sshConnected"];
	        this.runtimeAvailable = source["runtimeAvailable"];
	        this.sandboxActive = source["sandboxActive"];
	        this.activeSandboxID = source["activeSandboxID"];
	        this.activeSandboxName = source["activeSandboxName"];
	        this.dockerRunning = source["dockerRunning"];
	        this.containerID = source["containerID"];
	        this.activeContainerID = source["activeContainerID"];
	        this.activeContainerName = source["activeContainerName"];
	        this.dockerAvailable = source["dockerAvailable"];
	    }
	}
	export class SessionRun {


	    static createFrom(source: any = {}) {
	        return new SessionRun(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class SessionService {


	    static createFrom(source: any = {}) {
	        return new SessionService(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}
	export class SessionSnapshot {
	    SessionData?: model.SessionData;
	    MessageCount: number;
	    HasSessionRun: boolean;
	    DeferredSurfaceDebug?: DeferredSurfaceDebug;

	    static createFrom(source: any = {}) {
	        return new SessionSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SessionData = this.convertValues(source["SessionData"], model.SessionData);
	        this.MessageCount = source["MessageCount"];
	        this.HasSessionRun = source["HasSessionRun"];
	        this.DeferredSurfaceDebug = this.convertValues(source["DeferredSurfaceDebug"], DeferredSurfaceDebug);
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
	export class TerminalCommandResult {
	    command: string;
	    stdout: string;
	    stderr: string;
	    exitCode: number;

	    static createFrom(source: any = {}) {
	        return new TerminalCommandResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.stdout = source["stdout"];
	        this.stderr = source["stderr"];
	        this.exitCode = source["exitCode"];
	    }
	}
	export class WebSearchDiagnosticCheck {
	    id: string;
	    label: string;
	    status: string;
	    message: string;
	    details?: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchDiagnosticCheck(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.details = source["details"];
	    }
	}
	export class WebSearchProviderDiagnostic {
	    name: string;
	    type: string;
	    endpoint: string;
	    status: string;
	    message: string;
	    apiKeyEnv?: string;
	    apiKeyPresent?: boolean;
	    publicEndpoint: boolean;

	    static createFrom(source: any = {}) {
	        return new WebSearchProviderDiagnostic(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.endpoint = source["endpoint"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.apiKeyPresent = source["apiKeyPresent"];
	        this.publicEndpoint = source["publicEndpoint"];
	    }
	}
	export class WebSearchDiagnosticsResult {
	    enabled: boolean;
	    defaultProvider: string;
	    available: boolean;
	    summary: string;
	    checks: WebSearchDiagnosticCheck[];
	    providers: WebSearchProviderDiagnostic[];

	    static createFrom(source: any = {}) {
	        return new WebSearchDiagnosticsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.defaultProvider = source["defaultProvider"];
	        this.available = source["available"];
	        this.summary = source["summary"];
	        this.checks = this.convertValues(source["checks"], WebSearchDiagnosticCheck);
	        this.providers = this.convertValues(source["providers"], WebSearchProviderDiagnostic);
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

	export class WebSearchSmokeResult {
	    ok: boolean;
	    query: string;
	    provider: string;
	    url: string;
	    results: string[];
	    resultCount: number;
	    durationMs: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new WebSearchSmokeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.query = source["query"];
	        this.provider = source["provider"];
	        this.url = source["url"];
	        this.results = source["results"];
	        this.resultCount = source["resultCount"];
	        this.durationMs = source["durationMs"];
	        this.message = source["message"];
	    }
	}
	export class WorkspaceCleanupResultDTO {
	    tmpPath: string;
	    removedEntries: number;
	    reclaimedBytes: number;

	    static createFrom(source: any = {}) {
	        return new WorkspaceCleanupResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tmpPath = source["tmpPath"];
	        this.removedEntries = source["removedEntries"];
	        this.reclaimedBytes = source["reclaimedBytes"];
	    }
	}
	export class WorkspaceInfoDTO {
	    sshConnected: boolean;
	    active: boolean;
	    activeContainerID?: string;
	    sandboxID?: string;
	    sandboxName?: string;
	    runtime?: string;
	    workspacePath?: string;
	    sshHost?: string;
	    sshPort?: number;
	    fileCount: number;
	    totalSize: number;
	    refreshedAt: number;

	    static createFrom(source: any = {}) {
	        return new WorkspaceInfoDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sshConnected = source["sshConnected"];
	        this.active = source["active"];
	        this.activeContainerID = source["activeContainerID"];
	        this.sandboxID = source["sandboxID"];
	        this.sandboxName = source["sandboxName"];
	        this.runtime = source["runtime"];
	        this.workspacePath = source["workspacePath"];
	        this.sshHost = source["sshHost"];
	        this.sshPort = source["sshPort"];
	        this.fileCount = source["fileCount"];
	        this.totalSize = source["totalSize"];
	        this.refreshedAt = source["refreshedAt"];
	    }
	}

}
export namespace tools {

	export class RuntimeTaskOutput {
	    taskId: string;
	    status: string;
	    outputPath?: string;
	    content: string;
	    offset: number;
	    nextOffset: number;
	    size: number;
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new RuntimeTaskOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.status = source["status"];
	        this.outputPath = source["outputPath"];
	        this.content = source["content"];
	        this.offset = source["offset"];
	        this.nextOffset = source["nextOffset"];
	        this.size = source["size"];
	        this.truncated = source["truncated"];
	    }
	}
	export class RuntimeTaskSnapshot {
	    id: string;
	    sessionId: string;
	    type: string;
	    status: string;
	    description: string;
	    command?: string;
	    outputPath?: string;
	    outputSize?: number;
	    startedAt: number;
	    finishedAt?: number;
	    durationMs?: number;
	    exitCode?: number;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new RuntimeTaskSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.description = source["description"];
	        this.command = source["command"];
	        this.outputPath = source["outputPath"];
	        this.outputSize = source["outputSize"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.durationMs = source["durationMs"];
	        this.exitCode = source["exitCode"];
	        this.error = source["error"];
	    }
	}
	export class ToolPermissionRequest {
	    requestId: string;
	    sessionId?: string;
	    toolName: string;
	    title?: string;
	    description?: string;
	    toolClass?: string;
	    source?: string;
	    risk: string;
	    input?: string;
	    createdAt: number;

	    static createFrom(source: any = {}) {
	        return new ToolPermissionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.sessionId = source["sessionId"];
	        this.toolName = source["toolName"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.toolClass = source["toolClass"];
	        this.source = source["source"];
	        this.risk = source["risk"];
	        this.input = source["input"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class WorktreeDiffOutput {
	    workspacePath: string;
	    worktreePath: string;
	    worktreeBranch: string;
	    status: string;
	    diffStat: string;
	    diff?: string;
	    untrackedDiff?: string;
	    truncated?: boolean;
	    untrackedTruncated?: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new WorktreeDiffOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspacePath = source["workspacePath"];
	        this.worktreePath = source["worktreePath"];
	        this.worktreeBranch = source["worktreeBranch"];
	        this.status = source["status"];
	        this.diffStat = source["diffStat"];
	        this.diff = source["diff"];
	        this.untrackedDiff = source["untrackedDiff"];
	        this.truncated = source["truncated"];
	        this.untrackedTruncated = source["untrackedTruncated"];
	        this.message = source["message"];
	    }
	}
	export class WorktreeMergeOutput {
	    action: string;
	    workspacePath: string;
	    worktreePath: string;
	    worktreeBranch: string;
	    commitMessage?: string;
	    removed: boolean;
	    conflicted?: boolean;
	    conflictFiles?: string[];
	    mergeOutput?: string;
	    recoveryHint?: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new WorktreeMergeOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.workspacePath = source["workspacePath"];
	        this.worktreePath = source["worktreePath"];
	        this.worktreeBranch = source["worktreeBranch"];
	        this.commitMessage = source["commitMessage"];
	        this.removed = source["removed"];
	        this.conflicted = source["conflicted"];
	        this.conflictFiles = source["conflictFiles"];
	        this.mergeOutput = source["mergeOutput"];
	        this.recoveryHint = source["recoveryHint"];
	        this.message = source["message"];
	    }
	}
	export class WorktreeOutput {
	    action: string;
	    workspacePath: string;
	    worktreePath?: string;
	    worktreeBranch?: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new WorktreeOutput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.workspacePath = source["workspacePath"];
	        this.worktreePath = source["worktreePath"];
	        this.worktreeBranch = source["worktreeBranch"];
	        this.message = source["message"];
	    }
	}

}
