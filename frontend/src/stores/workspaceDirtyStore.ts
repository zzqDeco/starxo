import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface WorkspaceChangedEvent {
  sessionId?: string
  containerID?: string
  path?: string
  source?: string
  action?: string
  createdAt?: number
}

export const useWorkspaceDirtyStore = defineStore('workspaceDirty', () => {
  const revision = ref(0)
  const lastEvent = ref<WorkspaceChangedEvent | null>(null)

  function markChanged(event?: WorkspaceChangedEvent) {
    revision.value += 1
    lastEvent.value = {
      ...(event || {}),
      createdAt: event?.createdAt || Date.now(),
    }
  }

  function clear() {
    revision.value += 1
    lastEvent.value = null
  }

  return {
    revision,
    lastEvent,
    markChanged,
    clear,
  }
})
