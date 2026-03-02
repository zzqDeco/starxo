<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import { NButton, NInput, NCard } from 'naive-ui'
import { useChatStore } from '@/stores/chatStore'
import { ResumeWithAnswer, ResumeWithChoice, StopGeneration } from '../../../wailsjs/go/service/ChatService'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const chatStore = useChatStore()

const answers = ref<string[]>([])
const selectedIndex = ref(-1)
const isSubmitting = ref(false)

const interrupt = computed(() => chatStore.pendingInterrupt)
const isFollowUp = computed(() => interrupt.value?.type === 'followup')
const isChoice = computed(() => interrupt.value?.type === 'choice')

// Initialize per-question answer slots when interrupt changes
watch(interrupt, (val) => {
  if (val?.type === 'followup' && val.questions) {
    answers.value = new Array(val.questions.length).fill('')
  } else {
    answers.value = []
  }
}, { immediate: true })

const canSubmit = computed(() => answers.value.some(a => a.trim()))

async function submitAnswer() {
  if (!canSubmit.value || isSubmitting.value) return
  isSubmitting.value = true
  try {
    const questions = interrupt.value!.questions!
    let combined: string
    if (questions.length === 1) {
      // Single question: send answer directly
      combined = answers.value[0].trim()
    } else {
      // Multiple questions: structured format
      combined = questions
        .map((q, i) => `${i + 1}. ${q}\nAnswer: ${answers.value[i]?.trim() || '(no answer)'}`)
        .join('\n\n')
    }
    chatStore.clearInterrupt()
    chatStore.setGenerating(true)
    await ResumeWithAnswer(combined)
    answers.value = []
  } catch (e) {
    console.error('Failed to resume with answer:', e)
    chatStore.setGenerating(false)
  } finally {
    isSubmitting.value = false
  }
}

async function submitChoice(index: number) {
  if (isSubmitting.value) return
  isSubmitting.value = true
  selectedIndex.value = index
  try {
    chatStore.clearInterrupt()
    chatStore.setGenerating(true)
    await ResumeWithChoice(index)
  } catch (e) {
    console.error('Failed to resume with choice:', e)
    chatStore.setGenerating(false)
  } finally {
    isSubmitting.value = false
    selectedIndex.value = -1
  }
}

async function handleCancel() {
  chatStore.clearInterrupt()
  try {
    await StopGeneration()
  } catch (e) {
    console.error('Failed to stop:', e)
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    submitAnswer()
  }
}
</script>

<template>
  <div v-if="interrupt" class="interrupt-backdrop" @click.self="handleCancel">
    <NCard class="interrupt-card" :bordered="false">
      <!-- Follow-up Questions -->
      <template v-if="isFollowUp">
        <div class="interrupt-header">
          <span class="interrupt-icon">?</span>
          <h3 class="interrupt-title">{{ t('interrupt.agentNeedsInfo') }}</h3>
        </div>

        <div class="questions-list">
          <div
            v-for="(q, i) in interrupt!.questions"
            :key="i"
            class="question-item"
          >
            <span class="question-number">{{ i + 1 }}</span>
            <div class="question-content">
              <span class="question-text">{{ q }}</span>
              <NInput
                v-model:value="answers[i]"
                type="textarea"
                :placeholder="t('interrupt.answerPlaceholder')"
                :autosize="{ minRows: 1, maxRows: 4 }"
                :disabled="isSubmitting"
                @keydown="handleKeydown"
              />
            </div>
          </div>
        </div>

        <div class="interrupt-actions">
          <NButton
            quaternary
            size="small"
            @click="handleCancel"
            :disabled="isSubmitting"
          >
            {{ t('common.cancel') }}
          </NButton>
          <NButton
            type="primary"
            size="small"
            @click="submitAnswer"
            :disabled="!canSubmit || isSubmitting"
            :loading="isSubmitting"
          >
            {{ t('interrupt.submitAnswer') }}
          </NButton>
        </div>
      </template>

      <!-- Choice Selection -->
      <template v-if="isChoice">
        <div class="interrupt-header">
          <span class="interrupt-icon">&#x25C6;</span>
          <h3 class="interrupt-title">{{ interrupt!.question || t('interrupt.chooseOption') }}</h3>
        </div>

        <div class="options-list">
          <button
            v-for="(opt, i) in interrupt!.options"
            :key="i"
            class="option-card"
            :class="{ selected: selectedIndex === i, submitting: isSubmitting }"
            @click="submitChoice(i)"
            :disabled="isSubmitting"
          >
            <span class="option-index">{{ i + 1 }}</span>
            <div class="option-content">
              <span class="option-label">{{ opt.label }}</span>
              <span class="option-desc">{{ opt.description }}</span>
            </div>
          </button>
        </div>

        <div class="interrupt-actions">
          <NButton
            quaternary
            size="small"
            @click="handleCancel"
            :disabled="isSubmitting"
          >
            {{ t('common.cancel') }}
          </NButton>
        </div>
      </template>
    </NCard>
  </div>
</template>

<style scoped>
.interrupt-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 0 24px 100px;
  z-index: 100;
  animation: fadeInBackdrop 200ms ease;
}

@keyframes fadeInBackdrop {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.interrupt-card {
  max-width: 600px;
  width: 100%;
  background: var(--bg-elevated) !important;
  border: 1px solid var(--accent-cyan-dim) !important;
  border-radius: var(--radius-lg) !important;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4), 0 0 20px rgba(34, 211, 238, 0.1);
}

.interrupt-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.interrupt-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  border-radius: 50%;
  color: var(--accent-cyan);
  font-size: 14px;
  font-weight: 700;
  flex-shrink: 0;
}

.interrupt-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

/* Questions */
.questions-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.question-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px 12px;
  background: var(--bg-surface);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}

.question-number {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.1);
  border-radius: 50%;
  color: var(--accent-cyan);
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
  margin-top: 1px;
}

.question-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.question-text {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.question-content :deep(textarea) {
  font-size: 13px !important;
  background: var(--bg-deepest) !important;
  border-color: var(--border-subtle) !important;
  border-radius: var(--radius-md) !important;
}

.question-content :deep(textarea:focus) {
  border-color: var(--accent-cyan-dim) !important;
}

/* Options */
.options-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.option-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 200ms ease;
  text-align: left;
  color: inherit;
  font-family: inherit;
}

.option-card:hover:not(:disabled) {
  border-color: var(--accent-cyan-dim);
  background: var(--bg-elevated);
  transform: translateX(4px);
}

.option-card.selected {
  border-color: var(--accent-cyan);
  background: rgba(34, 211, 238, 0.08);
}

.option-card:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.option-index {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.1);
  border: 1px solid var(--accent-cyan-dim);
  border-radius: 6px;
  color: var(--accent-cyan);
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.option-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.option-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.option-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.4;
}

/* Actions */
.interrupt-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
