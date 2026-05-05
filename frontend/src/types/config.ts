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
  modified?: string
  preview?: string
  isOutput: boolean
}

export interface SandboxDiagnosticCheck {
  id: string
  label: string
  status: 'pass' | 'warn' | 'fail' | 'info' | 'skipped'
  message: string
  details?: string
  command?: string
  output?: string
  fixIDs?: string[]
}

export interface SandboxFixSuggestion {
  id: string
  title: string
  description: string
  risk: 'safe' | 'sudo' | 'security'
  platform?: string
  commands?: string[]
  copyOnly: boolean
  autoRunnable: boolean
}

export interface SandboxDiagnosticsResult {
  runtime: string
  os: string
  available: boolean
  summary: string
  checks: SandboxDiagnosticCheck[]
  fixes: SandboxFixSuggestion[]
  workspaceRoot?: string
  commandTimeoutSec: number
  memoryLimitMB: number
  networkEnabled: boolean
}

export interface WorkspaceInfo {
  sshConnected: boolean
  active: boolean
  sandboxID?: string
  sandboxName?: string
  runtime?: string
  workspacePath?: string
  sshHost?: string
  sshPort?: number
  fileCount: number
  totalSize: number
  refreshedAt: number
}

export interface WorkspaceCleanupResult {
  tmpPath: string
  removedEntries: number
  reclaimedBytes: number
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
