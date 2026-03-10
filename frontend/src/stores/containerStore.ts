import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ContainerInfo } from '@/types/session'
import { useSessionStore } from './sessionStore'
import { useConnectionStore } from './connectionStore'
import {
  ListContainers,
  RefreshContainerStatus,
  StartContainer,
  StopContainer,
  DestroyContainer,
  CreateContainer,
  ActivateContainer,
  DeactivateContainer,
} from '../../wailsjs/go/service/ContainerService'

export const useContainerStore = defineStore('container', () => {
  const containers = ref<ContainerInfo[]>([])
  const loading = ref(false)
  const activeContainerID = ref('')
  const creatingContainer = ref(false)
  const containerProgress = ref(0)
  const containerStep = ref('')
  const pendingActionKeys = ref(new Set<string>())

  const sessionStore = useSessionStore()

  const activeSessionContainers = computed(() =>
    containers.value.filter(c => c.sessionID === sessionStore.activeSessionId)
  )

  const otherContainers = computed(() =>
    containers.value.filter(c => c.sessionID !== sessionStore.activeSessionId)
  )

  function setActionPending(key: string, pending: boolean) {
    const next = new Set(pendingActionKeys.value)
    if (pending) next.add(key)
    else next.delete(key)
    pendingActionKeys.value = next
  }

  function isActionPending(key: string) {
    return pendingActionKeys.value.has(key)
  }

  async function loadContainers() {
    loading.value = true
    try {
      const list = await ListContainers()
      containers.value = (list || []) as ContainerInfo[]
    } catch (e) {
      console.error('Failed to load containers:', e)
      throw e
    } finally {
      loading.value = false
    }
  }

  async function refreshStatus(id: string) {
    const key = `refresh:${id}`
    setActionPending(key, true)
    try {
      const updated = await RefreshContainerStatus(id) as ContainerInfo
      if (updated) {
        const idx = containers.value.findIndex(c => c.id === id)
        if (idx >= 0) containers.value[idx] = updated
      }
    } catch (e) {
      console.error('Failed to refresh container status:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  async function startContainer(id: string) {
    const key = `start:${id}`
    setActionPending(key, true)
    try {
      await StartContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to start container:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  async function stopContainer(id: string) {
    const key = `stop:${id}`
    setActionPending(key, true)
    try {
      await StopContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to stop container:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  async function destroyContainer(id: string) {
    const key = `destroy:${id}`
    setActionPending(key, true)
    try {
      await DestroyContainer(id)
      if (activeContainerID.value === id) activeContainerID.value = ''
      await loadContainers()
    } catch (e) {
      console.error('Failed to destroy container:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  async function createContainer() {
    const connectionStore = useConnectionStore()
    if (!connectionStore.sshConnected) return

    creatingContainer.value = true
    containerProgress.value = 0
    containerStep.value = ''
    try {
      await CreateContainer()
      await loadContainers()
    } catch (e) {
      console.error('Failed to create container:', e)
      throw e
    } finally {
      creatingContainer.value = false
      containerStep.value = ''
      containerProgress.value = 0
    }
  }

  async function activateContainer(id: string) {
    const key = `activate:${id}`
    setActionPending(key, true)
    try {
      await ActivateContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to activate container:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  async function deactivateContainer() {
    const key = 'deactivate'
    setActionPending(key, true)
    try {
      await DeactivateContainer()
      activeContainerID.value = ''
      await loadContainers()
    } catch (e) {
      console.error('Failed to deactivate container:', e)
      throw e
    } finally {
      setActionPending(key, false)
    }
  }

  function setActiveContainer(id: string) {
    activeContainerID.value = id
  }

  function clearActiveContainer() {
    activeContainerID.value = ''
  }

  function updateContainerProgress(step: string, percent: number) {
    containerStep.value = step
    containerProgress.value = percent
  }

  return {
    containers,
    loading,
    activeContainerID,
    creatingContainer,
    containerProgress,
    containerStep,
    activeSessionContainers,
    otherContainers,
    pendingActionKeys,
    isActionPending,
    loadContainers,
    refreshStatus,
    startContainer,
    stopContainer,
    destroyContainer,
    createContainer,
    activateContainer,
    deactivateContainer,
    setActiveContainer,
    clearActiveContainer,
    updateContainerProgress,
  }
})
