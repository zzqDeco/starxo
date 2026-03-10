<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon } from 'naive-ui'
import { CheckmarkCircle, AlertCircle, Reload, EllipseOutline, ChevronForward } from '@vicons/ionicons5'
import { useChatStore } from '@/stores/chatStore'
import type { UnifiedTaskStatus } from '@/stores/chatStore'
import { useI18n } from 'vue-i18n'

const chatStore = useChatStore()
const { t } = useI18n()
const expanded = ref(false)

const tasks = computed(() => chatStore.unifiedTasks)
const taskStats = computed(() => chatStore.unifiedTaskStats)

const shownTasks = computed(() => tasks.value.slice(0, 8))

function toggleExpanded() {
  expanded.value = !expanded.value
}

function statusIcon(status: UnifiedTaskStatus) {
  if (status === 'done') return CheckmarkCircle
  if (status === 'blocked') return AlertCircle
  if (status === 'doing') return Reload
  return EllipseOutline
}

function statusClass(status: UnifiedTaskStatus) {
  if (status === 'doing') return 'is-doing'
  if (status === 'done') return 'is-done'
  if (status === 'blocked') return 'is-blocked'
  return 'is-todo'
}
</script>

<template>
  <section class="task-float" :class="{ expanded }">
    <button type="button" class="task-head" @click="toggleExpanded">
      <span class="task-title">{{ t('taskRail.title') }}</span>
      <span class="task-summary">{{ taskStats.done }}/{{ taskStats.total }}</span>
      <span class="task-current" :title="taskStats.currentTask?.title">
        {{ taskStats.currentTask?.title || t('taskRail.empty') }}
      </span>
      <span class="task-metrics">
        <span class="metric doing">{{ taskStats.counts.doing }}</span>
        <span class="metric blocked">{{ taskStats.counts.blocked }}</span>
      </span>
      <span class="task-chevron" :class="{ expanded }">
        <NIcon size="12"><ChevronForward /></NIcon>
      </span>
    </button>

    <div class="task-progress">
      <div class="task-progress-fill" :style="{ width: taskStats.progressPercent + '%' }"></div>
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
  --task-font-xs: 10px;
  --task-font-sm: 11px;
  --task-font-md: 12px;
  --task-line-height: 1.4;
  --task-space-1: 6px;
  --task-space-2: 8px;
  --task-space-3: 10px;
  --task-status-todo: var(--text-faint);
  --task-status-doing: var(--accent-cyan);
  --task-status-done: var(--accent-emerald);
  --task-status-blocked: var(--accent-rose);

  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(23, 26, 45, 0.9) 0%, rgba(18, 21, 35, 0.95) 100%);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.24);
}

.task-head {
  width: 100%;
  border: none;
  background: transparent;
  padding: var(--task-space-2) var(--task-space-3);
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--task-space-2);
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
}

.task-title {
  font-size: var(--task-font-xs);
  font-weight: 700;
  letter-spacing: 0.7px;
  text-transform: uppercase;
  color: var(--text-faint);
}

.task-summary {
  font-size: var(--task-font-sm);
  color: var(--task-status-done);
  font-family: var(--font-mono);
}

.task-current {
  font-size: var(--task-font-md);
  line-height: var(--task-line-height);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.task-metrics {
  display: inline-flex;
  gap: var(--task-space-1);
}

.metric {
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 999px;
  font-size: var(--task-font-xs);
  font-family: var(--font-mono);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.metric.doing {
  color: var(--task-status-doing);
  background: color-mix(in srgb, var(--task-status-doing) 12%, transparent);
}

.metric.blocked {
  color: var(--task-status-blocked);
  background: color-mix(in srgb, var(--task-status-blocked) 12%, transparent);
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
  margin: 0 var(--task-space-3) var(--task-space-2);
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.08);
}

.task-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--task-status-doing), var(--task-status-done));
  transition: width 220ms ease;
}

.task-list {
  max-height: 180px;
  overflow: auto;
  padding: 0 var(--task-space-2) var(--task-space-2);
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.task-empty {
  font-size: var(--task-font-sm);
  color: var(--text-faint);
  text-align: center;
  padding: var(--task-space-2) 0;
}

.task-item {
  display: flex;
  align-items: center;
  gap: var(--task-space-1);
  padding: var(--task-space-1) var(--task-space-2);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid transparent;
}

.task-item-title {
  min-width: 0;
  flex: 1;
  font-size: var(--task-font-sm);
  line-height: var(--task-line-height);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.task-item-id {
  font-size: var(--task-font-xs);
  color: var(--text-faint);
  font-family: var(--font-mono);
}

.task-item-icon.is-doing {
  color: var(--task-status-doing);
  animation: spin 1.5s linear infinite;
}

.task-item-icon.is-done {
  color: var(--task-status-done);
}

.task-item-icon.is-blocked {
  color: var(--task-status-blocked);
}

.task-item-icon.is-todo {
  color: var(--task-status-todo);
}

.task-item.is-blocked {
  border-color: color-mix(in srgb, var(--task-status-blocked) 35%, transparent);
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
