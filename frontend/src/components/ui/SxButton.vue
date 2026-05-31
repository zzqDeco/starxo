<script setup lang="ts">
withDefaults(defineProps<{
  kind?: 'primary' | 'secondary' | 'danger' | 'ghost'
  state?: 'default' | 'hover' | 'disabled'
  size?: 'small' | 'medium'
  type?: 'button' | 'submit'
}>(), {
  kind: 'secondary',
  state: 'default',
  size: 'medium',
  type: 'button',
})
</script>

<template>
  <button
    class="sx-button"
    :class="[`kind-${kind}`, `state-${state}`, `size-${size}`]"
    :type="type"
    :disabled="state === 'disabled'"
  >
    <slot />
  </button>
</template>

<style scoped>
.sx-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-8);
  background: var(--sx-control-bg);
  color: var(--sx-text-primary);
  font: 500 12px/16px var(--sx-font-sans);
  letter-spacing: 0;
  cursor: pointer;
  transition: background var(--transition-ui), border-color var(--transition-ui), color var(--transition-ui), box-shadow var(--transition-ui);
}
.size-small { height: 24px; padding: 0 12px; }
.size-medium { height: 32px; padding: 0 16px; }
.kind-primary {
  background: var(--sx-accent);
  border-color: var(--sx-accent);
  color: white;
}
.kind-danger {
  background: var(--sx-danger);
  border-color: var(--sx-danger);
  color: white;
}
.kind-ghost {
  border-color: transparent;
  background: transparent;
}
.sx-button:hover,
.state-hover {
  background: var(--sx-control-hover);
}
.kind-primary:hover,
.kind-primary.state-hover { background: var(--sx-accent-hover); }
.kind-danger:hover,
.kind-danger.state-hover { background: var(--sx-danger-pressed); }
.state-disabled,
.sx-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
  background: var(--sx-control-disabled);
  border-color: var(--sx-separator);
  color: var(--sx-text-tertiary);
}
.sx-button:focus-visible {
  outline: 2px solid var(--sx-focus);
  outline-offset: 2px;
}
</style>
