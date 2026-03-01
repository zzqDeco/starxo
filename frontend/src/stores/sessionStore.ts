import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Session } from '@/types/session'

// Wails bindings will be auto-generated — import from the generated path
import {
  ListSessionsEnriched,
  CreateSession,
  SwitchSession,
  DeleteSession,
  RenameSession,
  GetActiveSession,
  GetActiveSessionMessages,
  SaveChatDisplay,
  LoadChatDisplay,
} from '../../wailsjs/go/service/SessionService'

export const useSessionStore = defineStore('session', () => {
  const sessions = ref<Session[]>([])
  const activeSessionId = ref<string | null>(null)
  const loading = ref(false)

  const activeSession = computed(() =>
    sessions.value.find(s => s.id === activeSessionId.value) || null
  )

  async function loadSessions() {
    try {
      const list = await ListSessionsEnriched()
      sessions.value = (list || []) as Session[]

      // Also fetch active session from backend
      const active = await GetActiveSession()
      if (active) {
        activeSessionId.value = active.id
      }
    } catch (e) {
      console.error('Failed to load sessions:', e)
    }
  }

  async function createSession(title?: string) {
    try {
      const sess = await CreateSession(title || '')
      if (sess) {
        sessions.value.unshift(sess as Session)
        activeSessionId.value = sess.id
      }
      return sess
    } catch (e) {
      console.error('Failed to create session:', e)
      return null
    }
  }

  async function switchSession(sessionId: string) {
    if (sessionId === activeSessionId.value) return
    loading.value = true
    try {
      await SwitchSession(sessionId)
      activeSessionId.value = sessionId
    } catch (e) {
      console.error('Failed to switch session:', e)
    } finally {
      loading.value = false
    }
  }

  async function deleteSession(sessionId: string) {
    try {
      await DeleteSession(sessionId)
      sessions.value = sessions.value.filter(s => s.id !== sessionId)
    } catch (e: any) {
      console.error('Failed to delete session:', e)
      throw e
    }
  }

  async function renameSession(sessionId: string, title: string) {
    try {
      await RenameSession(sessionId, title)
      const sess = sessions.value.find(s => s.id === sessionId)
      if (sess) {
        sess.title = title
      }
    } catch (e) {
      console.error('Failed to rename session:', e)
    }
  }

  async function loadActiveMessages() {
    try {
      return await GetActiveSessionMessages()
    } catch (e) {
      console.error('Failed to load active messages:', e)
      return null
    }
  }

  /** Save the frontend's rich display messages (with timeline events) */
  async function saveChatDisplay(messages: any[]) {
    try {
      await SaveChatDisplay(JSON.stringify(messages))
    } catch (e) {
      console.error('Failed to save chat display:', e)
    }
  }

  /** Load the frontend's rich display messages */
  async function loadChatDisplay(): Promise<any[] | null> {
    try {
      const data = await LoadChatDisplay()
      if (data) {
        return JSON.parse(data)
      }
      return null
    } catch (e) {
      console.error('Failed to load chat display:', e)
      return null
    }
  }

  /** Update local session data (e.g., after session:switched event) */
  function setActiveSession(session: Session) {
    activeSessionId.value = session.id
    const idx = sessions.value.findIndex(s => s.id === session.id)
    if (idx >= 0) {
      sessions.value[idx] = { ...sessions.value[idx], ...session }
    } else {
      sessions.value.unshift(session)
    }
  }

  return {
    sessions,
    activeSessionId,
    activeSession,
    loading,
    loadSessions,
    createSession,
    switchSession,
    deleteSession,
    renameSession,
    loadActiveMessages,
    saveChatDisplay,
    loadChatDisplay,
    setActiveSession,
  }
})
