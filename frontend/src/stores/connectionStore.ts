import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Connect as SandboxConnect, Disconnect as SandboxDisconnect, GetStatus } from '../../wailsjs/go/service/SandboxService'
import { useSettingsStore } from './settingsStore'

export const useConnectionStore = defineStore('connection', () => {
  const sshConnected = ref(false)
  const dockerRunning = ref(false)
  const containerID = ref('')
  const initProgress = ref(0)
  const initStep = ref('')
  const connecting = ref(false)
  const error = ref('')

  const isReady = computed(() => sshConnected.value && dockerRunning.value)

  const statusText = computed(() => {
    if (connecting.value) return initStep.value || 'Connecting...'
    if (isReady.value) return 'Ready'
    if (sshConnected.value && !dockerRunning.value) return 'SSH connected, Docker not running'
    return 'Disconnected'
  })

  async function connect() {
    connecting.value = true
    error.value = ''
    initProgress.value = 0
    initStep.value = 'Saving settings...'

    // Auto-save settings before connecting
    try {
      const settingsStore = useSettingsStore()
      await settingsStore.saveSettings()
    } catch (e: any) {
      error.value = 'Failed to save settings: ' + (e?.message || String(e))
      connecting.value = false
      return
    }

    initStep.value = 'Initializing SSH connection...'
    try {
      await SandboxConnect()
    } catch (e: any) {
      error.value = e?.message || String(e)
      console.error('Connection failed:', e)
    } finally {
      connecting.value = false
      await refreshStatus()
    }
  }

  async function disconnect() {
    try {
      await SandboxDisconnect()
      sshConnected.value = false
      dockerRunning.value = false
      containerID.value = ''
      initProgress.value = 0
      initStep.value = ''
    } catch (e: any) {
      error.value = e?.message || String(e)
      console.error('Disconnect failed:', e)
    }
  }

  async function refreshStatus() {
    try {
      const status = await GetStatus()
      if (status) {
        sshConnected.value = status.sshConnected
        dockerRunning.value = status.dockerRunning
        containerID.value = status.containerID
      }
    } catch (e) {
      console.warn('Failed to get status:', e)
    }
  }

  function updateProgress(step: string, percent: number) {
    initStep.value = step
    initProgress.value = percent
  }

  function setReady() {
    sshConnected.value = true
    dockerRunning.value = true
    connecting.value = false
    initStep.value = ''
    initProgress.value = 100
    refreshStatus()
  }

  return {
    sshConnected,
    dockerRunning,
    containerID,
    initProgress,
    initStep,
    connecting,
    error,
    isReady,
    statusText,
    connect,
    disconnect,
    refreshStatus,
    updateProgress,
    setReady,
  }
})
