export interface Session {
  id: string
  title: string
  containers: string[]
  activeContainerID: string
  workspacePath?: string
  createdAt: number
  updatedAt: number
  messageCount: number
  // Enriched container info (from ListSessionsEnriched, for active container)
  containerStatus?: 'running' | 'stopped' | 'unknown' | 'destroyed' | 'unavailable' | ''
  containerName?: string
  containerSSH?: string
}

export interface ContainerInfo {
  id: string
  runtimeID?: string
  runtime?: string
  workspacePath?: string
  dockerID: string
  name: string
  image: string
  sshHost: string
  sshPort: number
  status: 'running' | 'stopped' | 'unknown' | 'destroyed' | 'unavailable'
  setupComplete: boolean
  sessionID: string
  createdAt: number
  lastUsedAt: number
}
