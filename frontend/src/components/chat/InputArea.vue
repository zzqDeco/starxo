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
    <div v-if="attachedFile" class="attached-file">
      <span class="attached-name">{{ attachedFileName }}</span>
      <button class="attached-remove" @click="removeAttachment">&times;</button>
    </div>

    <div class="input-shell">
      <NTooltip trigger="hover" placement="top">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="tiny"
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

      <NInput
        v-model:value="inputText"
        type="textarea"
        :placeholder="t('input.placeholder')"
        :autosize="{ minRows: 1, maxRows: 4 }"
        class="chat-input"
        @keydown="handleKeydown"
        :disabled="isStreaming"
      />

      <NTooltip trigger="hover" placement="top">
        <template #trigger>
          <NButton
            v-if="!isStreaming"
            type="primary"
            circle
            size="tiny"
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
            size="tiny"
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
  </div>
</template>

<style scoped>
.input-area {
  width: 100%;
}

.attached-file {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  padding: 4px 8px;
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
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

.input-shell {
  display: flex;
  align-items: flex-end;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
  box-shadow: var(--elev-2);
  position: relative;
  transition: border-color var(--transition-ui), box-shadow var(--transition-ui);
}

.input-shell::after {
  content: "";
  position: absolute;
  left: var(--space-md);
  right: var(--space-md);
  bottom: 0;
  height: 1px;
  background: var(--accent-cyan);
  transform: scaleX(0);
  transform-origin: left center;
  opacity: 0;
  transition: transform var(--transition-ui), opacity var(--transition-ui);
  pointer-events: none;
}

.input-shell:focus-within {
  border-color: var(--accent-cyan-dim);
}

.input-shell:focus-within::after {
  transform: scaleX(1);
  opacity: 0.7;
}

.chat-input {
  flex: 1;
}

.chat-input :deep(.n-input-wrapper) {
  background: transparent !important;
  box-shadow: none !important;
  padding-left: 0 !important;
  padding-right: 0 !important;
}

.chat-input :deep(textarea) {
  font-family: var(--font-sans) !important;
  font-size: var(--fs-sm) !important;
  line-height: var(--lh-normal) !important;
  padding: var(--space-sm) 0 !important;
  background: transparent !important;
  border: none !important;
  resize: none !important;
}

.chat-input :deep(textarea::placeholder) {
  color: var(--text-faint) !important;
}

.attach-btn {
  color: var(--text-muted) !important;
  flex-shrink: 0;
  transition: color var(--transition-ui) !important;
}

.attach-btn:hover {
  color: var(--text-primary) !important;
}

.send-btn {
  flex-shrink: 0;
  box-shadow: 0 2px 10px rgba(34, 211, 238, 0.24);
  transition: opacity var(--transition-ui), transform var(--transition-ui);
}

.send-btn:disabled {
  opacity: 0.4;
}

.stop-btn {
  flex-shrink: 0;
}
</style>
