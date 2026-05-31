<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon, type SxIconName } from './icons'

const props = withDefaults(defineProps<{
  icon?: SxIconName | string
  label: string
  intent?: 'neutral' | 'accent' | 'danger'
  state?: 'default' | 'hover' | 'selected' | 'disabled'
  size?: 'small' | 'medium'
  type?: 'button' | 'submit'
}>(), {
  icon: 'info',
  intent: 'neutral',
  state: 'default',
  size: 'medium',
  type: 'button',
})

const iconComponent = computed(() => resolveSxIcon(props.icon))
const isDisabled = computed(() => props.state === 'disabled')
</script>

<template>
  <button
    class="sx-icon-button"
    :class="[`intent-${intent}`, `state-${state}`, `size-${size}`]"
    :type="type"
    :aria-label="label"
    :title="label"
    :disabled="isDisabled"
  >
    <component :is="iconComponent" :size="size === 'small' ? 15 : 17" stroke-width="1.9" />
  </button>
</template>

<style scoped>
.sx-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border: 1px solid var(--sx-separator);
  background: var(--sx-control-bg);
  color: var(--sx-text-secondary);
  border-radius: var(--sx-radius-8);
  cursor: pointer;
  transition: background var(--transition-ui), color var(--transition-ui), border-color var(--transition-ui), box-shadow var(--transition-ui);
}

.size-small { width: 28px; height: 28px; }
.size-medium { width: 32px; height: 32px; }
.sx-icon-button:hover,
.state-hover {
  background: var(--sx-control-hover);
  color: var(--sx-text-primary);
}
.intent-accent,
.state-selected {
  color: var(--sx-accent);
  border-color: var(--sx-accent-border);
  background: var(--sx-accent-bg);
}
.intent-danger {
  color: var(--sx-danger);
  border-color: var(--sx-danger-border);
}
.state-disabled,
.sx-icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.sx-icon-button:focus-visible {
  outline: 2px solid var(--sx-focus);
  outline-offset: 2px;
}
</style>
