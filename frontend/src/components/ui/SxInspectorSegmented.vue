<script setup lang="ts">
import { resolveSxIcon } from './icons'

const props = withDefaults(defineProps<{
  active: 'runtime' | 'workspace' | 'terminal' | 'tasks'
}>(), {
  active: 'runtime',
})

const emit = defineEmits<{ (e: 'update:active', value: 'runtime' | 'workspace' | 'terminal' | 'tasks'): void }>()

const items = [
  { value: 'runtime', label: '运行时', icon: resolveSxIcon('runtime') },
  { value: 'workspace', label: '工作区', icon: resolveSxIcon('workspace') },
  { value: 'terminal', label: '终端', icon: resolveSxIcon('terminal') },
  { value: 'tasks', label: '任务', icon: resolveSxIcon('tasks') },
] as const
</script>

<template>
  <div class="sx-inspector-segmented" role="tablist" aria-label="Inspector">
    <button
      v-for="item in items"
      :key="item.value"
      type="button"
      :class="{ active: props.active === item.value }"
      :aria-selected="props.active === item.value"
      @click="emit('update:active', item.value)"
    >
      <component :is="item.icon" :size="13" stroke-width="2" />
      {{ item.label }}
    </button>
  </div>
</template>

<style scoped>
.sx-inspector-segmented {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 3px;
  height: 34px;
  padding: 3px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-10);
  background: var(--sx-sidebar-bg);
}
button {
  border: 0;
  border-radius: var(--sx-radius-8);
  background: transparent;
  color: var(--sx-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  font: 500 11px/14px var(--sx-font-sans);
  cursor: pointer;
}
button.active {
  background: var(--sx-control-bg);
  color: var(--sx-text-primary);
  box-shadow: 0 0 0 1px var(--sx-separator);
}
button:focus-visible {
  outline: 2px solid var(--sx-focus);
  outline-offset: 1px;
}
</style>
