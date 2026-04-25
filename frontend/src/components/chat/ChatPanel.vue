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
import TaskRailFloating from '@/components/layout/TaskRailFloating.vue'
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
        <div class="empty-kicker">
          <span :class="['empty-status-dot', connectionStore.isReady ? 'ready' : 'offline']"></span>
          {{ connectionStore.isReady ? t('chat.sandboxReady') : t('chat.sandboxRequired') }}
        </div>
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
        <TaskRailFloating />

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
  padding: var(--space-lg) 0;
}

.messages-list {
  max-width: var(--chat-content-max-width);
  margin: 0 auto;
  padding: 0 var(--chat-content-padding);
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 48px 24px;
  text-align: center;
  animation: fadeIn 600ms ease both;
}

.empty-kicker {
  display: inline-flex;
  align-items: center;
  gap: var(--space-sm);
  margin-bottom: var(--space-lg);
  padding: 5px 10px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-pill);
  background: rgba(2, 6, 23, 0.55);
  color: var(--text-muted);
  font-family: var(--font-brand);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.empty-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-faint);
}

.empty-status-dot.ready {
  background: var(--accent-emerald);
  box-shadow: 0 0 8px rgba(34, 197, 94, 0.42);
}

.empty-status-dot.offline {
  background: var(--accent-amber);
  box-shadow: 0 0 8px rgba(245, 158, 11, 0.35);
}

.empty-title {
  font-family: var(--font-brand);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-primary);
  letter-spacing: 0.5px;
  margin: 0 0 var(--space-sm) 0;
}

.empty-subtitle {
  font-size: var(--fs-md);
  color: var(--text-muted);
  margin: 0 0 var(--space-xl) 0;
  max-width: 520px;
}

.empty-capabilities {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-sm);
  width: min(620px, 100%);
  margin-bottom: var(--space-xl);
}

.capability-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  min-height: 40px;
  padding: 0 var(--space-md);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: rgba(15, 23, 42, 0.72);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
}

.empty-hints {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-md);
  justify-content: center;
  max-width: 620px;
}

.hint-card {
  padding: var(--space-md) var(--space-lg);
  background: rgba(15, 23, 42, 0.84);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-family: var(--font-sans);
  cursor: pointer;
  transition: border-color var(--transition-ui), color var(--transition-ui),
    background var(--transition-ui),
    box-shadow var(--transition-ui);
}

.hint-card:hover {
  border-color: var(--accent-cyan-dim);
  color: var(--text-primary);
  background: var(--bg-elevated);
  box-shadow: var(--shadow-cyan);
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
  box-shadow: 0 4px 16px rgba(34, 211, 238, 0.15);
}

.scroll-to-bottom.flash {
  animation: scrollBtnFlash 1s ease-in-out infinite;
}

@keyframes scrollBtnFlash {
  0%, 100% { border-color: var(--border-subtle); }
  50% { border-color: var(--accent-cyan); box-shadow: 0 0 12px rgba(34, 211, 238, 0.3); }
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
  background: linear-gradient(180deg, rgba(7, 17, 31, 0.76) 0%, rgba(2, 6, 23, 0.96) 100%);
  backdrop-filter: blur(14px);
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

@media (max-width: 720px) {
  .chat-panel {
    --chat-content-padding: var(--space-md);
  }

  .empty-capabilities {
    grid-template-columns: 1fr;
  }
}
</style>
