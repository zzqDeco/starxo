<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  agent: string
}>()

const agentColor = computed(() => {
  const name = props.agent.toLowerCase()
  if (name.includes('coder') || name.includes('code')) return 'var(--accent-cyan)'
  if (name.includes('plan')) return 'var(--accent-blue)'
  if (name.includes('review')) return 'var(--accent-amber)'
  if (name.includes('test')) return 'var(--accent-emerald)'
  return 'var(--accent-cyan)'
})
</script>

<template>
  <div class="agent-status" v-if="agent" role="status" aria-live="polite" :aria-label="`${agent} ${t('status.isWorking')}`">
    <div class="status-bar">
      <div class="thinking-animation">
        <span class="think-dot" :style="{ background: agentColor }"></span>
        <span class="think-dot" :style="{ background: agentColor }"></span>
        <span class="think-dot" :style="{ background: agentColor }"></span>
      </div>
      <span class="agent-label" :style="{ color: agentColor }">{{ agent }}</span>
      <span class="status-text">{{ t('status.isWorking') }}</span>
    </div>
  </div>
</template>

<style scoped>
.agent-status {
  max-width: 800px;
  margin: 0 auto;
  padding: 4px 24px;
}

.status-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--bg-surface);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  animation: fadeIn 200ms ease both;
}

.thinking-animation {
  display: flex;
  gap: 3px;
  align-items: center;
}

.think-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  animation: pulse 1.4s ease-in-out infinite;
}

.think-dot:nth-child(2) {
  animation-delay: 0.2s;
}

.think-dot:nth-child(3) {
  animation-delay: 0.4s;
}

.agent-label {
  font-size: 12px;
  font-weight: 700;
  font-family: var(--font-mono);
}

.status-text {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
