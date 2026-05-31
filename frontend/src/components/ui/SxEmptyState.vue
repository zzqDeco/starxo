<script setup lang="ts">
import { computed } from 'vue'
import { resolveSxIcon, type SxIconName } from './icons'

const props = withDefaults(defineProps<{
  icon?: SxIconName | string
  title: string
  description?: string
  align?: 'left' | 'center'
}>(), {
  icon: 'info',
  description: '',
  align: 'center',
})

const iconComponent = computed(() => resolveSxIcon(props.icon))
</script>

<template>
  <section :class="['sx-empty-state', `align-${align}`]">
    <span class="sx-empty-icon" aria-hidden="true">
      <component :is="iconComponent" :size="20" stroke-width="2" />
    </span>
    <div class="sx-empty-copy">
      <strong>{{ title }}</strong>
      <p v-if="description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="sx-empty-actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<style scoped>
.sx-empty-state {
  min-height: 160px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  color: var(--sx-text-tertiary);
}

.align-center {
  align-items: center;
  text-align: center;
}

.align-left {
  align-items: flex-start;
  text-align: left;
}

.sx-empty-icon {
  width: 34px;
  height: 34px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-10);
  background: var(--sx-control-hover);
  color: var(--sx-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.sx-empty-copy strong {
  display: block;
  color: var(--sx-text-primary);
  font: 600 13px/18px var(--sx-font-sans);
}

.sx-empty-copy p {
  margin-top: 3px;
  max-width: 360px;
  color: var(--sx-text-tertiary);
  font: 400 12px/18px var(--sx-font-sans);
}

.sx-empty-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: inherit;
  gap: 8px;
  margin-top: 2px;
}
</style>
