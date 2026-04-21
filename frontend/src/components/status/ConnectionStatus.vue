<script lang="ts" setup>
import { computed } from 'vue'
import { NTooltip } from 'naive-ui'
import { useConnectionStore } from '@/stores/connectionStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const connectionStore = useConnectionStore()

type PillState = 'connected' | 'connecting' | 'error' | 'disconnected'

const pillState = computed<PillState>(() => {
  if (connectionStore.error) return 'error'
  if (connectionStore.connecting) return 'connecting'
  if (connectionStore.sshConnected) return 'connected'
  return 'disconnected'
})

const pillTooltip = computed(() => {
  switch (pillState.value) {
    case 'connected': return t('status.sshConnected')
    case 'connecting': return connectionStore.initStep || t('status.sshConnecting') || t('status.sshDisconnected')
    case 'error': return connectionStore.error || t('status.sshDisconnected')
    default: return t('status.sshDisconnected')
  }
})
</script>

<template>
  <div class="connection-status" role="status" aria-live="polite">
    <NTooltip trigger="hover" placement="bottom">
      <template #trigger>
        <div :class="['status-pill', `pill-${pillState}`]">
          <span :class="['dot', `dot-${pillState}`]" aria-hidden="true"></span>
          <span class="pill-label">SSH</span>
        </div>
      </template>
      {{ pillTooltip }}
    </NTooltip>

    <span v-if="connectionStore.connecting" class="connecting-text">
      {{ connectionStore.initStep }}
    </span>
  </div>
</template>

<style scoped>
.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-pill);
  cursor: default;
  transition: background var(--transition-ui), border-color var(--transition-ui);
}

.status-pill:hover {
  background: var(--bg-hover);
}

.pill-connected { border-color: rgba(16, 185, 129, 0.35); }
.pill-connecting { border-color: rgba(245, 158, 11, 0.35); }
.pill-error { border-color: rgba(244, 63, 94, 0.35); }

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-connected {
  background: var(--accent-emerald);
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}

.dot-connecting {
  background: var(--accent-amber);
  box-shadow: 0 0 6px rgba(245, 158, 11, 0.45);
  animation: pulse 1.5s ease-in-out infinite;
}

.dot-error {
  background: var(--accent-rose);
  box-shadow: 0 0 6px rgba(244, 63, 94, 0.4);
}

.dot-disconnected {
  background: var(--text-muted);
}

.pill-label {
  font-family: var(--font-brand);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  color: var(--text-secondary);
  letter-spacing: 0.6px;
}

.connecting-text {
  font-size: var(--fs-2xs);
  color: var(--accent-amber);
  animation: pulse 1.5s ease-in-out infinite;
  white-space: nowrap;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
