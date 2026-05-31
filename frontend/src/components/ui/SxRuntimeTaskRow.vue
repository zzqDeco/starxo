<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon } from './icons'

const props = withDefaults(defineProps<{
  title: string
  meta?: string
  status?: 'pending' | 'running' | 'completed' | 'failed' | 'stopped' | 'killed' | string
  selected?: boolean
}>(), {
  meta: '',
  status: 'pending',
  selected: false,
})

const tone = computed(() => {
  if (props.status === 'completed') return 'success'
  if (props.status === 'failed') return 'danger'
  if (props.status === 'running' || props.status === 'pending') return 'accent'
  return 'neutral'
})

const icon = computed(() => {
  if (props.status === 'completed') return resolveSxIcon('check')
  if (props.status === 'failed') return resolveSxIcon('error')
  if (props.status === 'stopped' || props.status === 'killed') return resolveSxIcon('stop')
  return resolveSxIcon('clock')
})
</script>

<template>
  <button type="button" :class="['sx-runtime-task-row', `tone-${tone}`, { selected }]">
    <component :is="icon" :size="15" stroke-width="2" />
    <span class="sx-task-copy">
      <strong :title="title">{{ title }}</strong>
      <span v-if="meta">{{ meta }}</span>
    </span>
    <slot name="after" />
  </button>
</template>

<style scoped>
.sx-runtime-task-row {
  width: 100%;
  min-height: 52px;
  border: 1px solid transparent;
  border-radius: var(--sx-radius-8);
  background: transparent;
  color: var(--sx-text-secondary);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  text-align: left;
  cursor: pointer;
}

.sx-runtime-task-row:hover,
.sx-runtime-task-row.selected {
  background: var(--sx-control-hover);
  border-color: var(--sx-separator);
}

.tone-accent > svg {
  color: var(--sx-accent);
}

.tone-success > svg {
  color: var(--sx-success);
}

.tone-danger > svg {
  color: var(--sx-danger);
}

.sx-task-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sx-task-copy strong,
.sx-task-copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sx-task-copy strong {
  color: var(--sx-text-primary);
  font: 500 12px/16px var(--sx-font-sans);
}

.sx-task-copy span {
  color: var(--sx-text-tertiary);
  font: 400 11px/15px var(--sx-font-sans);
}
</style>
