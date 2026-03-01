export interface Session {
  id: string
  title: string
  containerID: string
  workspacePath?: string
  createdAt: number
  updatedAt: number
  messageCount: number
  // Enriched container info (from ListSessionsEnriched)
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
  createdAt: number
  lastUsedAt: number
}
