import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ConnectSSH, DisconnectSSH, GetStatus } from '../../wailsjs/go/service/SandboxService'
import { useSettingsStore } from './settingsStore'

export const useConnectionStore = defineStore('connection', () => {
  const sshConnected = ref(false)
  const initProgress = ref(0)
  const initStep = ref('')
  const connecting = ref(false)
  const error = ref('')

  const isReady = computed(() => sshConnected.value)

  const statusText = computed(() => {
    if (connecting.value) return initStep.value || 'Connecting...'
    if (sshConnected.value) return 'SSH Connected'
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
      await ConnectSSH()
    } catch (e: any) {
      error.value = e?.message || String(e)
      console.error('SSH connection failed:', e)
    } finally {
      connecting.value = false
      await refreshStatus()
    }
  }

  async function disconnect() {
    try {
      await DisconnectSSH()
      sshConnected.value = false
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
      }
    } catch (e) {
      console.warn('Failed to get status:', e)
    }
  }

  function updateProgress(step: string, percent: number) {
    initStep.value = step
    initProgress.value = percent
  }

  function setSSHConnected() {
    sshConnected.value = true
    connecting.value = false
    initStep.value = ''
    initProgress.value = 100
  }

  function setSSHDisconnected() {
    sshConnected.value = false
  }

  return {
    sshConnected,
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
    setSSHConnected,
    setSSHDisconnected,
  }
})
