<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon, type SxIconName } from './icons'

const props = withDefaults(defineProps<{
  title: string
  meta?: string
  mode?: 'full' | 'compact'
  selected?: boolean
  status?: 'default' | 'running' | 'error' | 'waiting' | 'plan'
  icon?: SxIconName | string
}>(), {
  meta: '',
  mode: 'full',
  selected: false,
  status: 'default',
  icon: 'session',
})

const iconComponent = computed(() => resolveSxIcon(props.icon))
</script>

<template>
  <div
    class="sx-source-row"
    :class="[`mode-${mode}`, `status-${status}`, { selected }]"
    :title="title"
  >
    <component :is="iconComponent" class="sx-source-row__icon" :size="16" stroke-width="1.9" />
    <template v-if="mode === 'full'">
      <span class="sx-source-row__text">
        <span class="sx-source-row__title">{{ title }}</span>
        <span v-if="meta" class="sx-source-row__meta">{{ meta }}</span>
      </span>
      <span v-if="status !== 'default'" class="sx-source-row__dot" aria-hidden="true"></span>
    </template>
  </div>
</template>

<style scoped>
.sx-source-row {
  display: flex;
  align-items: center;
  min-height: 32px;
  border-radius: var(--sx-radius-8);
  color: var(--sx-text-secondary);
  user-select: none;
}
.mode-full {
  gap: 10px;
  padding: 0 9px;
}
.mode-compact {
  justify-content: center;
  width: 40px;
  height: 36px;
}
.selected {
  background: var(--sx-control-selected);
  color: var(--sx-text-primary);
}
.sx-source-row__icon {
  flex: 0 0 auto;
}
.sx-source-row__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.sx-source-row__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font: 500 12px/16px var(--sx-font-sans);
}
.sx-source-row__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--sx-text-tertiary);
  font: 400 10px/13px var(--sx-font-sans);
}
.sx-source-row__dot {
  margin-left: auto;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sx-text-tertiary);
}
.status-running .sx-source-row__dot { background: var(--sx-warning); }
.status-waiting .sx-source-row__dot { background: var(--sx-accent); }
.status-plan .sx-source-row__dot { background: var(--sx-accent); }
.status-error .sx-source-row__dot { background: var(--sx-danger); }
</style>
