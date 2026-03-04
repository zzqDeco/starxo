<script lang="ts" setup>
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import { onMounted } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import { useSettingsStore } from '@/stores/settingsStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useChatStore } from '@/stores/chatStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useContainerStore } from '@/stores/containerStore'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type { Session } from '@/types/session'
import type { Message, TurnEvent, InterruptEvent, ModeChangedEvent } from '@/types/message'

const settingsStore = useSettingsStore()
const connectionStore = useConnectionStore()
const chatStore = useChatStore()
const sessionStore = useSessionStore()
const containerStore = useContainerStore()

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#22d3ee',
    primaryColorHover: '#67e8f9',
    primaryColorPressed: '#06b6d4',
    primaryColorSuppl: '#22d3ee',
    bodyColor: '#0c0e1a',
    cardColor: '#141726',
    modalColor: '#141726',
    popoverColor: '#1a1d33',
    tableColor: '#141726',
    inputColor: '#0f1122',
    actionColor: '#0f1122',
    tagColor: '#1a1d33',
    borderColor: '#2a2d45',
    dividerColor: '#2a2d45',
    hoverColor: '#1e2140',
    textColor1: '#f0f0f5',
    textColor2: '#c8c9d6',
    textColor3: '#8b8da3',
    placeholderColor: '#5a5c72',
    fontFamily: '"Nunito", "Segoe UI", system-ui, sans-serif'
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall: '6px',
  },
  Input: {
    borderRadius: '8px',
  },
  Card: {
    borderRadius: '12px',
  },
  Modal: {
    borderRadius: '16px',
  },
  Tag: {
    borderRadius: '6px',
  },
  Dropdown: {
    borderRadius: '8px',
    optionColorHover: '#1e2140',
    color: '#1a1d33',
  },
  Collapse: {
    titleFontSize: '13px',
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
  await sessionStore.loadSessions()

  // Restore messages for the active session
  await restoreActiveMessages()

  // Session switched event — SSH stays connected, only container state changes
  EventsOn('session:switched', async (data: { session: Session; containerID?: string }) => {
    if (data?.session) {
      sessionStore.setActiveSession(data.session as Session)
      await restoreActiveMessages()
      sessionStore.loadSessions()
      containerStore.loadContainers()
      // Update active container (SSH remains connected)
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
      containerStore.loadContainers()
      sessionStore.loadSessions()
    }
  })

  // Container activated (switched to existing container)
  EventsOn('container:activated', (data: { containerID: string }) => {
    if (data?.containerID) {
      containerStore.setActiveContainer(data.containerID)
      containerStore.loadContainers()
    }
  })

  // Container deactivated
  EventsOn('container:deactivated', () => {
    containerStore.clearActiveContainer()
  })

  // Timeline events (unified event stream — the only message source)
  EventsOn('agent:timeline', (data: TurnEvent) => {
    if (data) {
      chatStore.addTimelineEvent(data)
      if (!chatStore.agentDone && data.agent) {
        chatStore.setGenerating(true, data.agent)
      }
    }
  })

  // Agent done
  EventsOn('agent:done', () => {
    chatStore.setGenerating(false)
    sessionStore.loadSessions()
  })

  // Agent error
  EventsOn('agent:error', (data: string) => {
    chatStore.setGenerating(false)
    if (data) {
      chatStore.addMessage({
        id: crypto.randomUUID(),
        role: 'system',
        content: `Error: ${data}`,
        timestamp: Date.now(),
        events: []
      })
    }
  })

  // Interrupt event — agent needs user input
  EventsOn('agent:interrupt', (data: InterruptEvent) => {
    if (data) {
      chatStore.setInterrupt(data)
    }
  })

  // Mode changed event
  EventsOn('agent:mode_changed', (data: ModeChangedEvent) => {
    if (data?.mode) {
      chatStore.setMode(data.mode)
    }
  })
})
</script>

<template>
  <NConfigProvider :theme="darkTheme" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <MainLayout />
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
</style>
