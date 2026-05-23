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
                type="default"
                :class="['mode-btn', { active: agentMode === 'default' }]"
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
                type="default"
                :class="['mode-btn', { active: agentMode === 'plan' }]"
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
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-faint);
}

.mode-buttons {
  flex-shrink: 0;
}

:global(:root[data-platform="macos"] .composer-meta){
  margin-bottom: 6px;
}

:global(:root[data-platform="macos"] .mode-caption){
  display: none;
}

:global(:root[data-platform="macos"] .mode-buttons){
  padding: 2px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: color-mix(in srgb, var(--platform-bg-raised) 70%, transparent);
}

:global(:root[data-platform="macos"] .mode-buttons .n-button){
  --n-height: 22px !important;
  --n-border-radius: 6px !important;
  --n-padding: 0 8px !important;
  --n-font-size: 11px !important;
  --n-border: 1px solid transparent !important;
  --n-border-hover: 1px solid transparent !important;
  --n-border-pressed: 1px solid transparent !important;
  --n-border-focus: 1px solid transparent !important;
  --n-ripple-color: transparent !important;
}

:global(:root[data-platform="macos"] .mode-buttons .mode-btn.active){
  --n-color: var(--platform-bg-raised) !important;
  --n-color-hover: var(--platform-bg-raised) !important;
  --n-color-pressed: var(--platform-bg-raised) !important;
  --n-color-focus: var(--platform-bg-raised) !important;
  --n-text-color: var(--text-primary) !important;
  --n-text-color-hover: var(--text-primary) !important;
  --n-text-color-pressed: var(--text-primary) !important;
  --n-text-color-focus: var(--text-primary) !important;
  background: var(--platform-bg-raised) !important;
  color: var(--text-primary) !important;
  box-shadow: var(--platform-shadow-1);
}

:global(:root[data-platform="macos"] .mode-buttons .mode-btn.active .n-button__state-border),
:global(:root[data-platform="macos"] .mode-buttons .mode-btn.active .n-button__border){
  border-color: transparent !important;
}

.composer-hint {
  font-size: var(--fs-2xs);
  color: var(--text-faint);
  white-space: nowrap;
}

:global(:root[data-platform="macos"] .composer-hint){
  display: none;
}

.attached-file {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  padding: 4px 8px;
  background: color-mix(in srgb, var(--platform-bg-toolbar) 76%, transparent);
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
  padding: 6px 9px;
  border: 1px solid var(--border-subtle);
  border-radius: 9px;
  background: color-mix(in srgb, var(--platform-bg-raised) 76%, transparent);
  box-shadow: none;
  position: relative;
  transition: border-color var(--transition-ui), box-shadow var(--transition-ui);
}

:global(:root[data-platform="macos"] .input-shell){
  border-radius: 9px;
  background: color-mix(in srgb, var(--platform-bg-raised) 82%, transparent);
  box-shadow: none;
}

:global(:root[data-platform="macos"] .chat-input .n-input-wrapper){
  padding-left: 2px;
}

.input-shell:focus-within {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px var(--platform-accent-soft);
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
  padding: 6px 0 !important;
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
  box-shadow: none;
  transition: opacity var(--transition-ui), transform var(--transition-ui);
}

:global(:root[data-platform="macos"] .send-btn){
  --n-color: transparent !important;
  --n-color-hover: var(--platform-bg-hover) !important;
  --n-color-pressed: var(--platform-bg-active) !important;
  --n-color-focus: var(--platform-bg-hover) !important;
  --n-text-color: var(--platform-accent) !important;
  --n-text-color-hover: var(--platform-accent-hover) !important;
  --n-text-color-pressed: var(--platform-accent-hover) !important;
  --n-text-color-focus: var(--platform-accent) !important;
  --n-border: 1px solid transparent !important;
  --n-border-hover: 1px solid transparent !important;
  --n-border-pressed: 1px solid transparent !important;
  --n-border-focus: 1px solid transparent !important;
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
