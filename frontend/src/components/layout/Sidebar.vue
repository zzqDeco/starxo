<script lang="ts" setup>
import { NButton, NIcon, NDropdown, NInput, NEllipsis } from 'naive-ui'
import { Add, ChatbubbleEllipses, EllipsisVertical } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useSessionStore } from '@/stores/sessionStore'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { toggleWindowZoom } from '@/composables/useNativeWindow'

const { t } = useI18n()
const chatStore = useChatStore()
const connectionStore = useConnectionStore()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()
const props = defineProps<{ compact?: boolean }>()
const newChatDisabled = computed(() => sessionStore.isBusy)

// Renaming state
const renamingId = ref<string | null>(null)
const renameText = ref('')

async function handleNewChat() {
  if (sessionStore.isBusy) return
  try {
    await sessionStore.createSession()
    chatStore.clearMessages()
    feedback.success(t('feedback.sessionCreated'))
  } catch (e) {
    feedback.error(t('feedback.actions.createSession'), e)
  }
}

async function handleSessionClick(sessionId: string) {
  if (sessionStore.switching || sessionId === sessionStore.activeSessionId) return
  try {
    await sessionStore.switchSession(sessionId)
    feedback.info(t('feedback.sessionSwitched'))
  } catch (e) {
    feedback.error(t('feedback.actions.switchSession'), e)
  }
}

function startRename(sessionId: string, currentTitle: string) {
  renamingId.value = sessionId
  renameText.value = currentTitle
}

async function confirmRename(sessionId: string) {
  const title = renameText.value.trim()
  if (!title || sessionStore.isRenaming(sessionId)) {
    renamingId.value = null
    return
  }

  try {
    await sessionStore.renameSession(sessionId, title)
    feedback.success(t('feedback.sessionRenamed'))
  } catch (e) {
    feedback.error(t('feedback.actions.renameSession'), e)
  } finally {
    renamingId.value = null
  }
}

function cancelRename() {
  renamingId.value = null
}

async function handleDelete(sessionId: string) {
  if (sessionStore.isDeleting(sessionId)) return
  const confirmed = await feedback.confirmDanger(t('sidebar.deleteSessionConfirm'))
  if (!confirmed) return

  try {
    await sessionStore.deleteSession(sessionId)
    feedback.success(t('feedback.sessionDeleted'))
  } catch (e) {
    feedback.error(t('feedback.actions.deleteSession'), e)
  }
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
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function containerStatusDot(status?: string) {
  switch (status) {
    case 'running': return 'dot-green'
    case 'stopped': return 'dot-yellow'
    case 'destroyed': return 'dot-red'
    default: return 'dot-grey'
  }
}

function runStateFor(sessionId: string) {
  return chatStore.getSessionRunState(sessionId)
}

function runStateLabel(sessionId: string) {
  const state = runStateFor(sessionId)
  if (!state) return ''
  if (state.hasInterrupt) return t('sidebar.waitingInput')
  if (state.running) return state.currentAgent || t('sidebar.agentRunning')
  return state.mode === 'plan' ? t('sidebar.modePlanShort') : t('sidebar.modeDefaultShort')
}

function runStateClass(sessionId: string) {
  const state = runStateFor(sessionId)
  if (!state) return 'idle'
  if (state.hasInterrupt) return 'waiting'
  if (state.running) return 'running'
  return state.mode === 'plan' ? 'plan' : 'idle'
}

function handleTitlebarDoubleClick() {
  void toggleWindowZoom()
}
</script>

<template>
  <nav class="sidebar" :class="{ compact: props.compact }" :aria-label="t('sidebar.sessions')">
    <div
      class="sidebar-window-hit-area"
      aria-hidden="true"
      @dblclick="handleTitlebarDoubleClick"
    ></div>

    <!-- New Chat Button -->
    <div class="sidebar-top">
      <NButton
        type="primary"
        block
        class="new-chat-btn"
        :disabled="newChatDisabled"
        :loading="sessionStore.creating"
        @click="handleNewChat"
      >
        <template #icon>
          <NIcon><Add /></NIcon>
        </template>
        <span class="new-chat-label">{{ t('sidebar.newChat') }}</span>
        <kbd class="new-chat-kbd" aria-hidden="true">⌘N</kbd>
      </NButton>
    </div>

    <!-- Sessions List -->
    <div class="sessions-list">
      <div class="section-label">{{ t('sidebar.sessions') }}</div>
      <template v-if="sessionStore.loading && sessionStore.sessions.length === 0">
        <div v-for="n in 3" :key="n" class="session-skeleton" aria-hidden="true">
          <div class="skeleton skeleton-icon"></div>
          <div class="skeleton-lines">
            <div class="skeleton skeleton-line-title"></div>
            <div class="skeleton skeleton-line-meta"></div>
          </div>
        </div>
      </template>
      <div
        v-else-if="sessionStore.sessions.length === 0"
        class="empty-hint"
      >
        {{ t('sidebar.noSessions') }}
      </div>
      <div
        v-for="sess in sessionStore.sessions"
        :key="sess.id"
        :class="['session-item', { active: sess.id === sessionStore.activeSessionId, disabled: sessionStore.switching }]"
        :tabindex="sessionStore.switching ? -1 : 0"
        :title="sess.title || t('sidebar.untitled')"
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
            <span
              v-if="runStateFor(sess.id)"
              :class="['session-run-badge', runStateClass(sess.id)]"
            >
              <span class="run-pulse" aria-hidden="true"></span>
              {{ runStateLabel(sess.id) }}
            </span>
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
            :disabled="sessionStore.switching || sessionStore.isDeleting(sess.id) || sessionStore.isRenaming(sess.id)"
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
          <span class="conn-label">SSH</span>
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
        :aria-label="t('common.connect')"
        :loading="connectionStore.connecting"
        @click="connectionStore.connect()"
        class="conn-btn"
      >
        <span class="conn-btn-dot" aria-hidden="true"></span>
        <span class="conn-btn-label">{{ t('common.connect') }}</span>
      </NButton>
      <NButton
        v-else
        type="error"
        size="small"
        block
        ghost
        :aria-label="t('common.disconnect')"
        @click="connectionStore.disconnect()"
        class="conn-btn"
      >
        <span class="conn-btn-dot connected" aria-hidden="true"></span>
        <span class="conn-btn-label">{{ t('common.disconnect') }}</span>
      </NButton>
    </div>
  </nav>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 8px 10px;
  background: transparent;
  position: relative;
}

.sidebar-window-hit-area {
  display: none;
}

.sidebar-top {
  flex-shrink: 0;
  margin-bottom: 12px;
}

.new-chat-btn {
  --n-color: transparent !important;
  --n-color-hover: var(--platform-bg-hover) !important;
  --n-color-pressed: var(--platform-bg-active) !important;
  --n-color-focus: var(--platform-bg-hover) !important;
  --n-border: 1px solid var(--border-subtle) !important;
  --n-border-hover: 1px solid var(--border-strong) !important;
  --n-border-pressed: 1px solid var(--border-strong) !important;
  --n-border-focus: 1px solid var(--border-strong) !important;
  --n-text-color: var(--text-primary) !important;
  --n-text-color-hover: var(--text-primary) !important;
  --n-text-color-pressed: var(--text-primary) !important;
  --n-text-color-focus: var(--text-primary) !important;
  --n-ripple-color: transparent !important;
  font-weight: var(--fw-medium);
  letter-spacing: 0;
  border-radius: var(--radius-md) !important;
  box-shadow: none;
}

.new-chat-btn :deep(.n-button__content) {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  justify-content: center;
}

.new-chat-label {
  flex: 1;
  text-align: left;
  padding-left: 2px;
}

.new-chat-kbd {
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  padding: 2px 6px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--platform-bg-toolbar) 70%, transparent);
  color: var(--text-muted);
  letter-spacing: 0;
  line-height: 1;
  flex-shrink: 0;
}

.sessions-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
}

.section-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-faint);
  text-transform: none;
  letter-spacing: 0;
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

/* Session skeleton (loading) */
.session-skeleton {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  min-height: 56px;
}

.skeleton-icon {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  flex-shrink: 0;
}

.skeleton-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.skeleton-line-title {
  height: 10px;
  width: 70%;
  border-radius: 3px;
}

.skeleton-line-meta {
  height: 8px;
  width: 40%;
  border-radius: 3px;
}

/* Session item */
.session-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 7px 8px;
  min-height: 46px;
  border-radius: 7px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background var(--transition-ui), border-color var(--transition-ui), box-shadow var(--transition-ui);
  position: relative;
  outline: none;
  margin-bottom: var(--space-2xs);
}

.session-item:hover {
  background: var(--platform-bg-hover);
}

.session-item.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.session-item.active {
  background: var(--platform-bg-active);
  border-color: transparent;
  box-shadow: none;
}

:global(:root[data-platform="macos"] .session-item){
  min-height: 46px;
  border-radius: 8px;
  border-color: transparent;
}

:global(:root[data-platform="macos"] .session-item.active){
  background: var(--platform-bg-active);
  box-shadow: none;
}

:global(:root[data-platform="macos"] .session-title){
  font-weight: 500;
}

.session-item:focus-visible {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 2px var(--platform-accent-soft);
}

/* Active cyan bar */
.active-bar {
  display: none;
}

.session-icon {
  color: var(--text-muted);
  margin-top: 2px;
  flex-shrink: 0;
}

.session-item.active .session-icon {
  color: var(--text-primary);
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
  font-weight: 500;
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
  font-family: var(--font-sans);
  color: var(--text-faint);
  padding: 0;
  background: transparent;
  border-radius: 0;
  margin-top: 2px;
  width: fit-content;
}

:global(:root[data-platform="macos"] .new-chat-btn){
  height: 32px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

:global(:root[data-platform="macos"] .new-chat-kbd){
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-faint);
}

:global(:root[data-platform="macos"] .sidebar){
  padding: 44px 8px 10px;
}

:global(:root[data-platform="macos"] .sidebar-window-hit-area){
  display: block;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 42px;
  z-index: 2;
  --wails-draggable: no-drag;
}

:global(:root[data-platform="macos"] .section-label){
  padding: 0 6px;
  margin-bottom: 6px;
  font-size: 11px;
  font-weight: var(--fw-medium);
  color: var(--text-faint);
}

:global(:root[data-platform="macos"] .session-item){
  padding: 7px 8px;
  gap: 8px;
}

:global(:root[data-platform="macos"] .session-meta),
:global(:root[data-platform="macos"] .no-container-hint){
  font-size: 10.5px;
}

.container-badge-text {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-count {
  font-size: 9px;
  color: var(--text-faint);
  font-weight: 600;
}

.no-container-hint {
  font-size: 10px;
  color: var(--text-faint);
  font-style: italic;
  margin-top: 2px;
}

.session-run-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  width: fit-content;
  max-width: 100%;
  margin-top: 3px;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--text-faint);
  font-size: 10px;
  font-family: var(--font-sans);
  line-height: 1.2;
}

.session-run-badge.running {
  color: color-mix(in srgb, var(--platform-accent) 64%, var(--text-muted));
  border-color: color-mix(in srgb, var(--platform-accent) 16%, transparent);
}

.session-run-badge.waiting {
  color: var(--accent-amber);
  border-color: color-mix(in srgb, var(--accent-amber) 28%, transparent);
}

.session-run-badge.plan {
  color: var(--text-muted);
  border-color: var(--border-subtle);
}

.run-pulse {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.75;
}

.session-run-badge.running .run-pulse,
.session-run-badge.waiting .run-pulse {
  animation: pulse 1.5s ease-in-out infinite;
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
}

.dot-red {
  background: var(--accent-rose);
}

.dot-yellow {
  background: var(--accent-amber);
}

.dot-grey {
  background: var(--text-muted);
}

.dot-pulse {
  animation: pulse 1.5s ease-in-out infinite;
  background: var(--accent-amber);
}

/* Bottom section */
.sidebar-bottom {
  flex-shrink: 0;
  border-top: 1px solid var(--border-subtle);
  padding-top: 10px;
  margin-top: 8px;
  background: transparent;
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
  min-height: 20px;
}

.progress-text {
  font-size: var(--fs-2xs);
  color: var(--accent-amber);
  font-style: italic;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conn-error {
  padding: 6px 8px;
  background: color-mix(in srgb, var(--accent-rose) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-rose) 24%, transparent);
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
  --n-color: transparent !important;
  --n-color-hover: var(--platform-bg-hover) !important;
  --n-color-pressed: var(--platform-bg-active) !important;
  --n-color-focus: var(--platform-bg-hover) !important;
  --n-border: 1px solid var(--border-subtle) !important;
  --n-border-hover: 1px solid var(--border-strong) !important;
  --n-border-pressed: 1px solid var(--border-strong) !important;
  --n-border-focus: 1px solid var(--border-strong) !important;
  --n-text-color: var(--text-primary) !important;
  --n-text-color-hover: var(--text-primary) !important;
  --n-text-color-pressed: var(--text-primary) !important;
  --n-text-color-focus: var(--text-primary) !important;
  --n-ripple-color: transparent !important;
  margin-top: 2px;
}

.conn-btn-dot {
  display: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-rose);
}

.conn-btn-dot.connected {
  background: var(--accent-emerald);
}

.sidebar.compact {
  align-items: center;
  padding: 10px 6px;
}

:global(:root[data-platform="macos"] .sidebar.compact){
  padding-top: 44px;
}

.sidebar.compact .sidebar-top {
  width: 100%;
  margin-bottom: 10px;
}

.sidebar.compact .new-chat-btn {
  width: 44px;
  min-width: 44px;
  height: 34px;
  padding: 0 !important;
  margin: 0 auto;
}

.sidebar.compact .new-chat-btn :deep(.n-button__content) {
  justify-content: center;
  gap: 0;
}

.sidebar.compact .new-chat-label,
.sidebar.compact .new-chat-kbd,
.sidebar.compact .section-label,
.sidebar.compact .empty-hint,
.sidebar.compact .session-info,
.sidebar.compact .conn-label,
.sidebar.compact .conn-progress,
.sidebar.compact .conn-error,
.sidebar.compact .conn-btn-label {
  display: none;
}

.sidebar.compact .sessions-list {
  width: 100%;
}

.sidebar.compact .session-item {
  width: 44px;
  height: 44px;
  min-height: 44px;
  padding: 0;
  margin: 0 auto 6px;
  align-items: center;
  justify-content: center;
  gap: 0;
}

.sidebar.compact .session-icon {
  margin: 0;
  font-size: 18px;
}

.sidebar.compact .session-menu-btn {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 18px;
  height: 18px;
  min-width: 18px;
  opacity: 0;
  background: var(--platform-bg-raised);
  border: 1px solid var(--border-subtle);
  box-shadow: var(--elev-1);
}

.sidebar.compact .session-item:hover .session-menu-btn,
.sidebar.compact .session-item:focus-within .session-menu-btn {
  opacity: 1;
}

.sidebar.compact .session-item.active::after {
  content: '';
  position: absolute;
  right: 5px;
  top: 5px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-muted);
  box-shadow: 0 0 0 2px var(--platform-bg-sidebar);
}

.sidebar.compact .sidebar-bottom {
  width: 100%;
  display: flex;
  justify-content: center;
  padding-top: 10px;
  margin-top: 10px;
}

.sidebar.compact .conn-strip {
  display: none;
}

.sidebar.compact .conn-btn {
  display: inline-flex;
  width: 44px;
  min-width: 44px;
  height: 32px;
  margin: 0 auto;
  padding: 0 !important;
  justify-content: center;
  border-radius: var(--radius-md) !important;
}

.sidebar.compact .conn-btn :deep(.n-button__content) {
  justify-content: center;
}

.sidebar.compact .conn-btn-dot {
  display: block;
}
</style>
