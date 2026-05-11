<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { NAlert, NButton, NCheckbox, NIcon, NInput, NTag, NTooltip } from 'naive-ui'
import { CheckmarkCircle, ChevronDown, CopyOutline, GitBranch, GitMerge, Refresh, Search, Warning } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/sessionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { ExitRuntimeWorktree, GetRuntimeWorktreeState, MergeRuntimeWorktree, ReviewRuntimeWorktree } from '../../../wailsjs/go/service/ChatService'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { service, tools } from '../../../wailsjs/go/models'

const emit = defineEmits<{ (e: 'changed'): void }>()

type WorktreeMergeResult = tools.WorktreeMergeOutput & {
  conflicted?: boolean
  conflictFiles?: string[]
  mergeOutput?: string
  recoveryHint?: string
}

const { t } = useI18n()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()

const state = ref<service.RuntimeWorktreeStateDTO | null>(null)
const review = ref<tools.WorktreeDiffOutput | null>(null)
const lastMerge = ref<WorktreeMergeResult | null>(null)
const lastError = ref('')
const loading = ref(false)
const reviewing = ref(false)
const merging = ref(false)
const exiting = ref(false)
const expanded = ref(true)
const includePatch = ref(true)
const removeWorktree = ref(true)
const commitMessage = ref('Starxo runtime worktree merge')

let eventCleanups: Array<() => void> = []

const activeSessionId = computed(() => sessionStore.activeSessionId || '')
const hasReview = computed(() => !!review.value)
const hasChanges = computed(() => !!review.value && (!!review.value.status || !!review.value.diffStat || !!review.value.diff || !!review.value.untrackedDiff))
const reviewStatusLines = computed(() => nonEmptyLines(review.value?.status || ''))
const diffStatLines = computed(() => nonEmptyLines(review.value?.diffStat || ''))
const reviewPatch = computed(() => review.value?.diff || review.value?.untrackedDiff || '')
const worktreeShortPath = computed(() => compactPath(state.value?.worktreePath || ''))
const hasMergeConflict = computed(() => !!lastMerge.value?.conflicted)

function nonEmptyLines(text: string) {
  return text.split('\n').map((line) => line.trimEnd()).filter((line) => line.trim() !== '')
}

function compactPath(path: string) {
  if (!path) return '-'
  if (path.length <= 64) return path
  return `...${path.slice(-61)}`
}

function resetReview() {
  review.value = null
  lastMerge.value = null
  lastError.value = ''
}

async function refreshState() {
  loading.value = true
  try {
    state.value = await GetRuntimeWorktreeState(activeSessionId.value)
    if (!state.value?.active) resetReview()
  } catch (e) {
    state.value = null
    feedback.error(t('workspace.worktree.actions.refresh'), e)
  } finally {
    loading.value = false
  }
}

async function reviewWorktree() {
  if (!state.value?.active || reviewing.value) return
  reviewing.value = true
  lastError.value = ''
  lastMerge.value = null
  try {
    review.value = await ReviewRuntimeWorktree(activeSessionId.value, includePatch.value, 24000)
  } catch (e) {
    feedback.error(t('workspace.worktree.actions.review'), e)
  } finally {
    reviewing.value = false
  }
}

async function mergeWorktree() {
  if (!state.value?.active || merging.value) return
  const confirmed = await feedback.confirmDanger(t('workspace.worktree.mergeConfirm', {
    branch: state.value.worktreeBranch || '-',
    target: state.value.workspacePath || '-',
  }))
  if (!confirmed) return
  merging.value = true
  lastError.value = ''
  try {
    lastMerge.value = await MergeRuntimeWorktree(activeSessionId.value, commitMessage.value, removeWorktree.value) as WorktreeMergeResult
    if (lastMerge.value.conflicted) {
      feedback.info(t('workspace.worktree.conflictDetected'))
      await refreshState()
      return
    }
    feedback.success(t('workspace.worktree.mergeDone'))
    review.value = null
    await refreshState()
    emit('changed')
  } catch (e) {
    lastError.value = errorText(e)
    feedback.error(t('workspace.worktree.actions.merge'), e)
    await refreshState()
  } finally {
    merging.value = false
  }
}

async function exitKeep() {
  if (!state.value?.active || exiting.value) return
  exiting.value = true
  try {
    await ExitRuntimeWorktree(activeSessionId.value, 'keep', false)
    feedback.success(t('workspace.worktree.exitDone'))
    resetReview()
    await refreshState()
    emit('changed')
  } catch (e) {
    feedback.error(t('workspace.worktree.actions.exit'), e)
  } finally {
    exiting.value = false
  }
}

async function copyReview() {
  const parts = [
    review.value?.status,
    review.value?.diffStat,
    reviewPatch.value,
  ].filter(Boolean)
  if (parts.length === 0) return
  await navigator.clipboard.writeText(parts.join('\n\n'))
  feedback.success(t('workspace.worktree.reviewCopied'))
}

function errorText(e: unknown) {
  if (typeof e === 'string') return e
  if (e && typeof e === 'object' && 'message' in e) return String((e as { message?: unknown }).message || '')
  return String(e || '')
}

watch(activeSessionId, () => {
  resetReview()
  refreshState()
})

async function handleWorktreeChanged(evt: { sessionId?: string }) {
  if (evt?.sessionId && evt.sessionId !== activeSessionId.value) return
  await refreshState()
  emit('changed')
}

onMounted(() => {
  eventCleanups = [
    EventsOn('runtime:worktree_changed', (evt: { sessionId?: string }) => {
      void handleWorktreeChanged(evt)
    }),
  ]
  refreshState()
})

onUnmounted(() => {
  eventCleanups.forEach((cleanup) => cleanup())
  eventCleanups = []
})
</script>

<template>
  <section class="worktree-review">
    <button type="button" class="worktree-head" @click="expanded = !expanded">
      <span class="worktree-title">
        <NIcon size="14"><GitBranch /></NIcon>
        {{ t('workspace.worktree.title') }}
      </span>
      <NTag size="small" :type="state?.active ? 'info' : 'default'">
        {{ state?.active ? t('workspace.worktree.active') : t('workspace.worktree.inactive') }}
      </NTag>
      <span v-if="state?.active" class="worktree-current" :title="state.worktreePath">{{ worktreeShortPath }}</span>
      <span class="worktree-chevron" :class="{ expanded }">
        <NIcon size="13"><ChevronDown /></NIcon>
      </span>
    </button>

    <div v-if="expanded" class="worktree-body">
      <div v-if="!state?.active" class="worktree-empty">
        {{ t('workspace.worktree.empty') }}
      </div>

      <template v-else>
        <div class="worktree-meta">
          <div>
            <span>{{ t('workspace.worktree.branch') }}</span>
            <strong>{{ state.worktreeBranch }}</strong>
          </div>
          <div>
            <span>{{ t('workspace.worktree.original') }}</span>
            <strong :title="state.workspacePath">{{ state.workspacePath }}</strong>
          </div>
          <div>
            <span>{{ t('workspace.worktree.current') }}</span>
            <strong :title="state.worktreePath">{{ state.worktreePath }}</strong>
          </div>
        </div>

        <div class="worktree-actions">
          <NCheckbox v-model:checked="includePatch">{{ t('workspace.worktree.includePatch') }}</NCheckbox>
          <NButton size="tiny" secondary :loading="reviewing" @click="reviewWorktree">
            <template #icon><NIcon size="14"><Search /></NIcon></template>
            {{ t('workspace.worktree.review') }}
          </NButton>
          <NTooltip trigger="hover">
            <template #trigger>
              <NButton size="tiny" quaternary :loading="loading" @click="refreshState">
                <template #icon><NIcon size="14"><Refresh /></NIcon></template>
              </NButton>
            </template>
            {{ t('workspace.worktree.refresh') }}
          </NTooltip>
          <NTooltip trigger="hover">
            <template #trigger>
              <NButton size="tiny" quaternary :disabled="!hasReview" @click="copyReview">
                <template #icon><NIcon size="14"><CopyOutline /></NIcon></template>
              </NButton>
            </template>
            {{ t('workspace.worktree.copyReview') }}
          </NTooltip>
          <NButton size="tiny" quaternary :loading="exiting" @click="exitKeep">
            {{ t('workspace.worktree.exitKeep') }}
          </NButton>
        </div>

        <div v-if="hasReview" class="worktree-review-result">
          <div class="review-summary">
            <span><strong>{{ reviewStatusLines.length }}</strong>{{ t('workspace.worktree.statusLines') }}</span>
            <span><strong>{{ diffStatLines.length }}</strong>{{ t('workspace.worktree.statLines') }}</span>
            <NTag v-if="review?.truncated || review?.untrackedTruncated" size="small" type="warning">
              {{ t('workspace.worktree.truncated') }}
            </NTag>
            <NTag v-else-if="!hasChanges" size="small" type="success">
              <template #icon><NIcon><CheckmarkCircle /></NIcon></template>
              {{ t('workspace.worktree.clean') }}
            </NTag>
          </div>

          <div class="review-grid">
            <div class="review-block">
              <span class="review-label">{{ t('workspace.worktree.status') }}</span>
              <pre>{{ review?.status || t('workspace.worktree.noStatus') }}</pre>
            </div>
            <div class="review-block">
              <span class="review-label">{{ t('workspace.worktree.diffStat') }}</span>
              <pre>{{ review?.diffStat || t('workspace.worktree.noDiffStat') }}</pre>
            </div>
          </div>
          <div v-if="reviewPatch" class="review-block patch">
            <span class="review-label">{{ t('workspace.worktree.patch') }}</span>
            <pre>{{ reviewPatch }}</pre>
          </div>
        </div>

        <NAlert v-if="lastError" type="warning" class="worktree-alert">
          <template #icon><NIcon><Warning /></NIcon></template>
          {{ t('workspace.worktree.mergeFailed') }} {{ lastError }}
        </NAlert>

        <NAlert v-if="hasMergeConflict" type="warning" class="worktree-alert conflict-alert">
          <template #icon><NIcon><Warning /></NIcon></template>
          <div class="conflict-content">
            <strong>{{ t('workspace.worktree.conflictTitle') }}</strong>
            <p>{{ lastMerge?.recoveryHint || t('workspace.worktree.conflictHint') }}</p>
            <div v-if="lastMerge?.conflictFiles?.length" class="conflict-files">
              <span class="review-label">{{ t('workspace.worktree.conflictFiles') }}</span>
              <code v-for="file in lastMerge.conflictFiles" :key="file">{{ file }}</code>
            </div>
            <pre v-if="lastMerge?.mergeOutput">{{ lastMerge.mergeOutput }}</pre>
          </div>
        </NAlert>

        <NAlert v-else-if="lastMerge" type="success" class="worktree-alert">
          {{ lastMerge.message }}
        </NAlert>

        <div class="merge-controls">
          <NInput
            v-model:value="commitMessage"
            size="small"
            :placeholder="t('workspace.worktree.commitPlaceholder')"
          />
          <NCheckbox v-model:checked="removeWorktree">{{ t('workspace.worktree.removeAfterMerge') }}</NCheckbox>
          <NButton size="small" type="primary" :disabled="!hasReview || !hasChanges" :loading="merging" @click="mergeWorktree">
            <template #icon><NIcon size="14"><GitMerge /></NIcon></template>
            {{ t('workspace.worktree.merge') }}
          </NButton>
        </div>
      </template>
    </div>
  </section>
</template>

<style scoped>
.worktree-review {
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
  flex-shrink: 0;
}

.worktree-head {
  width: 100%;
  min-height: 36px;
  border: 0;
  background: transparent;
  color: var(--text-secondary);
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  text-align: left;
}

.worktree-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.7px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.worktree-current {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-muted);
}

.worktree-chevron {
  color: var(--text-faint);
  transform: rotate(-90deg);
  transition: transform 160ms ease;
}

.worktree-chevron.expanded {
  transform: rotate(0deg);
}

.worktree-body {
  padding: 0 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.worktree-empty {
  font-size: 12px;
  color: var(--text-faint);
  padding: 4px 0;
}

.worktree-meta {
  display: grid;
  grid-template-columns: minmax(110px, 0.7fr) repeat(2, minmax(0, 1.2fr));
  gap: 8px;
}

.worktree-meta div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.worktree-meta span,
.review-label {
  font-size: 10px;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.worktree-meta strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
  font-size: 11px;
  font-family: var(--font-mono);
  font-weight: 500;
}

.worktree-actions,
.merge-controls,
.review-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.review-summary {
  font-size: 11px;
  color: var(--text-muted);
}

.review-summary strong {
  margin-right: 3px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.worktree-review-result {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.review-grid {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
  gap: 8px;
}

.review-block {
  min-width: 0;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: var(--bg-deepest);
  padding: 7px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.review-block pre {
  margin: 0;
  max-height: 160px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.45;
  font-family: var(--font-mono);
}

.review-block.patch pre {
  max-height: 240px;
}

.worktree-alert {
  font-size: 12px;
}

.conflict-content {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.conflict-content strong {
  color: var(--text-secondary);
}

.conflict-content p {
  margin: 0;
  color: var(--text-muted);
  line-height: 1.45;
}

.conflict-files {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.conflict-files code {
  max-width: 100%;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  padding: 2px 6px;
  background: var(--bg-deepest);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 11px;
}

.conflict-alert pre {
  margin: 0;
  max-height: 140px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
  font-family: var(--font-mono);
}

.merge-controls {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto auto;
}

@media (max-width: 900px) {
  .worktree-meta,
  .review-grid,
  .merge-controls {
    grid-template-columns: 1fr;
  }
}
</style>
