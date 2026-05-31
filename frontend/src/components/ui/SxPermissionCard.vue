<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon } from './icons'

const props = withDefaults(defineProps<{
  title: string
  description?: string
  risk?: 'safe' | 'sudo' | 'security' | 'danger' | string
}>(), {
  description: '',
  risk: 'safe',
})

const icon = computed(() => {
  if (props.risk === 'security' || props.risk === 'danger') return resolveSxIcon('warn')
  if (props.risk === 'sudo') return resolveSxIcon('key')
  return resolveSxIcon('permission')
})
</script>

<template>
  <section :class="['sx-permission-card', `risk-${risk}`]">
    <span class="sx-permission-icon" aria-hidden="true">
      <component :is="icon" :size="18" stroke-width="2" />
    </span>
    <div class="sx-permission-main">
      <strong>{{ title }}</strong>
      <p v-if="description">{{ description }}</p>
      <slot />
    </div>
    <div v-if="$slots.actions" class="sx-permission-actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<style scoped>
.sx-permission-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  padding: 12px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-10);
  background: var(--sx-control-bg);
}

.sx-permission-icon {
  width: 32px;
  height: 32px;
  border-radius: var(--sx-radius-8);
  background: var(--sx-accent-bg);
  color: var(--sx-accent);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.risk-sudo .sx-permission-icon {
  background: var(--sx-warning-bg);
  color: var(--sx-warning);
}

.risk-security .sx-permission-icon,
.risk-danger .sx-permission-icon {
  background: var(--sx-danger-bg);
  color: var(--sx-danger);
}

.sx-permission-main {
  min-width: 0;
}

.sx-permission-main strong {
  display: block;
  color: var(--sx-text-primary);
  font: 600 13px/18px var(--sx-font-sans);
}

.sx-permission-main p {
  margin-top: 3px;
  color: var(--sx-text-secondary);
  font: 400 12px/18px var(--sx-font-sans);
}

.sx-permission-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 680px) {
  .sx-permission-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .sx-permission-actions {
    grid-column: 1 / -1;
    justify-content: flex-end;
  }
}
</style>
