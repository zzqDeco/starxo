<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  description?: string
  state?: 'default' | 'error' | 'disabled'
}>(), {
  description: '',
  state: 'default',
})
</script>

<template>
  <div :class="['sx-settings-row', `state-${state}`]">
    <div class="sx-settings-copy">
      <label>{{ label }}</label>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="sx-settings-control">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.sx-settings-row {
  display: grid;
  grid-template-columns: minmax(160px, 240px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
  padding: 14px 0;
  border-bottom: 1px solid var(--sx-separator);
}

.sx-settings-copy {
  min-width: 0;
}

.sx-settings-copy label {
  display: block;
  color: var(--sx-text-primary);
  font: 500 13px/18px var(--sx-font-sans);
}

.sx-settings-copy p {
  margin-top: 3px;
  color: var(--sx-text-tertiary);
  font: 400 12px/17px var(--sx-font-sans);
}

.sx-settings-control {
  min-width: 0;
}

.state-disabled {
  opacity: 0.56;
}

.state-error .sx-settings-copy label {
  color: var(--sx-danger);
}

@media (max-width: 760px) {
  .sx-settings-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }
}
</style>
