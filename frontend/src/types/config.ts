export interface SSHConfig {
  host: string
  port: number
  user: string
  password?: string
  privateKey?: string
}

export interface SandboxConfig {
  runtime: 'auto' | 'bwrap' | 'seatbelt'
  rootDir: string
  workDirName: string
  network: boolean
  memoryLimitMB: number
  commandTimeoutSec: number
  bootstrapPython: boolean
  pythonPackages: string[]
}

export interface LLMConfig {
  type: 'openai' | 'deepseek' | 'ark' | 'ollama'
  baseURL: string
  apiKey: string
  model: string
  headers?: Record<string, string>
}

export interface MCPServerConfig {
  name: string
  transport: 'stdio' | 'sse'
  command?: string
  args?: string[]
  url?: string
  enabled: boolean
}

export interface AppSettings {
  ssh: SSHConfig
  sandbox: SandboxConfig
  docker?: {
    image?: string
    memoryLimit?: number
    cpuLimit?: number
    workDir?: string
    network?: boolean
  }
  llm: LLMConfig
  mcp: { servers: MCPServerConfig[] }
  agent: { maxIterations: number }
}

export interface FileInfo {
  name: string
  path: string
  size: number
  modified: string
  preview: string
  isOutput: boolean
}

export interface SandboxStatus {
  sshConnected: boolean
  runtimeAvailable: boolean
  sandboxActive: boolean
  activeSandboxID: string
  activeSandboxName: string
  dockerRunning: boolean
  containerID: string
  activeContainerID: string
  activeContainerName: string
  dockerAvailable: boolean
}
