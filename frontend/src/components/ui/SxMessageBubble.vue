<script setup lang="ts">
withDefaults(defineProps<{
  role?: 'user' | 'assistant' | 'system'
  tone?: 'default' | 'running' | 'error' | 'success'
  title?: string
  meta?: string
}>(), {
  role: 'assistant',
  tone: 'default',
  title: '',
  meta: '',
})
</script>

<template>
  <article :class="['sx-message-bubble', `role-${role}`, `tone-${tone}`]">
    <header v-if="title || meta" class="sx-message-head">
      <strong v-if="title">{{ title }}</strong>
      <span v-if="meta">{{ meta }}</span>
    </header>
    <div class="sx-message-body">
      <slot />
    </div>
  </article>
</template>

<style scoped>
.sx-message-bubble {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--sx-radius-10);
  color: var(--sx-text-primary);
}

.role-user {
  margin-left: auto;
  max-width: min(680px, 86%);
  background: var(--sx-control-selected);
  border-color: var(--sx-separator);
}

.role-assistant,
.role-system {
  background: transparent;
}

.tone-error {
  background: var(--sx-danger-bg);
  border-color: var(--sx-danger-border);
}

.tone-success {
  background: var(--sx-success-bg);
  border-color: color-mix(in srgb, var(--sx-success) 20%, transparent);
}

.tone-running {
  background: var(--sx-accent-bg);
  border-color: var(--sx-accent-border);
}

.sx-message-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px 0;
  color: var(--sx-text-secondary);
  font: 500 12px/16px var(--sx-font-sans);
}

.sx-message-head span {
  color: var(--sx-text-tertiary);
  font-weight: 400;
}

.sx-message-body {
  padding: 10px 12px;
  font: 400 13px/1.58 var(--sx-font-sans);
}
</style>
