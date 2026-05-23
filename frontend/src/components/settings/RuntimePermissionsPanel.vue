<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { NButton, NEmpty, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/sessionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import {
  ClearToolPermissionGrants,
  ListToolPermissionGrants,
  ListToolPermissionRequests,
  RevokeToolPermissionGrant,
} from '../../../wailsjs/go/service/ChatService'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { model, tools } from '../../../wailsjs/go/models'

const { t } = useI18n()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()

const grants = ref<model.RuntimePermissionGrant[]>([])
const pending = ref<tools.ToolPermissionRequest[]>([])
const loading = ref(false)
const revokingTool = ref('')
const clearing = ref(false)
let eventCleanups: Array<() => void> = []

const activeSessionId = computed(() => sessionStore.activeSessionId || '')

function riskLabel(risk?: string) {
  const labels: Record<string, string> = {
    destructive: t('permissions.riskDestructive'),
    execute: t('permissions.riskExecute'),
    write: t('permissions.riskWrite'),
    read: t('permissions.riskRead'),
  }
  return labels[risk || ''] || t('permissions.riskStandard')
}

function sourceLabel(source?: string) {
  if (!source || source === 'runtime') return t('permissions.sourceRuntime')
  if (source === 'mcp') return t('permissions.sourceMcp')
  return source
}

function toolClassLabel(toolClass?: string) {
  const labels: Record<string, string> = {
    read: t('permissions.classRead'),
    write: t('permissions.classWrite'),
    execute: t('permissions.classExecute'),
    destructive: t('permissions.classDestructive'),
    task: t('permissions.classTask'),
    tool: t('permissions.classTool'),
  }
  return labels[toolClass || ''] || t('permissions.classTool')
}

function formatTime(ms?: number) {
  if (!ms) return ''
  return new Date(ms).toLocaleString()
}

async function refreshPermissions() {
  const sessionID = activeSessionId.value
  if (!sessionID) {
    grants.value = []
    pending.value = []
    loading.value = false
    return
  }
  loading.value = true
  try {
    const [nextGrants, nextPending] = await Promise.all([
      ListToolPermissionGrants(sessionID),
      ListToolPermissionRequests(sessionID),
    ])
    if (sessionID !== activeSessionId.value) return
    grants.value = (nextGrants || []) as model.RuntimePermissionGrant[]
    pending.value = (nextPending || []) as tools.ToolPermissionRequest[]
  } catch (e) {
    if (sessionID !== activeSessionId.value) return
    feedback.error(t('permissions.refresh'), e)
  } finally {
    if (sessionID === activeSessionId.value) {
      loading.value = false
    }
  }
}

async function revokeGrant(toolName?: string) {
  if (!activeSessionId.value || !toolName) return
  revokingTool.value = toolName
  try {
    await RevokeToolPermissionGrant(activeSessionId.value, toolName)
    await refreshPermissions()
  } catch (e) {
    feedback.error(t('permissions.revoke'), e)
  } finally {
    revokingTool.value = ''
  }
}

async function clearGrants() {
  if (!activeSessionId.value) return
  clearing.value = true
  try {
    await ClearToolPermissionGrants(activeSessionId.value)
    await refreshPermissions()
  } catch (e) {
    feedback.error(t('permissions.clear'), e)
  } finally {
    clearing.value = false
  }
}

watch(activeSessionId, refreshPermissions)

onMounted(() => {
  refreshPermissions()
  eventCleanups = [
    EventsOn('runtime:permission_request', refreshPermissions),
    EventsOn('runtime:permission_canceled', refreshPermissions),
    EventsOn('runtime:permission_resolved', refreshPermissions),
    EventsOn('runtime:permission_grants_changed', refreshPermissions),
  ]
})

onUnmounted(() => {
  eventCleanups.forEach((cleanup) => cleanup())
  eventCleanups = []
})
</script>

<template>
  <div class="permissions-panel">
    <section class="permissions-section">
      <div class="section-head">
        <div>
          <h3>{{ t('permissions.pendingTitle') }}</h3>
          <p>{{ t('permissions.pendingSubtitle') }}</p>
        </div>
        <NButton size="small" :loading="loading" @click="refreshPermissions">
          {{ t('permissions.refresh') }}
        </NButton>
      </div>

      <NEmpty v-if="!activeSessionId" :description="t('permissions.noSession')" />
      <NEmpty v-else-if="pending.length === 0" :description="t('permissions.noPending')" />
      <div v-else class="permission-list">
        <article v-for="request in pending" :key="request.requestId" class="permission-row">
          <div class="row-main">
            <strong>{{ request.title || request.toolName }}</strong>
            <span>{{ request.toolName }}</span>
          </div>
          <div class="row-meta">
            <span :class="['risk-badge', request.risk || 'standard']">{{ riskLabel(request.risk) }}</span>
            <span>{{ sourceLabel(request.source) }}</span>
            <span>{{ formatTime(request.createdAt) }}</span>
          </div>
        </article>
      </div>
    </section>

    <section class="permissions-section">
      <div class="section-head">
        <div>
          <h3>{{ t('permissions.grantsTitle') }}</h3>
          <p>{{ t('permissions.grantsSubtitle') }}</p>
        </div>
        <NPopconfirm :disabled="grants.length === 0" @positive-click="clearGrants">
          <template #trigger>
            <NButton size="small" :disabled="grants.length === 0" :loading="clearing">
              {{ t('permissions.clear') }}
            </NButton>
          </template>
          {{ t('permissions.clearConfirm') }}
        </NPopconfirm>
      </div>

      <NEmpty v-if="!activeSessionId" :description="t('permissions.noSession')" />
      <NEmpty v-else-if="grants.length === 0" :description="t('permissions.noGrants')" />
      <div v-else class="permission-list">
        <article v-for="grant in grants" :key="grant.toolName" class="permission-row">
          <div class="row-main">
            <strong>{{ grant.toolName }}</strong>
            <span>{{ sourceLabel(grant.source) }} / {{ toolClassLabel(grant.toolClass) }}</span>
          </div>
          <div class="row-actions">
            <span class="grant-time">{{ formatTime(grant.createdAt) }}</span>
            <NButton
              size="tiny"
              tertiary
              :loading="revokingTool === grant.toolName"
              @click="revokeGrant(grant.toolName)"
            >
              {{ t('permissions.revoke') }}
            </NButton>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.permissions-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.permissions-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-md);
}

.section-head h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: var(--fs-md);
}

.section-head p {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: 1.45;
}

.permission-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.permission-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-md);
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--platform-bg-window) 72%, transparent);
}

.row-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.row-main strong {
  color: var(--text-primary);
  font-size: var(--fs-sm);
}

.row-main span,
.row-meta,
.grant-time {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}

.row-meta,
.row-actions {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  flex-shrink: 0;
}

.risk-badge {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 2px 7px;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--border-subtle) 78%, transparent);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 74%, transparent);
  color: var(--text-muted);
  font-family: var(--font-sans);
  font-size: 10.5px;
  font-weight: var(--fw-medium);
  line-height: 1.3;
}

.risk-badge.write,
.risk-badge.execute {
  color: color-mix(in srgb, var(--platform-accent) 64%, var(--text-muted));
  border-color: color-mix(in srgb, var(--platform-accent) 16%, transparent);
  background: color-mix(in srgb, var(--platform-accent) 7%, transparent);
}

.risk-badge.destructive {
  color: var(--platform-danger);
  border-color: color-mix(in srgb, var(--platform-danger) 16%, transparent);
  background: color-mix(in srgb, var(--platform-danger) 7%, transparent);
}

@media (max-width: 720px) {
  .permission-row,
  .section-head {
    align-items: stretch;
    flex-direction: column;
  }

  .row-meta,
  .row-actions {
    justify-content: space-between;
  }
}
</style>
