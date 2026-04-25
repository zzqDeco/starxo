const WORKSPACE_OPEN_PATH_EVENT = 'starxo:workspace-open-path'

let pendingPath = ''

export function openWorkspacePath(path?: string) {
  pendingPath = path || ''
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent(WORKSPACE_OPEN_PATH_EVENT, {
    detail: { path: pendingPath }
  }))
}

export function consumePendingWorkspacePath() {
  const path = pendingPath
  pendingPath = ''
  return path
}

export function onWorkspaceOpenPath(handler: (path: string) => void) {
  if (typeof window === 'undefined') return () => {}
  const listener = (event: Event) => {
    const detail = (event as CustomEvent<{ path?: string }>).detail
    handler(detail?.path || '')
  }
  window.addEventListener(WORKSPACE_OPEN_PATH_EVENT, listener)
  return () => window.removeEventListener(WORKSPACE_OPEN_PATH_EVENT, listener)
}
