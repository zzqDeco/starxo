export namespace agentctx {
	
	export class Engine {
	
	
	    static createFrom(source: any = {}) {
	        return new Engine(source);
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
	export class SessionService {
	
	
	    static createFrom(source: any = {}) {
	        return new SessionService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

