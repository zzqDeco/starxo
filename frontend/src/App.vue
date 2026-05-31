<script lang="ts" setup>
import { NButton, NCard, NCode, NConfigProvider, NDialogProvider, NMessageProvider, NModal, NSpace, NTag, darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import MainLayout from '@/components/layout/MainLayout.vue'
import { useSettingsStore } from '@/stores/settingsStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useChatStore } from '@/stores/chatStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useContainerStore } from '@/stores/containerStore'
import { ApproveToolPermission, DenyToolPermission, GetMode, ListToolPermissionRequests } from '../wailsjs/go/service/ChatService'
import { GetPlatformUIInfo } from '../wailsjs/go/service/PlatformService'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type { Session } from '@/types/session'
import type { Message, TurnEvent, InterruptEvent, ModeChangedEvent, SessionRunState } from '@/types/message'

const settingsStore = useSettingsStore()
const connectionStore = useConnectionStore()
const chatStore = useChatStore()
const sessionStore = useSessionStore()
const containerStore = useContainerStore()
const { t } = useI18n()

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

const permissionQueue = ref<RuntimePermissionRequest[]>([])
const permissionBusy = ref(false)
const permissionSwitching = ref(false)
const eventCleanups: Array<() => void> = []
const platformInfo = ref({
  platform: 'linux',
  goos: 'linux',
  appearance: 'system',
  supportsTranslucency: false,
  supportsMica: false,
})
const prefersDark = ref(false)
let colorSchemeQuery: MediaQueryList | null = null
const pendingPermission = computed(() => permissionQueue.value[0] || null)
const permissionVisible = computed(() => pendingPermission.value !== null)
const permissionQueuePosition = computed(() => {
  const request = pendingPermission.value
  if (!request) return ''
  const index = permissionQueue.value.findIndex((item) => item.requestId === request.requestId)
  if (index < 0 || permissionQueue.value.length <= 1) return ''
  return `${index + 1} / ${permissionQueue.value.length}`
})
const pendingPermissionSession = computed(() => {
  const sessionId = pendingPermission.value?.sessionId
  if (!sessionId) return null
  return sessionStore.sessions.find((session) => session.id === sessionId) || null
})
const pendingPermissionSessionId = computed(() => pendingPermission.value?.sessionId || '')
const pendingPermissionSessionTitle = computed(() => {
  if (!pendingPermissionSessionId.value) return t('permissions.globalRequest')
  return pendingPermissionSession.value?.title || t('sidebar.untitled')
})
const permissionIsInactiveSession = computed(() => {
  const sessionId = pendingPermissionSessionId.value
  return !!sessionId && sessionId !== sessionStore.activeSessionId
})

const activeNaiveTheme = computed(() => prefersDark.value ? darkTheme : null)

const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0a66c2',
    primaryColorHover: '#0958a8',
    primaryColorPressed: '#084b8e',
    primaryColorSuppl: '#0a66c2',
    bodyColor: '#f5f5f7',
    cardColor: '#ffffff',
    modalColor: 'rgba(255, 255, 255, 0.94)',
    popoverColor: 'rgba(255, 255, 255, 0.96)',
    tableColor: '#ffffff',
    inputColor: 'rgba(255, 255, 255, 0.82)',
    actionColor: 'rgba(60, 60, 67, 0.07)',
    tagColor: 'rgba(60, 60, 67, 0.07)',
    borderColor: 'rgba(60, 60, 67, 0.14)',
    dividerColor: 'rgba(60, 60, 67, 0.14)',
    hoverColor: 'rgba(60, 60, 67, 0.07)',
    textColor1: '#1d1d1f',
    textColor2: '#3a3a3c',
    textColor3: '#6e6e73',
    placeholderColor: '#8e8e93',
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif',
    fontFamilyMono: '"SF Mono", "JetBrains Mono", "Cascadia Code", ui-monospace, monospace'
  },
  Button: {
    borderRadiusMedium: '7px',
    borderRadiusSmall: '7px',
    fontWeight: '500'
  },
  Input: {
    borderRadius: '8px'
  },
  Card: {
    borderRadius: '10px'
  },
  Modal: {
    borderRadius: '14px'
  },
  Tag: {
    borderRadius: '7px'
  },
  Dropdown: {
    borderRadius: '10px',
    optionColorHover: 'rgba(0, 122, 255, 0.08)',
    color: 'rgba(255, 255, 255, 0.96)'
  },
  Collapse: {
    titleFontSize: '13px'
  },
  Tabs: {
    tabFontWeight: '500',
    tabFontWeightActive: '600'
  }
}

const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#4da3ff',
    primaryColorHover: '#76b9ff',
    primaryColorPressed: '#2487e8',
    primaryColorSuppl: '#4da3ff',
    bodyColor: '#1e1e20',
    cardColor: '#2c2c2e',
    modalColor: 'rgba(36, 36, 38, 0.94)',
    popoverColor: 'rgba(44, 44, 46, 0.96)',
    tableColor: '#2c2c2e',
    inputColor: 'rgba(58, 58, 60, 0.72)',
    actionColor: 'rgba(235, 235, 245, 0.08)',
    tagColor: 'rgba(235, 235, 245, 0.08)',
    borderColor: 'rgba(235, 235, 245, 0.14)',
    dividerColor: 'rgba(235, 235, 245, 0.12)',
    hoverColor: 'rgba(235, 235, 245, 0.08)',
    textColor1: '#f5f5f7',
    textColor2: '#d1d1d6',
    textColor3: '#98989d',
    placeholderColor: '#8e8e93',
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif',
    fontFamilyMono: '"SF Mono", "JetBrains Mono", "Cascadia Code", ui-monospace, monospace'
  },
  Button: {
    borderRadiusMedium: '7px',
    borderRadiusSmall: '7px',
    fontWeight: '500'
  },
  Input: {
    borderRadius: '8px'
  },
  Card: {
    borderRadius: '10px'
  },
  Modal: {
    borderRadius: '14px'
  },
  Tag: {
    borderRadius: '7px'
  },
  Dropdown: {
    borderRadius: '10px',
    optionColorHover: 'rgba(10, 132, 255, 0.16)',
    color: 'rgba(44, 44, 46, 0.96)'
  },
  Collapse: {
    titleFontSize: '13px'
  },
  Tabs: {
    tabFontWeight: '500',
    tabFontWeightActive: '600'
  }
}

const semanticThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: 'var(--sx-accent)',
    primaryColorHover: 'var(--sx-accent-hover)',
    primaryColorPressed: 'var(--sx-accent-hover)',
    primaryColorSuppl: 'var(--sx-accent)',
    bodyColor: 'var(--sx-window-bg)',
    cardColor: 'var(--sx-elevated-bg)',
    modalColor: 'var(--sx-elevated-bg)',
    popoverColor: 'var(--sx-elevated-bg)',
    tableColor: 'var(--sx-elevated-bg)',
    inputColor: 'var(--sx-control-bg)',
    actionColor: 'var(--sx-control-hover)',
    tagColor: 'var(--sx-control-hover)',
    borderColor: 'var(--sx-separator)',
    dividerColor: 'var(--sx-separator)',
    hoverColor: 'var(--sx-control-hover)',
    textColor1: 'var(--sx-text-primary)',
    textColor2: 'var(--sx-text-secondary)',
    textColor3: 'var(--sx-text-tertiary)',
    placeholderColor: 'var(--sx-text-faint)',
    fontFamily: 'var(--sx-font-sans)',
    fontFamilyMono: 'var(--sx-font-mono)',
  },
  Button: {
    borderRadiusMedium: 'var(--sx-radius-8)',
    borderRadiusSmall: 'var(--sx-radius-8)',
    fontWeight: '500',
  },
  Input: {
    borderRadius: 'var(--sx-radius-8)',
  },
  Card: {
    borderRadius: 'var(--sx-radius-10)',
  },
  Modal: {
    borderRadius: 'var(--sx-radius-12)',
  },
  Tag: {
    borderRadius: 'var(--sx-radius-8)',
  },
  Dropdown: {
    borderRadius: 'var(--sx-radius-10)',
    optionColorHover: 'var(--sx-control-hover)',
    color: 'var(--sx-elevated-bg)',
  },
  Collapse: {
    titleFontSize: '13px',
  },
  Tabs: {
    tabFontWeight: '500',
    tabFontWeightActive: '600',
  },
}

const themeOverrides = computed<GlobalThemeOverrides>(() => semanticThemeOverrides)

function applyPlatformAttributes() {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.dataset.platform = platformInfo.value.platform || 'linux'
  root.dataset.theme = prefersDark.value ? 'dark' : 'light'
  root.dataset.translucent = platformInfo.value.supportsTranslucency ? 'true' : 'false'
}

function updatePreferredTheme(matches: boolean) {
  prefersDark.value = matches
  applyPlatformAttributes()
}

async function initPlatformTheme() {
  try {
    const info = await GetPlatformUIInfo()
    platformInfo.value = {
      platform: info?.platform || 'linux',
      goos: info?.goos || 'linux',
      appearance: info?.appearance || 'system',
      supportsTranslucency: !!info?.supportsTranslucency,
      supportsMica: !!info?.supportsMica,
    }
  } catch (e) {
    console.warn('Failed to load platform UI info:', e)
  }

  if (typeof window !== 'undefined' && window.matchMedia) {
    colorSchemeQuery = window.matchMedia('(prefers-color-scheme: dark)')
    prefersDark.value = colorSchemeQuery.matches
    const listener = (event: MediaQueryListEvent) => updatePreferredTheme(event.matches)
    colorSchemeQuery.addEventListener?.('change', listener)
    if (!colorSchemeQuery.addEventListener) {
      colorSchemeQuery.addListener(listener)
    }
    eventCleanups.push(() => {
      colorSchemeQuery?.removeEventListener?.('change', listener)
      if (colorSchemeQuery && !colorSchemeQuery.removeEventListener) {
        colorSchemeQuery.removeListener(listener)
      }
    })
  }

  applyPlatformAttributes()
}

/** Check if an event belongs to the currently active session */
function isActiveSession(data: any): boolean {
  const sid = data?.sessionId
  return !sid || sid === sessionStore.activeSessionId
}

function onWailsEvent<T = any>(eventName: string, handler: (data: T) => void | Promise<void>) {
  eventCleanups.push(EventsOn(eventName, handler as (...data: any[]) => void))
}

function shortSessionId(sessionId?: string) {
  return sessionId ? sessionId.slice(0, 8) : ''
}

async function switchToPermissionSession() {
  const sessionId = pendingPermissionSessionId.value
  if (!sessionId || sessionId === sessionStore.activeSessionId || permissionSwitching.value) return
  permissionSwitching.value = true
  try {
    await sessionStore.switchSession(sessionId)
  } catch (e) {
    console.error('Failed to switch to permission request session:', e)
  } finally {
    permissionSwitching.value = false
  }
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
    removePermissionRequest(request.requestId)
  } catch (e) {
    console.error('Failed to resolve permission request:', e)
  } finally {
    permissionBusy.value = false
  }
}

function sortPermissionRequests(requests: RuntimePermissionRequest[]) {
  return [...requests].sort((a, b) => {
    if ((a.createdAt || 0) === (b.createdAt || 0)) {
      return a.requestId.localeCompare(b.requestId)
    }
    return (a.createdAt || 0) - (b.createdAt || 0)
  })
}

function upsertPermissionRequest(request: RuntimePermissionRequest) {
  if (!request?.requestId) return
  const next = permissionQueue.value.filter((item) => item.requestId !== request.requestId)
  next.push(request)
  permissionQueue.value = sortPermissionRequests(next)
}

function removePermissionRequest(requestId?: string) {
  if (!requestId) return
  permissionQueue.value = permissionQueue.value.filter((item) => item.requestId !== requestId)
}

async function refreshPermissionQueue() {
  try {
    const requests = await ListToolPermissionRequests('')
    permissionQueue.value = sortPermissionRequests((requests || []) as RuntimePermissionRequest[])
  } catch (e) {
    console.warn('Failed to refresh permission queue:', e)
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
  await initPlatformTheme()
  settingsStore.loadSettings()

  // Load sessions (enriched with container info)
  await sessionStore.loadSessions().catch((e) => console.error('Failed to initialize sessions:', e))
  await refreshPermissionQueue()

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
  onWailsEvent('session:switched', async (data: {
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
      await refreshPermissionQueue()
    }
  })

  // SSH progress events
  onWailsEvent('ssh:progress', (data: { step: string; percent: number }) => {
    if (data) {
      connectionStore.updateProgress(data.step, data.percent)
    }
  })

  // SSH connected
  onWailsEvent('ssh:connected', () => {
    connectionStore.setSSHConnected()
  })

  // SSH disconnected (health check failure or manual disconnect)
  onWailsEvent('ssh:disconnected', () => {
    connectionStore.setSSHDisconnected()
    containerStore.clearActiveContainer()
  })

  // Container creation/activation progress
  onWailsEvent('container:progress', (data: { step: string; percent: number }) => {
    if (data) {
      containerStore.updateContainerProgress(data.step, data.percent)
    }
  })

  // New container ready
  onWailsEvent('container:ready', (data: { containerID: string }) => {
    if (data?.containerID) {
      containerStore.setActiveContainer(data.containerID)
      containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
      sessionStore.loadSessions().catch((e) => console.error('Failed to refresh sessions:', e))
    }
  })

  // Container activated (switched to existing container)
  onWailsEvent('container:activated', (data: { containerID: string }) => {
    if (data?.containerID) {
      containerStore.setActiveContainer(data.containerID)
      containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
    }
  })

  // Container deactivated
  onWailsEvent('container:deactivated', () => {
    containerStore.clearActiveContainer()
  })

  // Container destroyed
  onWailsEvent('container:destroyed', (data: { containerID?: string }) => {
    if (!data?.containerID || containerStore.activeContainerID === data.containerID) {
      containerStore.clearActiveContainer()
    }
    containerStore.loadContainers().catch((e) => console.error('Failed to refresh containers:', e))
  })

  // Timeline events (unified event stream — filtered by sessionId)
  onWailsEvent('agent:timeline', (data: TurnEvent) => {
    if (!data || !isActiveSession(data)) return
    chatStore.addTimelineEvent(data)
    if (!chatStore.agentDone && data.agent) {
      chatStore.setGenerating(true, data.agent)
    }
  })

  // Agent done (now receives object with sessionId)
  onWailsEvent('agent:done', (data: any) => {
    if (!isActiveSession(data)) return
    chatStore.setGenerating(false)
    sessionStore.loadSessions().catch((e) => console.error('Failed to refresh sessions:', e))
  })

  // Agent error (now receives object with sessionId + error)
  onWailsEvent('agent:error', (data: any) => {
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
  onWailsEvent('agent:interrupt', (data: InterruptEvent) => {
    if (!data || !isActiveSession(data)) return
    chatStore.setInterrupt(data)
  })

  // Mode changed event (filtered by sessionId)
  onWailsEvent('agent:mode_changed', (data: ModeChangedEvent) => {
    if (!data?.mode || !isActiveSession(data)) return
    chatStore.setMode(data.mode)
  })

  // Session-level run state stream — used by session rail and active composer.
  onWailsEvent('agent:run_state', (data: SessionRunState) => {
    if (!data?.sessionId) return
    chatStore.setSessionRunState(data)
    if (!isActiveSession(data)) return
    chatStore.setMode(data.mode)
    chatStore.setGenerating(data.running, data.currentAgent || '')
  })

  onWailsEvent('runtime:permission_request', (data: RuntimePermissionRequest) => {
    upsertPermissionRequest(data)
  })

  onWailsEvent('runtime:permission_canceled', (data: { requestId?: string }) => {
    removePermissionRequest(data?.requestId)
  })

  onWailsEvent('runtime:permission_resolved', (data: { requestId?: string }) => {
    removePermissionRequest(data?.requestId)
  })
})

onUnmounted(() => {
  eventCleanups.forEach((cleanup) => cleanup())
  eventCleanups.length = 0
})

watch(() => sessionStore.activeSessionId, () => {
  void refreshPermissionQueue()
})
</script>

<template>
  <NConfigProvider :theme="activeNaiveTheme" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <MainLayout />
        <NModal :show="permissionVisible" preset="card" class="permission-modal" :mask-closable="false">
          <template #header>
            <div class="permission-title">
              <span>{{ pendingPermission?.title || pendingPermission?.toolName }}</span>
              <div class="permission-title-actions">
                <span v-if="permissionQueuePosition" class="permission-queue-position">{{ permissionQueuePosition }}</span>
                <NTag v-if="permissionIsInactiveSession" size="small" type="warning">
                  {{ t('permissions.inactiveSession') }}
                </NTag>
                <NTag size="small" :type="permissionRiskType(pendingPermission?.risk)">
                  {{ pendingPermission?.risk || 'permission' }}
                </NTag>
              </div>
            </div>
          </template>
          <NCard embedded :bordered="false" class="permission-card">
            <div class="permission-session-context">
              <div class="permission-session-copy">
                <span class="permission-session-label">{{ t('permissions.sessionContext') }}</span>
                <strong>{{ pendingPermissionSessionTitle }}</strong>
                <span
                  v-if="pendingPermissionSessionId"
                  class="permission-session-id"
                  :title="pendingPermissionSessionId"
                >
                  #{{ shortSessionId(pendingPermissionSessionId) }}
                </span>
              </div>
              <NButton
                v-if="permissionIsInactiveSession"
                size="tiny"
                tertiary
                :disabled="permissionBusy || sessionStore.switching"
                :loading="permissionSwitching"
                @click="switchToPermissionSession"
              >
                {{ t('permissions.switchToSession') }}
              </NButton>
            </div>
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
              <NButton :disabled="permissionBusy" @click="resolvePermission('deny')">{{ t('permissions.deny') }}</NButton>
              <NButton :loading="permissionBusy" @click="resolvePermission('allow_once')">{{ t('permissions.allowOnce') }}</NButton>
              <NButton type="primary" :loading="permissionBusy" @click="resolvePermission('allow_session')">
                {{ t('permissions.allowSession') }}
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
  font-weight: 600;
}

.permission-title-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.permission-queue-position {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 500;
}

.permission-card {
  background: color-mix(in srgb, var(--platform-bg-toolbar) 76%, transparent);
}

.permission-session-context {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.permission-session-copy {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.permission-session-copy strong {
  color: var(--text-primary);
  font-size: 13px;
}

.permission-session-label,
.permission-session-id {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}

.permission-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 12px;
}

.permission-description {
  margin: 12px 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.permission-input {
  max-height: 260px;
  overflow: auto;
}
</style>
