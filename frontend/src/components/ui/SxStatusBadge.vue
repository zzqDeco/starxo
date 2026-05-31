<script setup lang="ts">
withDefaults(defineProps<{
  tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info'
}>(), {
  tone: 'neutral',
})
</script>

<template>
  <span class="sx-status-badge" :class="`tone-${tone}`">
    <slot />
  </span>
</template>

<style scoped>
.sx-status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 22px;
  padding: 0 10px;
  border: 1px solid currentColor;
  border-radius: var(--sx-radius-pill);
  font: 500 10px/13px var(--sx-font-sans);
  white-space: nowrap;
}
.tone-neutral { color: var(--sx-text-secondary); background: var(--sx-status-neutral-bg); }
.tone-success { color: var(--sx-success); background: var(--sx-success-bg); }
.tone-warning { color: var(--sx-warning); background: var(--sx-warning-bg); }
.tone-danger { color: var(--sx-danger); background: var(--sx-danger-bg); }
.tone-info { color: var(--sx-accent); background: var(--sx-accent-bg); }
</style>
