<script lang="ts" setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NButton, NIcon, NTag, NTooltip } from 'naive-ui'
import { AlertCircle, CheckmarkCircle, Close, Copy, List, Refresh, Reload, StopCircle, Terminal, Time } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/sessionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { ListRuntimeTasks, ReadRuntimeTaskOutput, StopRuntimeTask } from '../../../wailsjs/go/service/ChatService'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { tools } from '../../../wailsjs/go/models'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

const { t } = useI18n()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()

const tasks = ref<tools.RuntimeTaskSnapshot[]>([])
const selectedTaskId = ref('')
const output = ref<tools.RuntimeTaskOutput | null>(null)
const loading = ref(false)
const outputLoading = ref(false)
const stoppingTaskId = ref('')
const panelRef = ref<HTMLElement | null>(null)

let refreshTimer: number | null = null
let eventCleanups: Array<() => void> = []

const activeSessionId = computed(() => sessionStore.activeSessionId || '')
const selectedTask = computed(() => tasks.value.find((task) => task.id === selectedTaskId.value) || null)
const runningCount = computed(() => tasks.value.filter((task) => isRunningStatus(task.status)).length)
const completedCount = computed(() => tasks.value.filter((task) => isDoneStatus(task.status)).length)

function close() {
  emit('update:show', false)
}

function isRunningStatus(status?: string) {
  return status === 'running' || status === 'pending'
}

function isDoneStatus(status?: string) {
  return status === 'completed' || status === 'stopped' || status === 'killed'
}

function statusType(status?: string) {
  if (status === 'failed') return 'error'
  if (status === 'stopped' || status === 'killed') return 'warning'
  if (status === 'completed') return 'success'
  if (isRunningStatus(status)) return 'info'
  return 'default'
}

function statusIcon(status?: string) {
  if (status === 'completed') return CheckmarkCircle
  if (status === 'failed') return AlertCircle
  if (status === 'stopped' || status === 'killed') return StopCircle
  if (isRunningStatus(status)) return Reload
  return Time
}

function upsertTask(task: tools.RuntimeTaskSnapshot) {
  if (!task?.id) return
  if (task.sessionId && task.sessionId !== activeSessionId.value) return
  const next = [...tasks.value]
  const idx = next.findIndex((item) => item.id === task.id)
  if (idx >= 0) next[idx] = task
  else next.unshift(task)
  next.sort((a, b) => (b.startedAt || 0) - (a.startedAt || 0))
  tasks.value = next
  if (!selectedTaskId.value) selectedTaskId.value = task.id
}

async function refreshTasks() {
  if (!activeSessionId.value) {
    tasks.value = []
    selectedTaskId.value = ''
    output.value = null
    return
  }
  loading.value = true
  try {
    const list = await ListRuntimeTasks(activeSessionId.value)
    tasks.value = (list || []).sort((a, b) => (b.startedAt || 0) - (a.startedAt || 0))
    if (selectedTaskId.value && !tasks.value.some((task) => task.id === selectedTaskId.value)) {
      selectedTaskId.value = ''
      output.value = null
    }
    if (!selectedTaskId.value && tasks.value.length > 0) {
      selectedTaskId.value = tasks.value[0].id
    }
  } catch (e) {
    feedback.error(t('runtimeTasks.actions.refresh'), e)
  } finally {
    loading.value = false
  }
}

async function readOutput() {
  if (!selectedTaskId.value) {
    output.value = null
    return
  }
  outputLoading.value = true
  try {
    output.value = await ReadRuntimeTaskOutput(selectedTaskId.value, 0, 128 * 1024)
  } catch (e) {
    feedback.error(t('runtimeTasks.actions.readOutput'), e)
  } finally {
    outputLoading.value = false
  }
}

async function selectTask(taskId: string) {
  selectedTaskId.value = taskId
  await readOutput()
}

async function stopTask(taskId: string) {
  if (!taskId || stoppingTaskId.value) return
  stoppingTaskId.value = taskId
  try {
    const snapshot = await StopRuntimeTask(taskId)
    upsertTask(snapshot)
    await readOutput()
  } catch (e) {
    feedback.error(t('runtimeTasks.actions.stop'), e)
  } finally {
    stoppingTaskId.value = ''
  }
}

async function copyOutput() {
  if (!output.value?.content) return
  await navigator.clipboard.writeText(output.value.content)
  feedback.success(t('runtimeTasks.outputCopied'))
}

function formatTime(ms?: number) {
  if (!ms) return '-'
  return new Date(ms).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDuration(task: tools.RuntimeTaskSnapshot) {
  if (!task.startedAt) return '-'
  const end = task.finishedAt || Date.now()
  const seconds = Math.max(0, Math.round((end - task.startedAt) / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${seconds % 60}s`
}

function taskTitle(task: tools.RuntimeTaskSnapshot) {
  return task.description || task.command || task.id
}

function startTimer() {
  stopTimer()
  refreshTimer = window.setInterval(() => {
    if (!props.show) return
    refreshTasks()
    if (selectedTask.value && isRunningStatus(selectedTask.value.status)) {
      readOutput()
    }
  }, 4000)
}

function stopTimer() {
  if (refreshTimer !== null) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
}

watch(() => props.show, async (visible) => {
  if (!visible) return
  await refreshTasks()
  await readOutput()
  nextTick(() => panelRef.value?.focus())
})

watch(activeSessionId, async () => {
  if (!props.show) return
  selectedTaskId.value = ''
  output.value = null
  await refreshTasks()
  await readOutput()
})

watch(selectedTaskId, () => {
  if (!props.show) return
  readOutput()
})

onMounted(() => {
  eventCleanups = [
    EventsOn('runtime:task_started', upsertTask),
    EventsOn('runtime:task_completed', (task: tools.RuntimeTaskSnapshot) => {
      upsertTask(task)
      if (task?.id === selectedTaskId.value) readOutput()
    }),
    EventsOn('runtime:task_stopped', (task: tools.RuntimeTaskSnapshot) => {
      upsertTask(task)
      if (task?.id === selectedTaskId.value) readOutput()
    }),
  ]
  startTimer()
})

onUnmounted(() => {
  eventCleanups.forEach((cleanup) => cleanup())
  eventCleanups = []
  stopTimer()
})
</script>

<template>
  <div class="runtime-tasks-shell" :class="{ open: show }">
    <button type="button" class="tasks-backdrop" @click="close" />
    <aside ref="panelRef" class="runtime-tasks-panel" tabindex="-1" :aria-label="t('runtimeTasks.title')">
      <header class="tasks-head">
        <div class="tasks-title-block">
          <span class="tasks-kicker">{{ t('runtimeTasks.kicker') }}</span>
          <strong>{{ t('runtimeTasks.title') }}</strong>
        </div>
        <div class="tasks-head-actions">
          <NTooltip trigger="hover">
            <template #trigger>
              <NButton quaternary circle size="small" :loading="loading" :aria-label="t('runtimeTasks.refresh')" @click="refreshTasks">
                <template #icon><Refresh /></template>
              </NButton>
            </template>
            {{ t('runtimeTasks.refresh') }}
          </NTooltip>
          <NButton quaternary circle size="small" :aria-label="t('common.cancel')" @click="close">
            <template #icon><Close /></template>
          </NButton>
        </div>
      </header>

      <div class="tasks-stats">
        <span><strong>{{ tasks.length }}</strong>{{ t('runtimeTasks.total') }}</span>
        <span><strong>{{ runningCount }}</strong>{{ t('runtimeTasks.running') }}</span>
        <span><strong>{{ completedCount }}</strong>{{ t('runtimeTasks.completed') }}</span>
      </div>

      <div class="tasks-body">
        <section class="tasks-list-pane">
          <div v-if="tasks.length === 0" class="tasks-empty">
            <NIcon size="24"><List /></NIcon>
            <span>{{ activeSessionId ? t('runtimeTasks.empty') : t('runtimeTasks.noSession') }}</span>
          </div>
          <button
            v-for="task in tasks"
            :key="task.id"
            type="button"
            :class="['task-row', { active: task.id === selectedTaskId }]"
            @click="selectTask(task.id)"
          >
            <NIcon size="15" :class="['task-status-icon', task.status]">
              <component :is="statusIcon(task.status)" />
            </NIcon>
            <span class="task-row-main">
              <span class="task-row-title" :title="taskTitle(task)">{{ taskTitle(task) }}</span>
              <span class="task-row-meta">
                {{ task.type || 'task' }} · {{ formatTime(task.startedAt) }} · {{ formatDuration(task) }}
              </span>
            </span>
            <NTag size="small" :type="statusType(task.status)">{{ task.status || 'unknown' }}</NTag>
          </button>
        </section>

        <section class="task-detail-pane">
          <template v-if="selectedTask">
            <div class="detail-head">
              <div class="detail-title">
                <NIcon size="17"><Terminal /></NIcon>
                <span>{{ taskTitle(selectedTask) }}</span>
              </div>
              <div class="detail-actions">
                <NTooltip trigger="hover">
                  <template #trigger>
                    <NButton quaternary circle size="small" :disabled="!output?.content" :aria-label="t('runtimeTasks.copyOutput')" @click="copyOutput">
                      <template #icon><Copy /></template>
                    </NButton>
                  </template>
                  {{ t('runtimeTasks.copyOutput') }}
                </NTooltip>
                <NTooltip trigger="hover">
                  <template #trigger>
                    <NButton quaternary circle size="small" :loading="outputLoading" :aria-label="t('runtimeTasks.refreshOutput')" @click="readOutput">
                      <template #icon><Refresh /></template>
                    </NButton>
                  </template>
                  {{ t('runtimeTasks.refreshOutput') }}
                </NTooltip>
                <NButton
                  v-if="isRunningStatus(selectedTask.status)"
                  size="small"
                  type="error"
                  ghost
                  :loading="stoppingTaskId === selectedTask.id"
                  @click="stopTask(selectedTask.id)"
                >
                  <template #icon><StopCircle /></template>
                  {{ t('runtimeTasks.stop') }}
                </NButton>
              </div>
            </div>

            <div class="detail-meta">
              <span>{{ selectedTask.id }}</span>
              <span>{{ selectedTask.type }}</span>
              <span>{{ selectedTask.status }}</span>
              <span>{{ formatDuration(selectedTask) }}</span>
            </div>
            <pre v-if="selectedTask.command" class="task-command">{{ selectedTask.command }}</pre>
            <pre class="task-output">{{ output?.content || t('runtimeTasks.noOutput') }}</pre>
            <div v-if="output?.truncated" class="output-note">{{ t('runtimeTasks.truncated', { size: output.size }) }}</div>
            <div v-if="selectedTask.error" class="task-error">{{ selectedTask.error }}</div>
          </template>
          <div v-else class="detail-empty">
            <NIcon size="26"><Terminal /></NIcon>
            <span>{{ t('runtimeTasks.selectTask') }}</span>
          </div>
        </section>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.runtime-tasks-shell {
  position: fixed;
  inset: 0;
  z-index: 1700;
  pointer-events: none;
}

.runtime-tasks-shell.open {
  pointer-events: auto;
}

.tasks-backdrop {
  position: absolute;
  inset: 0;
  border: none;
  background: rgba(2, 6, 23, 0.5);
  opacity: 0;
  transition: opacity var(--transition-ui);
}

.runtime-tasks-shell.open .tasks-backdrop {
  opacity: 1;
}

.runtime-tasks-panel {
  position: absolute;
  top: 56px;
  right: 0;
  bottom: 0;
  width: min(860px, 94vw);
  display: flex;
  flex-direction: column;
  background: color-mix(in srgb, var(--bg-surface) 96%, black);
  border-left: 1px solid var(--border-subtle);
  box-shadow: -22px 0 48px rgba(0, 0, 0, 0.38);
  transform: translateX(100%);
  transition: transform 210ms var(--ease-out);
  outline: none;
}

.runtime-tasks-shell.open .runtime-tasks-panel {
  transform: translateX(0);
}

.tasks-head {
  height: 62px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: 0 var(--space-lg);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.tasks-title-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.tasks-title-block strong {
  color: var(--text-primary);
  font-size: var(--fs-lg);
  line-height: 1.2;
}

.tasks-kicker {
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  letter-spacing: 0.7px;
  text-transform: uppercase;
}

.tasks-head-actions {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.tasks-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-lg);
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.tasks-stats span {
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 5px;
}

.tasks-stats strong {
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: var(--fs-md);
}

.tasks-body {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(240px, 310px) minmax(0, 1fr);
}

.tasks-list-pane {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: var(--space-sm);
  border-right: 1px solid var(--border-subtle);
}

.tasks-empty,
.detail-empty {
  height: 100%;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  color: var(--text-faint);
  font-size: var(--fs-sm);
  text-align: center;
}

.task-row {
  width: 100%;
  min-height: 58px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm);
  text-align: left;
  cursor: pointer;
  transition: background var(--transition-ui), border-color var(--transition-ui);
}

.task-row:hover,
.task-row.active {
  background: var(--bg-elevated);
  border-color: var(--border-strong);
}

.task-status-icon {
  color: var(--text-faint);
}

.task-status-icon.running,
.task-status-icon.pending {
  color: var(--accent-cyan);
  animation: spin 1.3s linear infinite;
}

.task-status-icon.completed {
  color: var(--accent-emerald);
}

.task-status-icon.failed {
  color: var(--accent-rose);
}

.task-status-icon.stopped,
.task-status-icon.killed {
  color: var(--accent-amber);
}

.task-row-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.task-row-title,
.detail-title span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-row-title {
  color: var(--text-primary);
  font-size: var(--fs-sm);
}

.task-row-meta,
.detail-meta {
  color: var(--text-faint);
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
}

.task-detail-pane {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.detail-head {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: var(--space-md) var(--space-lg);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.detail-title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  color: var(--text-primary);
  font-weight: var(--fw-semibold);
}

.detail-title .n-icon {
  color: var(--accent-cyan);
  flex-shrink: 0;
}

.detail-actions {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  flex-shrink: 0;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-lg);
  border-bottom: 1px solid var(--border-subtle);
}

.task-command,
.task-output {
  margin: 0;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.task-command {
  color: var(--accent-cyan);
  background: var(--bg-deepest);
  border-bottom: 1px solid var(--border-subtle);
  padding: var(--space-sm) var(--space-lg);
  max-height: 110px;
  overflow: auto;
  flex-shrink: 0;
}

.task-output {
  flex: 1;
  min-height: 0;
  overflow: auto;
  color: var(--text-secondary);
  background: #050b16;
  padding: var(--space-lg);
}

.output-note,
.task-error {
  flex-shrink: 0;
  padding: var(--space-sm) var(--space-lg);
  border-top: 1px solid var(--border-subtle);
  font-size: var(--fs-xs);
}

.output-note {
  color: var(--accent-amber);
}

.task-error {
  color: var(--accent-rose);
}

@media (max-width: 720px) {
  .tasks-body {
    grid-template-columns: 1fr;
  }

  .tasks-list-pane {
    max-height: 220px;
    border-right: none;
    border-bottom: 1px solid var(--border-subtle);
  }

  .tasks-stats {
    grid-template-columns: 1fr;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
