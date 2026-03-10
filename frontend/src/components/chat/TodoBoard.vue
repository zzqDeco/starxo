<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  CheckmarkCircle, EllipseOutline, Reload, CloseCircle,
  LockClosed, ArrowForward, ChevronDown, ChevronForward
} from '@vicons/ionicons5'

export interface TodoItem {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'done' | 'failed' | 'blocked'
  depends_on?: string[]
}

const props = withDefaults(defineProps<{
  todos: TodoItem[]
  compact?: boolean
}>(), { compact: false })

const { t } = useI18n()

const isExpanded = ref(false)

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}

// Build adjacency info for rendering
const todoMap = computed(() => {
  const m = new Map<string, TodoItem>()
  for (const t of props.todos) {
    m.set(t.id, t)
  }
  return m
})

// Topological layers for DAG layout
const layers = computed<TodoItem[][]>(() => {
  const items = props.todos
  if (items.length === 0) return []

  const inDegree = new Map<string, number>()
  const children = new Map<string, string[]>()

  for (const t of items) {
    inDegree.set(t.id, 0)
    children.set(t.id, [])
  }
  for (const t of items) {
    for (const dep of (t.depends_on || [])) {
      inDegree.set(t.id, (inDegree.get(t.id) || 0) + 1)
      const c = children.get(dep) || []
      c.push(t.id)
      children.set(dep, c)
    }
  }

  const result: TodoItem[][] = []
  const visited = new Set<string>()
  let queue = items.filter(t => (inDegree.get(t.id) || 0) === 0)

  while (queue.length > 0) {
    result.push(queue)
    const nextQueue: TodoItem[] = []
    for (const t of queue) {
      visited.add(t.id)
      for (const childId of (children.get(t.id) || [])) {
        const newDeg = (inDegree.get(childId) || 1) - 1
        inDegree.set(childId, newDeg)
        if (newDeg === 0) {
          const child = todoMap.value.get(childId)
          if (child) nextQueue.push(child)
        }
      }
    }
    queue = nextQueue
  }

  // Add any remaining (cyclic or orphaned)
  const remaining = items.filter(t => !visited.has(t.id))
  if (remaining.length > 0) {
    result.push(remaining)
  }

  return result
})

const statusConfig: Record<string, { icon: any; color: string; key: string }> = {
  pending:     { icon: EllipseOutline, color: '#5a5c72', key: 'todo.status.pending' },
  in_progress: { icon: Reload,         color: '#22d3ee', key: 'todo.status.in_progress' },
  done:        { icon: CheckmarkCircle, color: '#34d399', key: 'todo.status.done' },
  failed:      { icon: CloseCircle,     color: '#f43f5e', key: 'todo.status.failed' },
  blocked:     { icon: LockClosed,      color: '#f59e0b', key: 'todo.status.blocked' },
}

function getStatusConfig(status: string) {
  return statusConfig[status] || statusConfig.pending
}

const stats = computed(() => {
  const counts: Record<string, number> = {}
  for (const t of props.todos) {
    counts[t.status] = (counts[t.status] || 0) + 1
  }
  return counts
})

const progressPercent = computed(() => {
  if (props.todos.length === 0) return 0
  const done = (stats.value.done || 0)
  return Math.round((done / props.todos.length) * 100)
})
</script>

<template>
  <div class="todo-board" :class="{ compact }">
    <!-- Header with progress -->
    <div class="todo-header" :class="{ clickable: compact }" @click="compact && toggleExpand()">
      <span class="todo-title">{{ t('todo.taskProgress') }}</span>
      <span class="todo-progress">{{ progressPercent }}%</span>
      <div class="todo-progress-bar">
        <div class="todo-progress-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>
      <div class="todo-stats">
        <span v-for="(count, status) in stats" :key="status" class="todo-stat-badge" :style="{ color: getStatusConfig(status as string).color }">
          {{ count }} {{ t(getStatusConfig(status as string).key) }}
        </span>
      </div>
      <NIcon v-if="compact" size="12" class="expand-toggle" :style="{ color: '#5a5c72' }">
        <ChevronDown v-if="isExpanded" />
        <ChevronForward v-else />
      </NIcon>
    </div>

    <!-- DAG layers -->
    <div v-if="!compact || isExpanded" class="todo-layers">
      <div v-for="(layer, li) in layers" :key="li" class="todo-layer">
        <div class="layer-connector" v-if="li > 0">
          <NIcon size="10" color="#5a5c72"><ArrowForward /></NIcon>
        </div>
        <div class="layer-items">
          <div
            v-for="item in layer"
            :key="item.id"
            class="todo-item"
            :class="`todo-${item.status}`"
            :style="{ '--status-color': getStatusConfig(item.status).color }"
          >
            <div class="todo-item-icon">
              <NIcon size="14" :style="{ color: getStatusConfig(item.status).color }">
                <component :is="getStatusConfig(item.status).icon" />
              </NIcon>
            </div>
            <div class="todo-item-content">
              <span class="todo-item-title">{{ item.title }}</span>
              <div v-if="item.depends_on && item.depends_on.length > 0" class="todo-item-deps">
                <span v-for="dep in item.depends_on" :key="dep" class="todo-dep-tag">
                  {{ todoMap.get(dep)?.title || dep }}
                </span>
              </div>
            </div>
            <span class="todo-item-id">{{ item.id }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.todo-board {
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  margin: 4px 0;
}

.todo-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.todo-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--text-faint);
}

.todo-progress {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--accent-emerald);
}

.todo-progress-bar {
  flex: 1;
  min-width: 60px;
  height: 3px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  overflow: hidden;
}

.todo-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-cyan), var(--accent-emerald));
  border-radius: 2px;
  transition: width 300ms ease;
}

.todo-stats {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

.todo-stat-badge {
  font-size: 10px;
  font-family: var(--font-mono);
  font-weight: 600;
}

/* DAG layers */
.todo-layers {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.todo-layer {
  display: flex;
  flex-direction: column;
}

.layer-connector {
  display: flex;
  justify-content: center;
  padding: 2px 0;
  color: var(--text-faint);
  transform: rotate(90deg);
}

.layer-items {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

/* Todo item */
.todo-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border-radius: 4px;
  border-left: 2px solid var(--status-color);
  background: color-mix(in srgb, var(--status-color) 4%, transparent);
  transition: background 150ms ease;
}

.todo-item:hover {
  background: color-mix(in srgb, var(--status-color) 8%, transparent);
}

.todo-in_progress {
  animation: todoGlow 2s ease-in-out infinite;
}

@keyframes todoGlow {
  0%, 100% { background: color-mix(in srgb, var(--status-color) 4%, transparent); }
  50% { background: color-mix(in srgb, var(--status-color) 10%, transparent); }
}

.todo-item-icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.todo-in_progress .todo-item-icon {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.todo-item-content {
  flex: 1;
  min-width: 0;
}

.todo-item-title {
  font-size: 12px;
  color: var(--text-primary);
  line-height: 1.4;
}

.todo-done .todo-item-title {
  text-decoration: line-through;
  color: var(--text-muted);
}

.todo-item-deps {
  display: flex;
  gap: 4px;
  margin-top: 2px;
  flex-wrap: wrap;
}

.todo-dep-tag {
  font-size: 9px;
  font-family: var(--font-mono);
  color: var(--text-faint);
  background: rgba(255, 255, 255, 0.04);
  padding: 1px 4px;
  border-radius: 2px;
}

.todo-item-id {
  font-size: 9px;
  font-family: var(--font-mono);
  color: var(--text-faint);
  flex-shrink: 0;
}

/* Compact mode */
.todo-board.compact {
  background: var(--bg-surface);
  border-color: var(--border-subtle);
  padding: 6px 10px;
  margin: 4px 0;
}

.todo-board.compact .todo-header {
  margin-bottom: 0;
}

.todo-board.compact .todo-layers {
  margin-top: 8px;
}

.todo-header.clickable {
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background 150ms ease;
}

.todo-header.clickable:hover {
  background: rgba(255, 255, 255, 0.03);
}

.expand-toggle {
  flex-shrink: 0;
  margin-left: 4px;
  transition: transform 150ms ease;
}
</style>
