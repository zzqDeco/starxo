import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ContainerInfo } from '@/types/session'
import { useSessionStore } from './sessionStore'
import {
  ListContainers,
  RefreshContainerStatus,
  StartContainer,
  StopContainer,
  DestroyContainer,
} from '../../wailsjs/go/service/ContainerService'

export const useContainerStore = defineStore('container', () => {
  const containers = ref<ContainerInfo[]>([])
  const loading = ref(false)

  const sessionStore = useSessionStore()

  const activeSessionContainers = computed(() =>
    containers.value.filter(c => c.sessionID === sessionStore.activeSessionId)
  )

  const otherContainers = computed(() =>
    containers.value.filter(c => c.sessionID !== sessionStore.activeSessionId)
  )

  async function loadContainers() {
    loading.value = true
    try {
      const list = await ListContainers()
      containers.value = (list || []) as ContainerInfo[]
    } catch (e) {
      console.error('Failed to load containers:', e)
    } finally {
      loading.value = false
    }
  }

  async function refreshStatus(id: string) {
    try {
      const updated = await RefreshContainerStatus(id) as ContainerInfo
      if (updated) {
        const idx = containers.value.findIndex(c => c.id === id)
        if (idx >= 0) {
          containers.value[idx] = updated
        }
      }
    } catch (e) {
      console.error('Failed to refresh container status:', e)
    }
  }

  async function startContainer(id: string) {
    try {
      await StartContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to start container:', e)
    }
  }

  async function stopContainer(id: string) {
    try {
      await StopContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to stop container:', e)
    }
  }

  async function destroyContainer(id: string) {
    try {
      await DestroyContainer(id)
      await loadContainers()
    } catch (e) {
      console.error('Failed to destroy container:', e)
    }
  }

  return {
    containers,
    loading,
    activeSessionContainers,
    otherContainers,
    loadContainers,
    refreshStatus,
    startContainer,
    stopContainer,
    destroyContainer,
  }
})
