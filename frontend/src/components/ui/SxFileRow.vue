<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon } from './icons'

const props = withDefaults(defineProps<{
  name: string
  meta?: string
  type?: 'file' | 'folder' | 'diff'
  state?: 'default' | 'selected' | 'disabled'
}>(), {
  meta: '',
  type: 'file',
  state: 'default',
})

const icon = computed(() => resolveSxIcon(props.type === 'folder' ? 'folder' : props.type === 'diff' ? 'diff' : 'file'))
</script>

<template>
  <div class="sx-file-row" :class="[`state-${state}`, `type-${type}`]">
    <component :is="icon" class="sx-file-row__icon" :size="16" stroke-width="1.8" />
    <span class="sx-file-row__name">{{ name }}</span>
    <span v-if="meta" class="sx-file-row__meta">{{ meta }}</span>
  </div>
</template>

<style scoped>
.sx-file-row {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 28px;
  padding: 0 8px;
  border-radius: var(--sx-radius-6);
  color: var(--sx-text-primary);
}
.state-selected { background: var(--sx-accent-bg); }
.state-disabled { opacity: 0.55; }
.sx-file-row__icon { color: var(--sx-text-secondary); flex: 0 0 auto; }
.type-diff .sx-file-row__icon { color: var(--sx-accent); }
.sx-file-row__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font: 400 11px/15px var(--sx-font-sans);
}
.state-selected .sx-file-row__name { font-weight: 600; }
.sx-file-row__meta {
  color: var(--sx-text-tertiary);
  font: 400 10px/14px var(--sx-font-sans);
}
</style>
