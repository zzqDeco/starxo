<script setup lang="ts">
withDefaults(defineProps<{
  text: string
  type?: 'command' | 'stdout' | 'stderr' | 'info'
  time?: string
}>(), {
  type: 'stdout',
  time: '',
})
</script>

<template>
  <div :class="['sx-terminal-row', `type-${type}`]">
    <span v-if="time" class="sx-terminal-time">{{ time }}</span>
    <span class="sx-terminal-prefix">{{ type === 'command' ? '$' : '' }}</span>
    <span class="sx-terminal-text">{{ text }}</span>
  </div>
</template>

<style scoped>
.sx-terminal-row {
  display: grid;
  grid-template-columns: auto 12px minmax(0, 1fr);
  gap: 7px;
  padding: 2px 6px;
  border-radius: var(--sx-radius-6);
  font: 400 12px/1.48 var(--sx-font-mono);
  color: var(--sx-text-secondary);
}

.sx-terminal-time {
  color: var(--sx-text-tertiary);
  font-size: 11px;
}

.sx-terminal-prefix {
  color: var(--sx-text-tertiary);
}

.sx-terminal-text {
  white-space: pre-wrap;
  word-break: break-word;
}

.type-command {
  color: var(--sx-text-primary);
  background: var(--sx-control-hover);
}

.type-stderr {
  color: var(--sx-danger);
  background: var(--sx-danger-bg);
}

.type-info {
  color: var(--sx-accent);
}
</style>
