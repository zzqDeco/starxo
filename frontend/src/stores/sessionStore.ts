import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Session } from '@/types/session'

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
  LoadSessionData,
} from '../../wailsjs/go/service/SessionService'

export const useSessionStore = defineStore('session', () => {
  const sessions = ref<Session[]>([])
  const activeSessionId = ref<string | null>(null)
  const loading = ref(false)
  const creating = ref(false)
  const switching = ref(false)
  const deletingSessionIds = ref(new Set<string>())
  const renamingSessionIds = ref(new Set<string>())

  const activeSession = computed(() =>
    sessions.value.find(s => s.id === activeSessionId.value) || null
  )

  const isBusy = computed(() => creating.value || switching.value || loading.value)

  async function loadSessions() {
    loading.value = true
    try {
      const list = await ListSessionsEnriched()
      sessions.value = (list || []) as Session[]
      const active = await GetActiveSession()
      if (active) activeSessionId.value = active.id
    } catch (e) {
      console.error('Failed to load sessions:', e)
      throw e
    } finally {
      loading.value = false
    }
  }

  async function createSession(title?: string) {
    creating.value = true
    try {
      const sess = await CreateSession(title || '')
      if (sess) {
        sessions.value.unshift(sess as Session)
        activeSessionId.value = sess.id
      }
      return sess
    } catch (e) {
      console.error('Failed to create session:', e)
      throw e
    } finally {
      creating.value = false
    }
  }

  async function switchSession(sessionId: string) {
    if (sessionId === activeSessionId.value) return
    switching.value = true
    try {
      await SwitchSession(sessionId)
      activeSessionId.value = sessionId
    } catch (e) {
      console.error('Failed to switch session:', e)
      throw e
    } finally {
      switching.value = false
    }
  }

  function setDeleting(sessionId: string, deleting: boolean) {
    const next = new Set(deletingSessionIds.value)
    if (deleting) next.add(sessionId)
    else next.delete(sessionId)
    deletingSessionIds.value = next
  }

  function setRenaming(sessionId: string, renaming: boolean) {
    const next = new Set(renamingSessionIds.value)
    if (renaming) next.add(sessionId)
    else next.delete(sessionId)
    renamingSessionIds.value = next
  }

  async function deleteSession(sessionId: string) {
    setDeleting(sessionId, true)
    try {
      await DeleteSession(sessionId)
      sessions.value = sessions.value.filter(s => s.id !== sessionId)
    } catch (e: any) {
      console.error('Failed to delete session:', e)
      throw e
    } finally {
      setDeleting(sessionId, false)
    }
  }

  async function renameSession(sessionId: string, title: string) {
    setRenaming(sessionId, true)
    try {
      await RenameSession(sessionId, title)
      const sess = sessions.value.find(s => s.id === sessionId)
      if (sess) sess.title = title
    } catch (e) {
      console.error('Failed to rename session:', e)
      throw e
    } finally {
      setRenaming(sessionId, false)
    }
  }

  function isDeleting(sessionId: string) {
    return deletingSessionIds.value.has(sessionId)
  }

  function isRenaming(sessionId: string) {
    return renamingSessionIds.value.has(sessionId)
  }

  async function loadActiveMessages() {
    try {
      return await GetActiveSessionMessages()
    } catch (e) {
      console.error('Failed to load active messages:', e)
      return null
    }
  }

  async function saveChatDisplay(messages: any[]) {
    try {
      await SaveChatDisplay(JSON.stringify(messages))
    } catch (e) {
      console.error('Failed to save chat display:', e)
    }
  }

  async function loadChatDisplay(): Promise<any[] | null> {
    try {
      const data = await LoadChatDisplay()
      if (data) return JSON.parse(data)
      return null
    } catch (e) {
      console.error('Failed to load chat display:', e)
      return null
    }
  }

  async function loadSessionData(): Promise<any | null> {
    try {
      return await LoadSessionData()
    } catch (e) {
      console.error('Failed to load session data:', e)
      return null
    }
  }

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
    creating,
    switching,
    isBusy,
    loadSessions,
    createSession,
    switchSession,
    deleteSession,
    renameSession,
    deletingSessionIds,
    renamingSessionIds,
    isDeleting,
    isRenaming,
    loadActiveMessages,
    saveChatDisplay,
    loadChatDisplay,
    loadSessionData,
    setActiveSession,
  }
})
