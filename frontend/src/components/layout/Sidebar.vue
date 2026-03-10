<script lang="ts" setup>
import { NButton, NIcon, NDropdown, NInput, NEllipsis } from 'naive-ui'
import { Add, ChatbubbleEllipses, EllipsisVertical } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useSessionStore } from '@/stores/sessionStore'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()
const chatStore = useChatStore()
const connectionStore = useConnectionStore()
const sessionStore = useSessionStore()

// Renaming state
const renamingId = ref<string | null>(null)
const renameText = ref('')

function handleNewChat() {
  sessionStore.createSession().then(() => {
    chatStore.clearMessages()
  })
}

function handleSessionClick(sessionId: string) {
  if (sessionId === sessionStore.activeSessionId) return
  sessionStore.switchSession(sessionId)
}

function startRename(sessionId: string, currentTitle: string) {
  renamingId.value = sessionId
  renameText.value = currentTitle
}

function confirmRename(sessionId: string) {
  if (renameText.value.trim()) {
    sessionStore.renameSession(sessionId, renameText.value.trim())
  }
  renamingId.value = null
}

function cancelRename() {
  renamingId.value = null
}

function handleDelete(sessionId: string) {
  sessionStore.deleteSession(sessionId)
}

function getSessionDropdownOptions() {
  return [
    { label: t('sidebar.renameSession'), key: 'rename' },
    { label: t('sidebar.deleteSession'), key: 'delete' },
  ]
}

function handleSessionAction(key: string, sessionId: string, title: string) {
  if (key === 'rename') {
    startRename(sessionId, title)
  } else if (key === 'delete') {
    handleDelete(sessionId)
  }
}

function formatTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString(locale.value, { month: 'short', day: 'numeric' })
}

function containerStatusDot(status?: string) {
  switch (status) {
    case 'running': return 'dot-green'
    case 'stopped': return 'dot-yellow'
    case 'destroyed': return 'dot-red'
    default: return 'dot-grey'
  }
}
</script>

<template>
  <div class="sidebar">
    <!-- New Chat Button -->
    <div class="sidebar-top">
      <NButton
        type="primary"
        block
        class="new-chat-btn"
        @click="handleNewChat"
      >
        <template #icon>
          <NIcon><Add /></NIcon>
        </template>
        {{ t('sidebar.newChat') }}
      </NButton>
    </div>

    <!-- Sessions List -->
    <div class="sessions-list">
      <div class="section-label">{{ t('sidebar.sessions') }}</div>
      <div v-if="sessionStore.sessions.length === 0" class="empty-hint">
        {{ t('sidebar.noSessions') }}
      </div>
      <div
        v-for="sess in sessionStore.sessions"
        :key="sess.id"
        :class="['session-item', { active: sess.id === sessionStore.activeSessionId }]"
        tabindex="0"
        role="button"
        @click="handleSessionClick(sess.id)"
        @keydown.enter="handleSessionClick(sess.id)"
      >
        <!-- Active indicator bar -->
        <div v-if="sess.id === sessionStore.activeSessionId" class="active-bar"></div>

        <NIcon size="16" class="session-icon">
          <ChatbubbleEllipses />
        </NIcon>
        <div class="session-info">
          <!-- Inline rename -->
          <template v-if="renamingId === sess.id">
            <NInput
              v-model:value="renameText"
              size="tiny"
              autofocus
              @keydown.enter.prevent="confirmRename(sess.id)"
              @keydown.escape="cancelRename"
              @blur="confirmRename(sess.id)"
              class="rename-input"
            />
          </template>
          <template v-else>
            <NEllipsis class="session-title">{{ sess.title || t('sidebar.untitled') }}</NEllipsis>
            <span class="session-meta">
              {{ sess.messageCount || 0 }} {{ t('sidebar.messages') }}
              <span v-if="sess.updatedAt" class="session-time">· {{ formatTime(sess.updatedAt) }}</span>
            </span>
            <!-- Inline container status badge -->
            <span v-if="sess.activeContainerID" class="container-badge">
              <span :class="['dot', 'dot-mini', containerStatusDot(sess.containerStatus)]"></span>
              <span class="container-badge-text">{{ sess.containerName || sess.activeContainerID.substring(0, 8) }}</span>
              <span v-if="sess.containers && sess.containers.length > 1" class="container-count">+{{ sess.containers.length - 1 }}</span>
            </span>
            <span v-else class="no-container-hint">{{ t('sidebar.noContainer') }}</span>
          </template>
        </div>
        <!-- Session dropdown menu -->
        <NDropdown
          trigger="click"
          :options="getSessionDropdownOptions()"
          @select="(key: string) => handleSessionAction(key, sess.id, sess.title)"
          size="small"
        >
          <NButton
            quaternary
            circle
            size="tiny"
            class="session-menu-btn"
            @click.stop
          >
            <template #icon><NIcon size="14"><EllipsisVertical /></NIcon></template>
          </NButton>
        </NDropdown>
      </div>
    </div>

    <!-- Bottom: Connection Status -->
    <div class="sidebar-bottom">
      <div class="conn-strip">
        <div class="conn-item">
          <span :class="['dot', connectionStore.sshConnected ? 'dot-green' : 'dot-red']"></span>
          <span class="conn-label">{{ t('status.ssh') }}</span>
        </div>
      </div>

      <!-- Progress -->
      <div v-if="connectionStore.connecting && connectionStore.initStep" class="conn-progress">
        <span class="dot dot-pulse"></span>
        <span class="progress-text">{{ connectionStore.initStep }}</span>
      </div>

      <!-- Error -->
      <div v-if="connectionStore.error" class="conn-error">
        <span class="error-text">{{ connectionStore.error }}</span>
      </div>

      <!-- Connect / Disconnect -->
      <NButton
        v-if="!connectionStore.sshConnected"
        type="primary"
        size="small"
        block
        :loading="connectionStore.connecting"
        @click="connectionStore.connect()"
        class="conn-btn"
      >
        {{ t('common.connect') }}
      </NButton>
      <NButton
        v-else
        type="error"
        size="small"
        block
        ghost
        @click="connectionStore.disconnect()"
        class="conn-btn"
      >
        {{ t('common.disconnect') }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px;
}

.sidebar-top {
  flex-shrink: 0;
  margin-bottom: 16px;
}

.new-chat-btn {
  font-weight: 600;
  letter-spacing: 0.3px;
}

.sessions-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
}

.section-label {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 1px;
  padding: 0 8px;
  margin-bottom: 8px;
}

.empty-hint {
  font-size: 12px;
  color: var(--text-muted);
  padding: 12px 8px;
  font-style: italic;
  text-align: center;
  line-height: 1.5;
}

/* Session item */
.session-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 10px 10px 14px;
  border-radius: var(--radius-md);
  cursor: pointer;
  border: 1px solid transparent;
  transition: background var(--transition-fast), border-color var(--transition-fast);
  position: relative;
  outline: none;
  margin-bottom: 2px;
}

.session-item:hover {
  background: var(--bg-hover);
}

.session-item.active {
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.06) 0%, rgba(26, 29, 51, 0.8) 100%);
  border-color: var(--border-subtle);
}

.session-item:focus-visible {
  border-color: var(--accent-cyan);
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.25);
}

/* Active cyan bar */
.active-bar {
  position: absolute;
  left: 0;
  top: 8px;
  bottom: 8px;
  width: 3px;
  border-radius: 0 2px 2px 0;
  background: var(--accent-cyan);
  box-shadow: 0 0 8px rgba(34, 211, 238, 0.4);
}

.session-icon {
  color: var(--accent-cyan);
  margin-top: 2px;
  flex-shrink: 0;
}

.session-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.session-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.session-meta {
  font-size: 11px;
  color: var(--text-muted);
  display: flex;
  gap: 4px;
  align-items: center;
}

.session-time {
  color: var(--text-faint);
}

.session-menu-btn {
  opacity: 0;
  flex-shrink: 0;
  align-self: center;
  transition: opacity var(--transition-fast);
}

.session-item:hover .session-menu-btn,
.session-item:focus-within .session-menu-btn {
  opacity: 1;
}

.rename-input {
  font-size: 13px;
}

/* Container badge */
.container-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg-deepest);
  border-radius: 4px;
  margin-top: 2px;
  width: fit-content;
}

.container-badge-text {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-count {
  font-size: 9px;
  color: var(--accent-cyan);
  font-weight: 600;
}

.no-container-hint {
  font-size: 10px;
  color: var(--text-faint);
  font-style: italic;
  margin-top: 2px;
}

/* Dot system */
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-mini {
  width: 5px;
  height: 5px;
}

.dot-green {
  background: var(--accent-emerald);
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}

.dot-red {
  background: var(--accent-rose);
  box-shadow: 0 0 6px rgba(244, 63, 94, 0.3);
}

.dot-yellow {
  background: #eab308;
  box-shadow: 0 0 6px rgba(234, 179, 8, 0.4);
}

.dot-grey {
  background: #6b7280;
}

.dot-pulse {
  animation: pulse 1.5s ease-in-out infinite;
  background: var(--accent-amber);
  box-shadow: 0 0 6px rgba(245, 158, 11, 0.4);
}

/* Bottom section */
.sidebar-bottom {
  flex-shrink: 0;
  border-top: 1px solid var(--border-subtle);
  padding-top: 12px;
  margin-top: 8px;
}

.conn-strip {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 4px;
  margin-bottom: 10px;
}

.conn-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.conn-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 600;
}

.conn-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px;
  margin-bottom: 8px;
}

.progress-text {
  font-size: 11px;
  color: var(--accent-amber);
  font-style: italic;
}

.conn-error {
  padding: 6px 8px;
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.25);
  border-radius: var(--radius-md);
  margin-bottom: 8px;
}

.error-text {
  font-size: 11px;
  color: var(--accent-rose);
  word-break: break-word;
  line-height: 1.4;
}

.conn-btn {
  margin-top: 2px;
}
</style>
