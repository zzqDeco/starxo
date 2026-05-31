<script setup lang="ts">
withDefaults(defineProps<{
  status?: 'running' | 'success' | 'warning' | 'error'
  tool: string
  description?: string
  meta?: string
}>(), {
  status: 'success',
  description: '',
  meta: '',
})
</script>

<template>
  <div class="sx-timeline-item" :class="`status-${status}`">
    <span class="sx-timeline-item__dot" aria-hidden="true"></span>
    <span class="sx-timeline-item__tool">{{ tool }}</span>
    <span class="sx-timeline-item__description">{{ description }}</span>
    <span v-if="meta" class="sx-timeline-item__meta">{{ meta }}</span>
  </div>
</template>

<style scoped>
.sx-timeline-item {
  display: grid;
  grid-template-columns: 10px 82px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid var(--sx-separator);
  border-radius: var(--sx-radius-10);
  background: var(--sx-inspector-bg);
}
.sx-timeline-item__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--sx-success);
}
.status-running .sx-timeline-item__dot,
.status-warning .sx-timeline-item__dot { background: var(--sx-warning); }
.status-error .sx-timeline-item__dot { background: var(--sx-danger); }
.sx-timeline-item__tool {
  font: 600 11px/15px var(--sx-font-sans);
  color: var(--sx-text-primary);
}
.sx-timeline-item__description {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--sx-text-secondary);
  font: 400 11px/15px var(--sx-font-sans);
}
.sx-timeline-item__meta {
  color: var(--sx-text-tertiary);
  font: 500 10px/13px var(--sx-font-sans);
}
</style>
