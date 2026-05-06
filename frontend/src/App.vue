<script lang="ts" setup>
import { NButton, NCard, NCode, NConfigProvider, NDialogProvider, NMessageProvider, NModal, NSpace, NTag, darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import { useSettingsStore } from '@/stores/settingsStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useChatStore } from '@/stores/chatStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useContainerStore } from '@/stores/containerStore'
import { ApproveToolPermission, DenyToolPermission, GetMode } from '../wailsjs/go/service/ChatService'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type { Session } from '@/types/session'
import type { Message, TurnEvent, InterruptEvent, ModeChangedEvent, SessionRunState } from '@/types/message'

const settingsStore = useSettingsStore()
const connectionStore = useConnectionStore()
const chatStore = useChatStore()
const sessionStore = useSessionStore()
const containerStore = useContainerStore()

interface RuntimePermissionRequest {
  requestId: string
  sessionId?: string
  toolName: string
  title?: string
  description?: string
  toolClass?: string
  source?: string
  risk: string
  input?: string
  createdAt: number
}

const pendingPermission = ref<RuntimePermissionRequest | null>(null)
const permissionBusy = ref(false)
const permissionVisible = computed(() => pendingPermission.value !== null)

// Keep palette / radius values in sync with `:root` in src/style.css.
// Naive UI resolves theme values at component setup, so CSS custom properties
// cannot be used here — the source of truth stays in style.css and we mirror.
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#22d3ee',
    primaryColorHover: '#67e8f9',
    primaryColorPressed: '#0891b2',
    primaryColorSuppl: '#22d3ee',
    bodyColor: '#07111f',
    cardColor: '#0f172a',
    modalColor: '#0f172a',
    popoverColor: '#172033',
    tableColor: '#0f172a',
    inputColor: '#020617',
    actionColor: '#172033',
    tagColor: '#172033',
    borderColor: '#263348',
    dividerColor: '#263348',
    hoverColor: '#243044',
    textColor1: '#f8fafc',
    textColor2: '#d9e2ef',
    textColor3: '#94a3b8',
    placeholderColor: '#64748b',
    fontFamily: '"Nunito", "Segoe UI", system-ui, sans-serif',
    fontFamilyMono: '"JetBrains Mono", "Cascadia Code", "Fira Code", "Consolas", monospace'
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall: '6px',
    fontWeight: '500'
  },
  Input: {
    borderRadius: '8px'
  },
  Card: {
    borderRadius: '8px'
  },
  Modal: {
    borderRadius: '10px'
  },
  Tag: {
    borderRadius: '6px'
  },
  Dropdown: {
    borderRadius: '8px',
    optionColorHover: '#1e2140',
    color: '#1a1d33'
  },
  Collapse: {
    titleFontSize: '13px'
  },
  Tabs: {
    tabFontWeight: '500',
    tabFontWeightActive: '600'
  }
}

/** Check if an event belongs to the currently active session */
function isActiveSession(data: any): boolean {
  const sid = data?.sessionId
  return !sid || sid === sessionStore.activeSessionId
}

function permissionRiskType(risk?: string) {
  if (risk === 'execute' || risk === 'destructive') return 'warning'
  if (risk === 'write') return 'info'
  return 'default'
}

async function resolvePermission(decision: 'allow_once' | 'allow_session' | 'deny') {
  const request = pendingPermission.value
  if (!request || permissionBusy.value) return
  permissionBusy.value = true
  try {
    if (decision === 'deny') {
      await DenyToolPermission(request.requestId)
    } else {
      await ApproveToolPermission(request.requestId, decision)
    }
    pendingPermission.value = null
  } catch (e) {
    console.error('Failed to resolve permission request:', e)
  } finally {
    permissionBusy.value = false
  }
}

/** Restore messages from persisted data into chatStore */
async function restoreActiveMessages() {
  // Always clear first — prevents stale messages when switching to empty sessions
  chatStore.clearMessages()

  // Load unified session data from backend (includes messages + display + streaming state)
  const data = await sessionStore.loadSessionData()
  if (data?.display?.length > 0) {
    for (const turn of data.display) {
      chatStore.addMessage({
        id: turn.id || crypto.randomUUID(),
        role: turn.role as any,
        content: turn.content || '',
        agent: turn.agent,
        timestamp: turn.timestamp || Date.now(),
        events: turn.events || []
      })
    }
    // If there was an interrupted streaming state, show it as an incomplete message
    if (data.streaming?.partialContent) {
      chatStore.addMessage({
        id: crypto.randomUUID(),
        role: 'assistant',
        content: data.streaming.partialContent + '\n\n[streaming interrupted]',
        agent: data.streaming.agentName || 'coding_agent',
        timestamp: Date.now(),
        events: []
      })
    }
    // Restore todo state from restored events
    chatStore.restoreTodosFromMessages()
    return
  }

  // Fallback: try legacy display data
  const display = await sessionStore.loadChatDisplay()
  if (display && display.length > 0) {
    for (const msg of display) {
      chatStore.addMessage({
        id: msg.id || crypto.randomUUID(),
        role: msg.role,
        content: msg.content || '',
        agent: msg.agent,
        timestamp: msg.timestamp || Date.now(),
        events: msg.events || []
      })
    }
    chatStore.restoreTodosFromMessages()
    return
  }

  // Fallback to basic persisted messages (no timeline events)
  const persisted = await sessionStore.loadActiveMessages()
  if (persisted && persisted.length > 0) {
    for (const pm of persisted) {
      const msg: Message = {
        id: crypto.randomUUID(),
        role: pm.role as Message['role'],
        content: pm.content,
        agent: pm.name || undefined,
        timestamp: Date.now(),
        events: []
      }
      chatStore.addMessage(msg)
    }
  }
}

onMounted(async () => {
  settingsStore.loadSettings()

  // Load sessions (enriched with container info)
  await sessionStore.loadSessions().catch((e) => console.error('Failed to initialize sessions:', e))

  // Sync mode from backend for the active session at startup.
  try {
    const mode = await GetMode()
    if (mode === 'default' || mode === 'plan') {
      chatStore.setMode(mode)
    }
  } catch (e) {
    console.warn('Failed to get agent mode:', e)
  }

  // Restore messages for the active session
  await restoreActiveMessages()

  // Session switched event — full state restore from backend snapshot
  EventsOn('session:switched', async (data: {
    session: Session;
    containerID?: string;
    agentRunning?: boolean;
    currentAgent?: string;
    mode?: string;
    hasInterrupt?: boolean;
    interrupt?: InterruptEvent;
  }) => {
    if (data?.session) {
      sessionStore.setActiveSession(data.session as Session)

      // 1. Restore message history
      await restoreActiveMessages()

      // 2. Sync running state → input box enabled/disabled
      chatStore.setGenerating(data.agentRunning || false, data.currentAgent || '')
      chatStore.setSessionRunState({
        sessionId: data.session.id,
        running: data.agentRunning || false,
        currentAgent: data.currentAgent || '',
        mode: data.mode === 'plan' ? 'plan' : 'default',
        hasInterrupt: data.hasInterrupt || false,
      })

      // 3. Sync agent mode
      if (data.mode) {
        chatStore.setMode(data.mode as 'default' | 'plan')
      }

      // 4. Sync interrupt dialog
      if (data.hasInterrupt && data.interrupt) {
        chatStore.setInterrupt(data.interrupt)
      } else {
        chatStore.clearInterrupt()
      }

      // 5. Container & session list
      sessionStore.loadSessions().catch((e) => console.error('Failed to refresh sessions:', e))
      containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
      if (data.containerID) {
        containerStore.setActiveContainer(data.containerID)
      } else {
        containerStore.clearActiveContainer()
      }
    }
  })

  // SSH progress events
  EventsOn('ssh:progress', (data: { step: string; percent: number }) => {
    if (data) {
      connectionStore.updateProgress(data.step, data.percent)
    }
  })

  // SSH connected
  EventsOn('ssh:connected', () => {
    connectionStore.setSSHConnected()
  })

  // SSH disconnected (health check failure or manual disconnect)
  EventsOn('ssh:disconnected', () => {
    connectionStore.setSSHDisconnected()
    containerStore.clearActiveContainer()
  })

  // Container creation/activation progress
  EventsOn('container:progress', (data: { step: string; percent: number }) => {
    if (data) {
      containerStore.updateContainerProgress(data.step, data.percent)
    }
  })

  // New container ready
  EventsOn('container:ready', (data: { containerID: string }) => {
    if (data?.containerID) {
      containerStore.setActiveContainer(data.containerID)
      containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
      sessionStore.loadSessions().catch((e) => console.error('Failed to refresh sessions:', e))
    }
  })

  // Container activated (switched to existing container)
  EventsOn('container:activated', (data: { containerID: string }) => {
    if (data?.containerID) {
      containerStore.setActiveContainer(data.containerID)
      containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
    }
  })

  // Container deactivated
  EventsOn('container:deactivated', () => {
    containerStore.clearActiveContainer()
  })

  // Timeline events (unified event stream — filtered by sessionId)
  EventsOn('agent:timeline', (data: TurnEvent) => {
    if (!data || !isActiveSession(data)) return
    chatStore.addTimelineEvent(data)
    if (!chatStore.agentDone && data.agent) {
      chatStore.setGenerating(true, data.agent)
    }
  })

  // Agent done (now receives object with sessionId)
  EventsOn('agent:done', (data: any) => {
    if (!isActiveSession(data)) return
    chatStore.setGenerating(false)
    sessionStore.loadSessions().catch((e) => console.error('Failed to refresh sessions:', e))
  })

  // Agent error (now receives object with sessionId + error)
  EventsOn('agent:error', (data: any) => {
    if (!isActiveSession(data)) return
    chatStore.setGenerating(false)
    const errMsg = typeof data === 'string' ? data : data?.error
    if (errMsg) {
      chatStore.addMessage({
        id: crypto.randomUUID(),
        role: 'system',
        content: `Error: ${errMsg}`,
        timestamp: Date.now(),
        events: []
      })
    }
  })

  // Interrupt event — agent needs user input (filtered by sessionId)
  EventsOn('agent:interrupt', (data: InterruptEvent) => {
    if (!data || !isActiveSession(data)) return
    chatStore.setInterrupt(data)
  })

  // Mode changed event (filtered by sessionId)
  EventsOn('agent:mode_changed', (data: ModeChangedEvent) => {
    if (!data?.mode || !isActiveSession(data)) return
    chatStore.setMode(data.mode)
  })

  // Session-level run state stream — used by session rail and active composer.
  EventsOn('agent:run_state', (data: SessionRunState) => {
    if (!data?.sessionId) return
    chatStore.setSessionRunState(data)
    if (!isActiveSession(data)) return
    chatStore.setMode(data.mode)
    chatStore.setGenerating(data.running, data.currentAgent || '')
  })

  EventsOn('runtime:permission_request', (data: RuntimePermissionRequest) => {
    if (!data || !isActiveSession(data)) return
    pendingPermission.value = data
  })

  EventsOn('runtime:permission_canceled', (data: { requestId?: string }) => {
    if (!data?.requestId || pendingPermission.value?.requestId !== data.requestId) return
    pendingPermission.value = null
  })
})
</script>

<template>
  <NConfigProvider :theme="darkTheme" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <MainLayout />
        <NModal :show="permissionVisible" preset="card" class="permission-modal" :mask-closable="false">
          <template #header>
            <div class="permission-title">
              <span>{{ pendingPermission?.title || pendingPermission?.toolName }}</span>
              <NTag size="small" :type="permissionRiskType(pendingPermission?.risk)">
                {{ pendingPermission?.risk || 'permission' }}
              </NTag>
            </div>
          </template>
          <NCard embedded :bordered="false" class="permission-card">
            <div class="permission-meta">
              <span>{{ pendingPermission?.source || 'runtime' }}</span>
              <span>{{ pendingPermission?.toolClass || 'tool' }}</span>
              <span>{{ pendingPermission?.toolName }}</span>
            </div>
            <p v-if="pendingPermission?.description" class="permission-description">
              {{ pendingPermission.description }}
            </p>
            <NCode
              v-if="pendingPermission?.input"
              class="permission-input"
              :code="pendingPermission.input"
              language="json"
              word-wrap
            />
          </NCard>
          <template #footer>
            <NSpace justify="end">
              <NButton :disabled="permissionBusy" @click="resolvePermission('deny')">拒绝</NButton>
              <NButton :loading="permissionBusy" @click="resolvePermission('allow_once')">允许一次</NButton>
              <NButton type="primary" :loading="permissionBusy" @click="resolvePermission('allow_session')">
                本会话允许
              </NButton>
            </NSpace>
          </template>
        </NModal>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
#app {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.permission-modal {
  width: min(680px, calc(100vw - 32px));
}

.permission-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 15px;
  font-weight: 700;
}

.permission-card {
  background: #0b1220;
}

.permission-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: #94a3b8;
  font-family: "JetBrains Mono", "Cascadia Code", "Fira Code", "Consolas", monospace;
  font-size: 12px;
}

.permission-description {
  margin: 12px 0;
  color: #d9e2ef;
  line-height: 1.5;
}

.permission-input {
  max-height: 260px;
  overflow: auto;
}
</style>
