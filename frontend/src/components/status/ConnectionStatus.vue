<script lang="ts" setup>
import { NTooltip } from 'naive-ui'
import { useConnectionStore } from '@/stores/connectionStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const connectionStore = useConnectionStore()
</script>

<template>
  <div class="connection-status">
    <NTooltip trigger="hover" placement="bottom">
      <template #trigger>
        <div class="status-pill">
          <span
            :class="['dot', connectionStore.sshConnected ? 'dot-green' : 'dot-red']"
          ></span>
          <span class="pill-label">{{ t('status.ssh') }}</span>
        </div>
      </template>
      {{ connectionStore.sshConnected ? t('status.sshConnected') : t('status.sshDisconnected') }}
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
  gap: 5px;
  padding: 3px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 20px;
  cursor: default;
  transition: background var(--transition-fast);
}

.status-pill:hover {
  background: var(--bg-hover);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-green {
  background: var(--accent-emerald);
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}

.dot-red {
  background: var(--accent-rose);
  box-shadow: 0 0 6px rgba(244, 63, 94, 0.3);
}

.pill-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  letter-spacing: 0.3px;
}

.connecting-text {
  font-size: 11px;
  color: var(--accent-amber);
  animation: pulse 1.5s ease-in-out infinite;
  white-space: nowrap;
}
</style>
