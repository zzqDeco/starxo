<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon } from 'naive-ui'
import { CheckmarkCircle, AlertCircle, Reload, EllipseOutline, ChevronForward } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import { useI18n } from 'vue-i18n'

interface FloatingTask {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'done' | 'failed' | 'skipped'
  detail?: string
}

const chatStore = useChatStore()
const { t } = useI18n()
const expanded = ref(false)

const todoTasks = computed<FloatingTask[]>(() => {
  return chatStore.latestTodos.map((task) => ({
    id: task.id,
    title: task.title,
    status: task.status === 'blocked' ? 'pending' : task.status,
  }))
})

const planTasks = computed<FloatingTask[]>(() => {
  return chatStore.planSteps.map((step) => ({
    id: String(step.taskId),
    title: step.desc,
    status: step.status === 'doing' ? 'in_progress' : (step.status === 'todo' ? 'pending' : step.status),
    detail: step.execResult,
  }))
})

const tasks = computed<FloatingTask[]>(() => {
  if (todoTasks.value.length > 0) return todoTasks.value
  return planTasks.value
})

const currentTask = computed(() => {
  return tasks.value.find((task) => task.status === 'in_progress')
    || tasks.value.find((task) => task.status === 'pending')
    || tasks.value.find((task) => task.status === 'failed')
    || tasks.value[tasks.value.length - 1]
})

const shownTasks = computed(() => tasks.value.slice(0, 8))
const doneCount = computed(() => tasks.value.filter((t) => t.status === 'done' || t.status === 'skipped').length)
const failedCount = computed(() => tasks.value.filter((t) => t.status === 'failed').length)
const runningCount = computed(() => tasks.value.filter((t) => t.status === 'in_progress').length)
const progressPercent = computed(() => {
  if (tasks.value.length === 0) return 0
  return Math.round((doneCount.value / tasks.value.length) * 100)
})

function toggleExpanded() {
  expanded.value = !expanded.value
}

function statusIcon(status: FloatingTask['status']) {
  if (status === 'done' || status === 'skipped') return CheckmarkCircle
  if (status === 'failed') return AlertCircle
  if (status === 'in_progress') return Reload
  return EllipseOutline
}

function statusClass(status: FloatingTask['status']) {
  if (status === 'in_progress') return 'is-running'
  if (status === 'done' || status === 'skipped') return 'is-done'
  if (status === 'failed') return 'is-failed'
  return 'is-pending'
}
</script>

<template>
  <section class="task-float" :class="{ expanded }">
    <button type="button" class="task-head" @click="toggleExpanded">
      <span class="task-title">{{ t('taskRail.title') }}</span>
      <span class="task-summary">{{ doneCount }}/{{ tasks.length }}</span>
      <span class="task-current" :title="currentTask?.title">
        {{ currentTask?.title || t('taskRail.empty') }}
      </span>
      <span class="task-metrics">
        <span class="metric running">{{ runningCount }}</span>
        <span class="metric failed">{{ failedCount }}</span>
      </span>
      <span class="task-chevron" :class="{ expanded }">
        <NIcon size="12"><ChevronForward /></NIcon>
      </span>
    </button>

    <div class="task-progress">
      <div class="task-progress-fill" :style="{ width: progressPercent + '%' }"></div>
    </div>

    <transition name="task-expand">
      <div v-if="expanded" class="task-list">
        <div v-if="shownTasks.length === 0" class="task-empty">{{ t('taskRail.empty') }}</div>
        <div
          v-for="task in shownTasks"
          :key="task.id"
          class="task-item"
          :class="statusClass(task.status)"
        >
          <NIcon size="12" class="task-item-icon" :class="statusClass(task.status)">
            <component :is="statusIcon(task.status)" />
          </NIcon>
          <span class="task-item-title" :title="task.title">{{ task.title }}</span>
          <span class="task-item-id">#{{ task.id }}</span>
        </div>
      </div>
    </transition>
  </section>
</template>

<style scoped>
.task-float {
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(23, 26, 45, 0.9) 0%, rgba(18, 21, 35, 0.95) 100%);
  box-shadow: var(--shadow-level-2);
}

.task-head {
  width: 100%;
  border: none;
  background: transparent;
  padding: 8px 10px;
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
}

.task-title {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.7px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.task-summary {
  font-size: 11px;
  color: var(--accent-emerald);
  font-family: var(--font-mono);
}

.task-current {
  font-size: 12px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.task-metrics {
  display: inline-flex;
  gap: 6px;
}

.metric {
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 999px;
  font-size: 10px;
  font-family: var(--font-mono);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.metric.running {
  color: var(--accent-cyan);
  background: rgba(34, 211, 238, 0.12);
}

.metric.failed {
  color: var(--accent-rose);
  background: rgba(244, 63, 94, 0.12);
}

.task-chevron {
  color: var(--text-faint);
  transition: transform 160ms ease;
}

.task-chevron.expanded {
  transform: rotate(90deg);
}

.task-progress {
  height: 3px;
  margin: 0 10px 8px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.08);
}

.task-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-cyan), var(--accent-emerald));
  transition: width 220ms ease;
}

.task-list {
  max-height: 180px;
  overflow: auto;
  padding: 0 8px 8px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.task-empty {
  font-size: 11px;
  color: var(--text-faint);
  text-align: center;
  padding: 8px 0;
}

.task-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid transparent;
}

.task-item-title {
  min-width: 0;
  flex: 1;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.task-item-id {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
}

.task-item-icon.is-running {
  color: var(--accent-cyan);
  animation: var(--motion-spin);
}

.task-item-icon.is-done {
  color: var(--accent-emerald);
}

.task-item-icon.is-failed {
  color: var(--accent-rose);
}

.task-item-icon.is-pending {
  color: var(--text-faint);
}

.task-item.is-failed {
  border-color: rgba(244, 63, 94, 0.3);
}

.task-expand-enter-active,
.task-expand-leave-active {
  transition: all 170ms ease;
  overflow: hidden;
}

.task-expand-enter-from,
.task-expand-leave-to {
  opacity: 0;
  max-height: 0;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
