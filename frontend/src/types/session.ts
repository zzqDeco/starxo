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
  containerStatus?: 'running' | 'stopped' | 'unknown' | 'destroyed' | ''
  containerName?: string
  containerSSH?: string
}

export interface ContainerInfo {
  id: string
  dockerID: string
  name: string
  image: string
  sshHost: string
  sshPort: number
  status: 'running' | 'stopped' | 'unknown' | 'destroyed'
  setupComplete: boolean
  sessionID: string
  createdAt: number
  lastUsedAt: number
}
