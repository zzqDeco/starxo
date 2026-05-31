<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon, type SxIconName } from './icons'

const props = withDefaults(defineProps<{
  icon?: SxIconName | string
  label: string
  active?: boolean
  disabled?: boolean
  compact?: boolean
}>(), {
  icon: 'info',
  active: false,
  disabled: false,
  compact: false,
})

const iconComponent = computed(() => resolveSxIcon(props.icon))
</script>

<template>
  <button
    type="button"
    :class="['sx-toolbar-item', { active, compact }]"
    :disabled="disabled"
    :aria-label="label"
  >
    <component :is="iconComponent" :size="15" stroke-width="2" />
    <span v-if="!compact">{{ label }}</span>
  </button>
</template>

<style scoped>
.sx-toolbar-item {
  height: 30px;
  min-width: 30px;
  border: 1px solid transparent;
  border-radius: var(--sx-radius-8);
  background: transparent;
  color: var(--sx-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 9px;
  font: 500 12px/16px var(--sx-font-sans);
  cursor: pointer;
  transition: background var(--transition-ui), color var(--transition-ui), border-color var(--transition-ui);
}

.sx-toolbar-item.compact {
  width: 30px;
  padding: 0;
}

.sx-toolbar-item:hover:not(:disabled),
.sx-toolbar-item.active {
  background: var(--sx-control-hover);
  color: var(--sx-text-primary);
}

.sx-toolbar-item.active {
  border-color: var(--sx-separator);
  background: var(--sx-control-selected);
}

.sx-toolbar-item:disabled {
  opacity: 0.48;
  cursor: default;
}

.sx-toolbar-item:focus-visible {
  outline: 2px solid var(--sx-focus);
  outline-offset: 1px;
}
</style>
