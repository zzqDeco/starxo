<script lang="ts" setup>
import { ref, computed } from 'vue'
import { NInput, NButton, NIcon, NTooltip, NButtonGroup } from 'naive-ui'
import { Send, Attach, StopCircle, GitBranch, DocumentText } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  isStreaming: boolean
  agentMode: 'default' | 'plan'
  modeSwitching: boolean
}>()

const emit = defineEmits<{
  (e: 'send', content: string, filePath?: string): void
  (e: 'stop'): void
  (e: 'switch-mode', mode: 'default' | 'plan'): void
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

    <div class="composer-meta">
      <div class="mode-control" :aria-label="t('chat.modeLabel')">
        <span class="mode-caption">{{ t('chat.modeLabel') }}</span>
        <NButtonGroup size="tiny" class="mode-buttons">
          <NTooltip trigger="hover" placement="top">
            <template #trigger>
              <NButton
                :type="agentMode === 'default' ? 'primary' : 'default'"
                :disabled="isStreaming || modeSwitching"
                :loading="modeSwitching && agentMode !== 'default'"
                @click="emit('switch-mode', 'default')"
              >
                <template #icon><NIcon size="13"><GitBranch /></NIcon></template>
                {{ t('chat.modeDefault') }}
              </NButton>
            </template>
            {{ t('chat.modeDefaultHint') }}
          </NTooltip>
          <NTooltip trigger="hover" placement="top">
            <template #trigger>
              <NButton
                :type="agentMode === 'plan' ? 'primary' : 'default'"
                :disabled="isStreaming || modeSwitching"
                :loading="modeSwitching && agentMode !== 'plan'"
                @click="emit('switch-mode', 'plan')"
              >
                <template #icon><NIcon size="13"><DocumentText /></NIcon></template>
                {{ t('chat.modePlan') }}
              </NButton>
            </template>
            {{ t('chat.modePlanHint') }}
          </NTooltip>
        </NButtonGroup>
      </div>

      <span class="composer-hint">{{ t('input.shiftEnter') }}</span>
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

.composer-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  margin-bottom: var(--space-sm);
}

.mode-control {
  display: inline-flex;
  align-items: center;
  gap: var(--space-sm);
  min-width: 0;
}

.mode-caption {
  font-family: var(--font-brand);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.6px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.mode-buttons {
  flex-shrink: 0;
}

.composer-hint {
  font-size: var(--fs-2xs);
  color: var(--text-faint);
  white-space: nowrap;
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
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--bg-surface) 88%, black);
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
  box-shadow: var(--shadow-cyan);
  transition: opacity var(--transition-ui), transform var(--transition-ui);
}

.send-btn:disabled {
  opacity: 0.4;
}

.stop-btn {
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .composer-meta {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .composer-hint {
    display: none;
  }
}
</style>
