<script lang="ts" setup>
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { NIcon, NButton, NButtonGroup, NTooltip } from 'naive-ui'
import { ArrowDown } from '@vicons/ionicons5'
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

const { t } = useI18n()
const chatStore = useChatStore()
const connectionStore = useConnectionStore()

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
    chatStore.addMessage({
      id: crypto.randomUUID(),
      role: 'system',
      content: `Failed to send message: ${e}`,
      timestamp: Date.now(),
      events: []
    })
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
  if (mode === chatStore.agentMode || chatStore.isStreaming) {
    return
  }
  try {
    await SetMode(mode)
    chatStore.setMode(mode)
  } catch (e) {
    console.error('Failed to switch mode:', e)
    chatStore.addMessage({
      id: crypto.randomUUID(),
      role: 'system',
      content: `Failed to switch mode: ${e}`,
      timestamp: Date.now(),
      events: []
    })
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
    <div class="mode-toolbar">
      <span class="mode-label">{{ t('chat.modeLabel') }}</span>
      <NButtonGroup size="tiny">
        <NTooltip trigger="hover" placement="bottom">
          <template #trigger>
            <NButton
              :type="currentMode === 'default' ? 'primary' : 'default'"
              :disabled="chatStore.isStreaming"
              @click="handleModeSwitch('default')"
            >
              {{ t('chat.modeDefault') }}
            </NButton>
          </template>
          {{ t('chat.modeDefaultHint') }}
        </NTooltip>
        <NTooltip trigger="hover" placement="bottom">
          <template #trigger>
            <NButton
              :type="currentMode === 'plan' ? 'primary' : 'default'"
              :disabled="chatStore.isStreaming"
              @click="handleModeSwitch('plan')"
            >
              {{ t('chat.modePlan') }}
            </NButton>
          </template>
          {{ t('chat.modePlanHint') }}
        </NTooltip>
      </NButtonGroup>
    </div>

    <div
      ref="scrollContainer"
      class="messages-area"
      @scroll="onScroll"
    >
      <div v-if="!hasMessages" class="empty-state">
        <div class="empty-icon-wrap">
          <svg class="empty-icon-svg" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M24 4L28.9 18.1L44 20.3L33 30.5L35.8 44L24 37.1L12.2 44L15 30.5L4 20.3L19.1 18.1L24 4Z"
              stroke="currentColor" stroke-width="2" fill="none" opacity="0.6" />
            <circle cx="24" cy="24" r="8" stroke="currentColor" stroke-width="1.5" fill="none" opacity="0.4" />
          </svg>
        </div>
        <h2 class="empty-title">{{ t('chat.title') }}</h2>
        <p class="empty-subtitle">
          {{ connectionStore.isReady
            ? t('chat.emptyConnected')
            : t('chat.emptyDisconnected')
          }}
        </p>
        <div class="empty-hints">
          <button class="hint-card hint-card-primary" @click="handleHintClick('chat.hint1')">{{ t('chat.hint1') }}</button>
          <button class="hint-card" @click="handleHintClick('chat.hint2')">{{ t('chat.hint2') }}</button>
        </div>
      </div>

      <div v-else class="messages-list">
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
          @send="handleSend"
          @stop="handleStop"
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
  background: var(--bg-base);
  position: relative;
  --chat-content-max-width: 920px;
  --chat-content-padding: 24px;
}

.mode-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px var(--chat-content-padding) 0;
  max-width: var(--chat-content-max-width);
  width: 100%;
  margin: 0 auto;
}

.mode-label {
  font-size: 11px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  letter-spacing: 0.2px;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px 0;
}

.messages-list {
  max-width: var(--chat-content-max-width);
  margin: 0 auto;
  padding: 0 var(--chat-content-padding);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 48px 24px;
  text-align: center;
  animation: var(--motion-chat-enter);
}

.empty-icon-wrap {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
  color: var(--accent-cyan);
  filter: drop-shadow(0 0 20px rgba(34, 211, 238, 0.3));
  animation: var(--motion-empty-icon);
}

.empty-icon-svg {
  width: 100%;
  height: 100%;
}

.empty-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.empty-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0 0 32px 0;
  max-width: 400px;
}

.empty-hints {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  max-width: 600px;
}

.hint-card {
  padding: 10px 16px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: 13px;
  font-family: var(--font-sans);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.hint-card:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
  background: var(--bg-elevated);
  box-shadow: var(--shadow-level-1);
}

.hint-card-primary {
  border-color: var(--accent-cyan-dim);
  background: rgba(34, 211, 238, 0.12);
  color: var(--text-primary);
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
  box-shadow: var(--shadow-level-2);
  z-index: 50;
}

.scroll-to-bottom:hover {
  background: var(--bg-hover);
  color: var(--accent-cyan);
  border-color: var(--accent-cyan-dim);
  box-shadow: var(--shadow-level-2);
}

.scroll-to-bottom.flash {
  animation: var(--motion-scroll-flash);
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
  background: linear-gradient(180deg, rgba(15, 17, 34, 0.78) 0%, rgba(15, 17, 34, 0.96) 100%);
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
</style>
