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
	
	export class AgentConfig {
	    maxIterations: number;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxIterations = source["maxIterations"];
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
	    docker: DockerConfig;
	    llm: LLMConfig;
	    mcp: MCPConfig;
	    agent: AgentConfig;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ssh = this.convertValues(source["ssh"], SSHConfig);
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
	    dockerID: string;
	    name: string;
	    image: string;
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
	    deferredAnnouncementState?: DeferredAnnouncementState;
	    mcpInstructionsDeltaState?: MCPInstructionsDeltaState;
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
	        this.deferredAnnouncementState = this.convertValues(source["deferredAnnouncementState"], DeferredAnnouncementState);
	        this.mcpInstructionsDeltaState = this.convertValues(source["mcpInstructionsDeltaState"], MCPInstructionsDeltaState);
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
	    isOutput: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
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
	
	export class SandboxStatusDTO {
	    sshConnected: boolean;
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

}

