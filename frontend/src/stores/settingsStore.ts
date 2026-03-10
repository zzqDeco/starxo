import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { AppSettings } from '@/types/config'
import { GetSettings, SaveSettings } from '../../wailsjs/go/service/SettingsService'

const defaultSettings: AppSettings = {
  ssh: {
    host: '127.0.0.1',
    port: 22,
    user: 'root',
    password: '',
    privateKey: ''
  },
  docker: {
    image: 'ubuntu:22.04',
    memoryLimit: 2048,
    cpuLimit: 2,
    workDir: '/workspace',
    network: true
  },
  llm: {
    type: 'openai',
    baseURL: 'https://api.openai.com/v1',
    apiKey: '',
    model: 'gpt-4',
    headers: {}
  },
  mcp: {
    servers: []
  },
  agent: {
    maxIterations: 30,
    motionLevel: 'normal'
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<AppSettings>(structuredClone(defaultSettings))
  const loaded = ref(false)
  const saving = ref(false)

  async function loadSettings() {
    try {
      const result = await GetSettings()
      if (result) {
        const merged = structuredClone(defaultSettings)
        const src = result as unknown as Partial<AppSettings>
        // Merge top-level sections, preserving defaults for missing fields
        if (src.ssh) Object.assign(merged.ssh, src.ssh)
        if (src.docker) Object.assign(merged.docker, src.docker)
        if (src.llm) Object.assign(merged.llm, src.llm)
        if (src.agent) Object.assign(merged.agent, src.agent)
        if (merged.agent.motionLevel !== 'normal' && merged.agent.motionLevel !== 'reduced') {
          merged.agent.motionLevel = 'normal'
        }
        if (src.mcp) {
          merged.mcp.servers = Array.isArray(src.mcp.servers) ? src.mcp.servers : []
        }
        settings.value = merged
      }
      loaded.value = true
    } catch (e) {
      console.warn('Failed to load settings from backend, using defaults:', e)
      loaded.value = true
    }
  }

  async function saveSettings() {
    saving.value = true
    try {
      await SaveSettings(settings.value as any)
    } catch (e) {
      console.error('Failed to save settings:', e)
      throw e
    } finally {
      saving.value = false
    }
  }

  function updateSSH(partial: Partial<AppSettings['ssh']>) {
    Object.assign(settings.value.ssh, partial)
  }

  function updateDocker(partial: Partial<AppSettings['docker']>) {
    Object.assign(settings.value.docker, partial)
  }

  function updateLLM(partial: Partial<AppSettings['llm']>) {
    Object.assign(settings.value.llm, partial)
  }

  function addMCPServer(server: AppSettings['mcp']['servers'][0]) {
    if (!settings.value.mcp.servers) {
      settings.value.mcp.servers = []
    }
    settings.value.mcp.servers.push(server)
  }

  function removeMCPServer(index: number) {
    settings.value.mcp.servers.splice(index, 1)
  }

  function resetToDefaults() {
    settings.value = structuredClone(defaultSettings)
  }

  return {
    settings,
    loaded,
    saving,
    loadSettings,
    saveSettings,
    updateSSH,
    updateDocker,
    updateLLM,
    addMCPServer,
    removeMCPServer,
    resetToDefaults
  }
})
