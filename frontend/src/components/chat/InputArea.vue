<script lang="ts" setup>
import { ref, computed } from 'vue'
import { NInput, NButton, NIcon, NTooltip } from 'naive-ui'
import { Send, Attach, StopCircle } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  isStreaming: boolean
}>()

const emit = defineEmits<{
  (e: 'send', content: string, filePath?: string): void
  (e: 'stop'): void
}>()

const inputText = ref('')
const attachedFile = ref('')

const canSend = computed(() => inputText.value.trim().length > 0 && !props.isStreaming)

function handleSend() {
  const text = inputText.value.trim()
  if (!text || props.isStreaming) return
  emit('send', text, attachedFile.value || undefined)
  inputText.value = ''
  attachedFile.value = ''
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

async function handleAttach() {
  try {
    // @ts-ignore - Wails runtime dialog
    const result = await window.runtime?.OpenFileDialog({
      title: t('input.selectFile'),
    })
    if (result) {
      attachedFile.value = result
    }
  } catch (e) {
    console.warn('File dialog not available:', e)
  }
}

function removeAttachment() {
  attachedFile.value = ''
}

const attachedFileName = computed(() => {
  if (!attachedFile.value) return ''
  const parts = attachedFile.value.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1]
})
</script>

<template>
  <div class="input-area">
    <!-- Attached file indicator -->
    <div v-if="attachedFile" class="attached-file">
      <span class="attached-name">{{ attachedFileName }}</span>
      <button class="attached-remove" @click="removeAttachment">&times;</button>
    </div>

    <div class="input-row">
      <!-- Attach button -->
      <NTooltip trigger="hover" placement="top">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="attach-btn"
            @click="handleAttach"
            :disabled="isStreaming"
          >
            <template #icon>
              <NIcon><Attach /></NIcon>
            </template>
          </NButton>
        </template>
        {{ t('input.attachFile') }}
      </NTooltip>

      <!-- Text input -->
      <NInput
        v-model:value="inputText"
        type="textarea"
        :placeholder="t('input.placeholder')"
        :autosize="{ minRows: 1, maxRows: 6 }"
        class="chat-input"
        @keydown="handleKeydown"
        :disabled="isStreaming"
      />

      <!-- Send / Stop button -->
      <NTooltip trigger="hover" placement="top">
        <template #trigger>
          <NButton
            v-if="!isStreaming"
            type="primary"
            circle
            size="small"
            class="send-btn"
            :disabled="!canSend"
            @click="handleSend"
          >
            <template #icon>
              <NIcon><Send /></NIcon>
            </template>
          </NButton>
          <NButton
            v-else
            type="error"
            circle
            size="small"
            class="stop-btn"
            @click="emit('stop')"
          >
            <template #icon>
              <NIcon><StopCircle /></NIcon>
            </template>
          </NButton>
        </template>
        {{ isStreaming ? t('input.stopGeneration') : t('input.sendMessage') }}
      </NTooltip>
    </div>

    <div class="input-hint">
      <span>{{ t('input.shiftEnter') }}</span>
    </div>
  </div>
</template>

<style scoped>
.input-area {
  flex-shrink: 0;
  padding: 12px 24px 16px;
  background: var(--bg-base);
  border-top: 1px solid var(--border-subtle);
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

.attached-file {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  padding: 4px 10px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  width: fit-content;
}

.attached-name {
  font-size: 12px;
  color: var(--accent-cyan);
  font-family: var(--font-mono);
}

.attached-remove {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 0 2px;
}

.attached-remove:hover {
  color: var(--accent-rose);
}

.input-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.chat-input {
  flex: 1;
}

.chat-input :deep(textarea) {
  font-family: var(--font-sans) !important;
  font-size: 13.5px !important;
  line-height: 1.5 !important;
  padding: 10px 14px !important;
  background: var(--bg-surface) !important;
  border-color: var(--border-subtle) !important;
  border-radius: var(--radius-lg) !important;
  resize: none !important;
  transition: border-color var(--transition-fast) !important;
}

.chat-input :deep(textarea:focus) {
  border-color: var(--accent-cyan-dim) !important;
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.1) !important;
}

.attach-btn {
  color: var(--text-muted) !important;
  flex-shrink: 0;
}

.attach-btn:hover {
  color: var(--text-primary) !important;
}

.send-btn {
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(34, 211, 238, 0.3);
}

.stop-btn {
  flex-shrink: 0;
}

.input-hint {
  display: flex;
  justify-content: flex-end;
  margin-top: 6px;
}

.input-hint span {
  font-size: 10px;
  color: var(--text-faint);
}
</style>
