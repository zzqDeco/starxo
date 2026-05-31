<script lang="ts" setup>
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { NIcon } from 'naive-ui'
import { ArrowDown, Terminal, FolderOpen, ShieldCheckmark } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useAutoScroll } from '@/composables/useHelpers'
import MessageBubble from './MessageBubble.vue'
import InputArea from './InputArea.vue'
import InterruptDialog from './InterruptDialog.vue'
import AgentStatus from '@/components/status/AgentStatus.vue'
import SxStatusBadge from '@/components/ui/SxStatusBadge.vue'
import { SendMessage, SetMode, StopGeneration } from '../../../wailsjs/go/service/ChatService'
import { useI18n } from 'vue-i18n'
import { useUiFeedback } from '@/composables/useUiFeedback'

const { t } = useI18n()
const chatStore = useChatStore()
const connectionStore = useConnectionStore()
const feedback = useUiFeedback()
const modeSwitching = ref(false)

const scrollContainer = ref<HTMLElement | null>(null)
const bottomAreaRef = ref<HTMLElement | null>(null)
const scrollBtnBottom = ref(80)
const { isAutoScroll, isNearBottom, scrollToBottom, onScroll } = useAutoScroll(scrollContainer)

const hasMessages = computed(() => chatStore.visibleMessages.length > 0)
const taskStats = computed(() => chatStore.unifiedTaskStats)
const showInlineTaskStatus = computed(() => chatStore.isStreaming || taskStats.value.total > 0)
const inlineTaskTitle = computed(() => {
  if (chatStore.isStreaming) return t('chat.agentRunning')
  if (taskStats.value.total > 0) return t('chat.taskProgress', { done: taskStats.value.done, total: taskStats.value.total })
  return ''
})

// Show scroll-to-bottom button when not near bottom
const showScrollBtn = computed(() => hasMessages.value && !isNearBottom.value)
const currentMode = computed(() => chatStore.agentMode)

// Flash animation for new messages
const hasNewMessages = ref(false)
let flashTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => chatStore.messages.length,
  () => {
    if (isAutoScroll.value) {
      nextTick(() => scrollToBottom())
    } else {
      hasNewMessages.value = true
      if (flashTimer) clearTimeout(flashTimer)
      flashTimer = setTimeout(() => { hasNewMessages.value = false }, 2000)
    }
  }
)

watch(
  () => {
    const last = chatStore.messages[chatStore.messages.length - 1]
    return last?.events?.length || 0
  },
  () => {
    if (isAutoScroll.value) {
      nextTick(() => scrollToBottom(false))
    }
  }
)

function handleScrollToBottom() {
  scrollToBottom()
  hasNewMessages.value = false
}

async function handleSend(content: string) {
  chatStore.addUserMessage(content)
  chatStore.setGenerating(true)
  nextTick(() => scrollToBottom())

  try {
    await SendMessage(content)
  } catch (e) {
    console.error('Failed to send message:', e)
    chatStore.setGenerating(false)
    feedback.error(t('feedback.actions.sendMessage'), e)
  }
}

function handleHintClick(hintKey: string) {
  const text = t(hintKey).replace(/^"|"$/g, '')
  handleSend(text)
}

function handleStop() {
  try {
    StopGeneration()
  } catch (e) {
    console.error('Failed to stop generation:', e)
  }
  chatStore.setGenerating(false)
}

async function handleModeSwitch(mode: 'default' | 'plan') {
  if (mode === chatStore.agentMode || chatStore.isStreaming || modeSwitching.value) {
    return
  }
  modeSwitching.value = true
  try {
    await SetMode(mode)
    chatStore.setMode(mode)
    feedback.success(t('feedback.modeSwitched', { mode: mode === 'plan' ? t('chat.modePlan') : t('chat.modeDefault') }))
  } catch (e) {
    console.error('Failed to switch mode:', e)
    feedback.error(t('feedback.actions.switchMode'), e)
  } finally {
    modeSwitching.value = false
  }
}

let bottomObserver: ResizeObserver | null = null
onMounted(() => {
  if (bottomAreaRef.value) {
    bottomObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        scrollBtnBottom.value = entry.contentRect.height + 12
      }
    })
    bottomObserver.observe(bottomAreaRef.value)
  }
})
onUnmounted(() => {
  bottomObserver?.disconnect()
})
</script>

<template>
  <div class="chat-panel">
    <div
      ref="scrollContainer"
      class="messages-area"
      @scroll="onScroll"
    >
      <div v-if="!hasMessages" class="empty-state">
        <span :class="['empty-status-dot', connectionStore.isReady ? 'ready' : 'offline']" aria-hidden="true"></span>
        <h2 class="empty-title">{{ t('chat.workbenchTitle') }}</h2>
        <p class="empty-subtitle">
          {{ connectionStore.isReady
            ? t('chat.emptyConnected')
            : t('chat.emptyDisconnected')
          }}
        </p>
        <div class="empty-capabilities" :aria-label="t('chat.capabilitiesLabel')">
          <div class="capability-item">
            <NIcon size="16"><Terminal /></NIcon>
            <span>{{ t('chat.capabilityRuntime') }}</span>
          </div>
          <div class="capability-item">
            <NIcon size="16"><FolderOpen /></NIcon>
            <span>{{ t('chat.capabilityWorkspace') }}</span>
          </div>
          <div class="capability-item">
            <NIcon size="16"><ShieldCheckmark /></NIcon>
            <span>{{ t('chat.capabilityIsolation') }}</span>
          </div>
        </div>
        <div class="empty-hints">
          <button class="hint-card" @click="handleHintClick('chat.hint1')">{{ t('chat.hint1') }}</button>
          <button class="hint-card" @click="handleHintClick('chat.hint2')">{{ t('chat.hint2') }}</button>
          <button class="hint-card" @click="handleHintClick('chat.hint3')">{{ t('chat.hint3') }}</button>
        </div>
      </div>

      <div
        v-else
        class="messages-list"
        role="log"
        aria-live="polite"
        aria-relevant="additions"
        :aria-busy="chatStore.isStreaming"
      >
        <MessageBubble
          v-for="msg in chatStore.visibleMessages"
          :key="msg.id"
          :message="msg"
        />
      </div>

      <AgentStatus
        v-if="chatStore.isStreaming"
        :agent="chatStore.currentAgent"
      />
    </div>

    <Transition name="scroll-btn">
      <button
        v-if="showScrollBtn"
        :class="['scroll-to-bottom', { flash: hasNewMessages }]"
        :style="{ bottom: scrollBtnBottom + 'px' }"
        @click="handleScrollToBottom"
      >
        <NIcon size="18"><ArrowDown /></NIcon>
      </button>
    </Transition>

    <div ref="bottomAreaRef" class="bottom-area">
      <InterruptDialog />

      <div class="bottom-stack">
        <div v-if="showInlineTaskStatus" class="inline-task-status" aria-live="polite">
          <div class="inline-task-copy">
            <SxStatusBadge :tone="chatStore.isStreaming ? 'info' : 'success'">
              {{ chatStore.isStreaming ? t('chat.statusRunning') : t('chat.statusReady') }}
            </SxStatusBadge>
            <span>{{ inlineTaskTitle }}</span>
          </div>
          <span v-if="taskStats.currentTask" class="inline-task-current">
            {{ taskStats.currentTask.title }}
          </span>
        </div>

        <InputArea
          :is-streaming="chatStore.isStreaming"
          :agent-mode="currentMode"
          :mode-switching="modeSwitching"
          @send="handleSend"
          @stop="handleStop"
          @switch-mode="handleModeSwitch"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: transparent;
  position: relative;
  --chat-content-max-width: 820px;
  --chat-content-padding: var(--space-xl);
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 18px 0;
}

:global(:root[data-platform="macos"] .messages-area){
  padding: 14px 0;
}

.messages-list {
  max-width: var(--chat-content-max-width);
  margin: 0 auto;
  padding: 0 var(--chat-content-padding);
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

:global(:root[data-platform="macos"] .messages-list){
  gap: 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  height: 100%;
  width: min(580px, 100%);
  margin: 0 auto;
  padding: 48px 32px;
  text-align: left;
  animation: fadeIn 220ms ease both;
}

.empty-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-faint);
  margin-bottom: 14px;
}

.empty-status-dot.ready {
  background: var(--accent-emerald);
}

.empty-status-dot.offline {
  background: var(--accent-amber);
}

.empty-title {
  font-family: var(--font-brand);
  font-size: 20px;
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
  letter-spacing: 0;
  margin: 0 0 6px 0;
}

.empty-subtitle {
  font-size: var(--fs-sm);
  color: var(--text-muted);
  margin: 0 0 18px 0;
  max-width: 440px;
}

.empty-capabilities {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0;
  width: min(360px, 100%);
  margin-bottom: 18px;
  border: 1px solid var(--border-subtle);
  border-radius: 9px;
  overflow: hidden;
  background: var(--platform-bg-raised);
}

.capability-item {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  min-height: 34px;
  padding: 0 11px;
  border: 0;
  border-bottom: 1px solid var(--border-subtle);
  border-radius: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.capability-item:last-child {
  border-bottom: 0;
}

:global(:root[data-platform="macos"] .capability-item),
:global(:root[data-platform="macos"] .hint-card){
  background: color-mix(in srgb, var(--platform-bg-raised) 62%, transparent);
  border-color: color-mix(in srgb, var(--border-subtle) 72%, transparent);
  box-shadow: none;
}

.empty-hints {
  display: grid;
  gap: 6px;
  width: min(420px, 100%);
}

.hint-card {
  padding: 7px 10px;
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
  color: var(--text-secondary);
  font-size: var(--fs-xs);
  font-family: var(--font-sans);
  cursor: pointer;
  text-align: left;
  transition: border-color var(--transition-ui), color var(--transition-ui),
    background var(--transition-ui),
    box-shadow var(--transition-ui);
}

.hint-card:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
  background: var(--platform-bg-hover);
  box-shadow: none;
}

.scroll-to-bottom {
  position: absolute;
  right: 24px;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  box-shadow: var(--shadow-md);
  z-index: 50;
}

.scroll-to-bottom:hover {
  background: var(--bg-hover);
  color: var(--accent-cyan);
  border-color: var(--accent-cyan-dim);
  box-shadow: var(--platform-shadow-1);
}

.scroll-to-bottom.flash {
  animation: scrollBtnFlash 1s ease-in-out infinite;
}

@keyframes scrollBtnFlash {
  0%, 100% { border-color: var(--border-subtle); }
  50% { border-color: var(--accent-cyan); box-shadow: 0 0 0 3px var(--platform-accent-soft); }
}

.scroll-btn-enter-active,
.scroll-btn-leave-active {
  transition: all 200ms ease;
}

.scroll-btn-enter-from,
.scroll-btn-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

.bottom-area {
  flex-shrink: 0;
  border-top: 1px solid var(--border-subtle);
  background: var(--platform-bg-toolbar);
  backdrop-filter: blur(20px) saturate(1.12);
}

:global(:root[data-platform="macos"] .bottom-area){
  background: color-mix(in srgb, var(--platform-bg-toolbar) 86%, transparent);
}

:global(:root[data-platform="macos"] .bottom-stack){
  padding-top: 7px;
  padding-bottom: 10px;
}

.bottom-stack {
  max-width: var(--chat-content-max-width);
  width: 100%;
  margin: 0 auto;
  padding: 10px var(--chat-content-padding) 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.inline-task-status {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 10px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-10);
  background: color-mix(in srgb, var(--sx-toolbar-bg) 84%, transparent);
  color: var(--sx-text-secondary);
  font: 400 12px/16px var(--sx-font-sans);
}

.inline-task-copy {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.inline-task-copy > span:last-child,
.inline-task-current {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inline-task-current {
  color: var(--sx-text-tertiary);
  font-size: 11px;
  text-align: right;
}

@media (max-width: 720px) {
  .chat-panel {
    --chat-content-padding: var(--space-md);
  }

  .empty-capabilities {
    grid-template-columns: 1fr;
  }
}
</style>
