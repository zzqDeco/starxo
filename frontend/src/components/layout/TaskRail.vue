<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { CheckmarkCircle, AlertCircle, Reload, EllipseOutline } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import { useI18n } from 'vue-i18n'

interface RailTask {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'done' | 'failed' | 'skipped'
  detail?: string
  dependsOn?: string[]
  source: 'todos' | 'plan'
}

const chatStore = useChatStore()
const { t } = useI18n()

const filter = ref<'all' | 'running' | 'failed' | 'done'>('all')

const todoTasks = computed<RailTask[]>(() => {
  return chatStore.latestTodos.map((t) => ({
    id: t.id,
    title: t.title,
    status: t.status === 'blocked' ? 'pending' : t.status,
    dependsOn: t.depends_on || [],
    source: 'todos',
  }))
})

const planTasks = computed<RailTask[]>(() => {
  return chatStore.planSteps.map((s) => ({
    id: String(s.taskId),
    title: s.desc,
    status: s.status === 'doing' ? 'in_progress' : (s.status === 'todo' ? 'pending' : s.status),
    detail: s.execResult,
    source: 'plan',
  }))
})

const tasks = computed<RailTask[]>(() => {
  if (todoTasks.value.length > 0) return todoTasks.value
  return planTasks.value
})

const filteredTasks = computed<RailTask[]>(() => {
  switch (filter.value) {
    case 'running':
      return tasks.value.filter(t => t.status === 'in_progress')
    case 'failed':
      return tasks.value.filter(t => t.status === 'failed')
    case 'done':
      return tasks.value.filter(t => t.status === 'done' || t.status === 'skipped')
    default:
      return tasks.value
  }
})

const doneCount = computed(() => tasks.value.filter(t => t.status === 'done' || t.status === 'skipped').length)
const runningCount = computed(() => tasks.value.filter(t => t.status === 'in_progress').length)
const failedCount = computed(() => tasks.value.filter(t => t.status === 'failed').length)
const progressPercent = computed(() => {
  if (tasks.value.length === 0) return 0
  return Math.round((doneCount.value / tasks.value.length) * 100)
})

function statusIcon(status: RailTask['status']) {
  if (status === 'done' || status === 'skipped') return CheckmarkCircle
  if (status === 'failed') return AlertCircle
  if (status === 'in_progress') return Reload
  return EllipseOutline
}

function statusClass(status: RailTask['status']): string {
  if (status === 'in_progress') return 'is-running'
  if (status === 'done' || status === 'skipped') return 'is-done'
  if (status === 'failed') return 'is-failed'
  return 'is-pending'
}

const sourceLabel = computed(() =>
  todoTasks.value.length > 0 ? t('taskRail.sourceTodos') : t('taskRail.sourcePlan')
)
</script>

<template>
  <section class="task-rail">
    <header class="task-head">
      <div class="task-title-row">
        <h3 class="task-title">{{ t('taskRail.title') }}</h3>
        <span class="task-source">{{ sourceLabel }}</span>
      </div>

        <div class="task-metrics">
        <span class="metric metric-running">{{ runningCount }} {{ t('taskRail.running') }}</span>
        <span class="metric metric-failed">{{ failedCount }} {{ t('taskRail.failed') }}</span>
        <span class="metric metric-done">{{ doneCount }}/{{ tasks.length }} {{ t('taskRail.done') }}</span>
      </div>

      <div class="task-progress">
        <div class="task-progress-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>

      <div class="task-filters">
        <NButton size="tiny" quaternary :type="filter === 'all' ? 'primary' : 'default'" @click="filter = 'all'">{{ t('taskRail.filterAll') }}</NButton>
        <NButton size="tiny" quaternary :type="filter === 'running' ? 'primary' : 'default'" @click="filter = 'running'">{{ t('taskRail.filterRunning') }}</NButton>
        <NButton size="tiny" quaternary :type="filter === 'failed' ? 'primary' : 'default'" @click="filter = 'failed'">{{ t('taskRail.filterFailed') }}</NButton>
        <NButton size="tiny" quaternary :type="filter === 'done' ? 'primary' : 'default'" @click="filter = 'done'">{{ t('taskRail.filterDone') }}</NButton>
      </div>
    </header>

    <div class="task-body">
      <div v-if="filteredTasks.length === 0" class="task-empty">{{ t('taskRail.empty') }}</div>

      <div
        v-for="task in filteredTasks"
        :key="task.id"
        class="task-item"
        :class="statusClass(task.status)"
      >
        <div class="task-item-status">
          <NIcon size="14" :class="statusClass(task.status)">
            <component :is="statusIcon(task.status)" />
          </NIcon>
        </div>

        <div class="task-item-main">
          <div class="task-item-title">{{ task.title }}</div>
          <div v-if="task.detail" class="task-item-detail">{{ task.detail }}</div>
          <div v-if="task.dependsOn && task.dependsOn.length > 0" class="task-item-deps">
            {{ t('taskRail.dependsOn') }}: {{ task.dependsOn.join(', ') }}
          </div>
        </div>

        <div class="task-item-id">#{{ task.id }}</div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.task-rail {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, rgba(26, 29, 51, 0.85) 0%, rgba(20, 23, 38, 0.95) 100%);
  border-bottom: 1px solid var(--border-subtle);
}

.task-head {
  padding: 12px 14px 10px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.task-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.task-title {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.7px;
  text-transform: uppercase;
  color: var(--text-primary);
}

.task-source {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
}

.task-metrics {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.metric {
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--text-faint);
}

.metric-running { color: var(--accent-cyan); }
.metric-failed { color: var(--accent-rose); }
.metric-done { color: var(--accent-emerald); }

.task-progress {
  height: 4px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
  margin-bottom: 8px;
}

.task-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-cyan), var(--accent-emerald));
  transition: width 220ms ease;
}

.task-filters {
  display: flex;
  gap: 4px;
}

.task-body {
  flex: 1;
  overflow: auto;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.task-empty {
  color: var(--text-faint);
  font-size: 11px;
  text-align: center;
  padding: 16px 0;
}

.task-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: rgba(8, 10, 20, 0.45);
}

.task-item-main {
  min-width: 0;
  flex: 1;
}

.task-item-title {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.35;
}

.task-item-detail {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  margin-top: 3px;
  line-height: 1.35;
}

.task-item-deps {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  margin-top: 3px;
}

.task-item-id {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.is-running .task-item-title { color: var(--text-primary); }
.is-running .task-item-status {
  color: var(--accent-cyan);
  animation: spin 1.5s linear infinite;
}
.is-done .task-item-title {
  color: var(--text-muted);
  text-decoration: line-through;
}
.is-done .task-item-status { color: var(--accent-emerald); }
.is-failed {
  border-color: rgba(244, 63, 94, 0.35);
  background: rgba(244, 63, 94, 0.08);
}
.is-failed .task-item-status,
.is-failed .task-item-title { color: var(--accent-rose); }
.is-pending .task-item-status { color: var(--text-faint); }

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
