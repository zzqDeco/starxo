<script lang="ts" setup>
import { onMounted } from 'vue'
import { NButton, NIcon, NEmpty, NSpin, NCollapse, NCollapseItem, NTag, NPopconfirm } from 'naive-ui'
import { Refresh, Play, Stop, Trash, Server } from '@vicons/ionicons5'
import { useContainerStore } from '@/stores/containerStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useI18n } from 'vue-i18n'
import type { ContainerInfo } from '@/types/session'

const { t } = useI18n()
const containerStore = useContainerStore()
const sessionStore = useSessionStore()

onMounted(() => {
  containerStore.loadContainers()
})

function statusType(status: string): 'success' | 'warning' | 'error' | 'default' {
  switch (status) {
    case 'running': return 'success'
    case 'stopped': return 'warning'
    case 'destroyed': return 'error'
    default: return 'default'
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case 'running': return t('containers.running')
    case 'stopped': return t('containers.stopped')
    case 'destroyed': return t('containers.destroyed')
    default: return t('containers.unknown')
  }
}

function isActive(container: ContainerInfo): boolean {
  const session = sessionStore.activeSession
  return session?.activeContainerID === container.id
}

function formatTime(ts: number): string {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function sessionTitle(sessionID: string): string {
  const sess = sessionStore.sessions.find(s => s.id === sessionID)
  return sess?.title || sessionID.substring(0, 8)
}
</script>

<template>
  <div class="container-panel">
    <!-- Header -->
    <div class="panel-header">
      <span class="panel-title">{{ t('containers.title') }}</span>
      <NButton quaternary circle size="tiny" @click="containerStore.loadContainers()" :loading="containerStore.loading">
        <template #icon><NIcon size="14"><Refresh /></NIcon></template>
      </NButton>
    </div>

    <NSpin :show="containerStore.loading" class="panel-body">
      <!-- Current Session Containers -->
      <div class="section">
        <div class="section-label">{{ t('containers.currentSession') }} ({{ containerStore.activeSessionContainers.length }})</div>

        <div v-if="containerStore.activeSessionContainers.length === 0" class="empty-hint">
          {{ t('containers.noContainers') }}
        </div>

        <div
          v-for="c in containerStore.activeSessionContainers"
          :key="c.id"
          :class="['container-card', { active: isActive(c) }]"
        >
          <div class="card-header">
            <NIcon size="16" class="card-icon"><Server /></NIcon>
            <span class="card-name">{{ c.name || c.id.substring(0, 8) }}</span>
            <NTag :type="statusType(c.status)" size="small" round>{{ statusLabel(c.status) }}</NTag>
            <NTag v-if="isActive(c)" type="info" size="small" round>{{ t('containers.active') }}</NTag>
          </div>
          <div class="card-details">
            <span class="detail-item">{{ c.image }}</span>
            <span class="detail-item">{{ c.sshHost }}:{{ c.sshPort }}</span>
            <span class="detail-item detail-time">{{ formatTime(c.lastUsedAt) }}</span>
          </div>
          <div class="card-actions">
            <NButton v-if="c.status === 'stopped'" quaternary size="tiny" type="success" @click="containerStore.startContainer(c.id)">
              <template #icon><NIcon size="14"><Play /></NIcon></template>
              {{ t('containers.start') }}
            </NButton>
            <NButton v-if="c.status === 'running'" quaternary size="tiny" type="warning" @click="containerStore.stopContainer(c.id)">
              <template #icon><NIcon size="14"><Stop /></NIcon></template>
              {{ t('containers.stop') }}
            </NButton>
            <NButton quaternary size="tiny" @click="containerStore.refreshStatus(c.id)">
              <template #icon><NIcon size="14"><Refresh /></NIcon></template>
            </NButton>
            <NPopconfirm @positive-click="containerStore.destroyContainer(c.id)">
              <template #trigger>
                <NButton quaternary size="tiny" type="error">
                  <template #icon><NIcon size="14"><Trash /></NIcon></template>
                  {{ t('containers.destroy') }}
                </NButton>
              </template>
              {{ t('containers.destroyConfirm') }}
            </NPopconfirm>
          </div>
        </div>
      </div>

      <!-- All Containers (collapsible) -->
      <div v-if="containerStore.otherContainers.length > 0" class="section">
        <NCollapse>
          <NCollapseItem :title="`${t('containers.allContainers')} (${containerStore.otherContainers.length})`">
            <div
              v-for="c in containerStore.otherContainers"
              :key="c.id"
              class="container-card"
            >
              <div class="card-header">
                <NIcon size="16" class="card-icon"><Server /></NIcon>
                <span class="card-name">{{ c.name || c.id.substring(0, 8) }}</span>
                <NTag :type="statusType(c.status)" size="small" round>{{ statusLabel(c.status) }}</NTag>
              </div>
              <div class="card-details">
                <span class="detail-item">{{ c.image }}</span>
                <span class="detail-item detail-session">{{ sessionTitle(c.sessionID) }}</span>
                <span class="detail-item detail-time">{{ formatTime(c.lastUsedAt) }}</span>
              </div>
              <div class="card-actions">
                <NButton v-if="c.status === 'stopped'" quaternary size="tiny" type="success" @click="containerStore.startContainer(c.id)">
                  <template #icon><NIcon size="14"><Play /></NIcon></template>
                  {{ t('containers.start') }}
                </NButton>
                <NButton v-if="c.status === 'running'" quaternary size="tiny" type="warning" @click="containerStore.stopContainer(c.id)">
                  <template #icon><NIcon size="14"><Stop /></NIcon></template>
                  {{ t('containers.stop') }}
                </NButton>
                <NButton quaternary size="tiny" @click="containerStore.refreshStatus(c.id)">
                  <template #icon><NIcon size="14"><Refresh /></NIcon></template>
                </NButton>
                <NPopconfirm @positive-click="containerStore.destroyContainer(c.id)">
                  <template #trigger>
                    <NButton quaternary size="tiny" type="error">
                      <template #icon><NIcon size="14"><Trash /></NIcon></template>
                      {{ t('containers.destroy') }}
                    </NButton>
                  </template>
                  {{ t('containers.destroyConfirm') }}
                </NPopconfirm>
              </div>
            </div>
          </NCollapseItem>
        </NCollapse>
      </div>

      <!-- Truly empty -->
      <NEmpty v-if="containerStore.containers.length === 0 && !containerStore.loading" :description="t('containers.empty')" class="empty-state" />
    </NSpin>
  </div>
</template>

<style scoped>
.container-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.panel-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
}

.section {
  margin-bottom: 16px;
}

.section-label {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 1px;
  padding: 0 4px;
  margin-bottom: 8px;
}

.empty-hint {
  font-size: 12px;
  color: var(--text-muted);
  padding: 12px 4px;
  font-style: italic;
  text-align: center;
}

.container-card {
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  margin-bottom: 8px;
  transition: border-color var(--transition-fast);
}

.container-card:hover {
  border-color: var(--accent-cyan);
}

.container-card.active {
  border-color: var(--accent-cyan);
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.06) 0%, var(--bg-deepest) 100%);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.card-icon {
  color: var(--accent-cyan);
  flex-shrink: 0;
}

.card-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-details {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.detail-item {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.detail-session {
  color: var(--accent-cyan);
}

.detail-time {
  color: var(--text-faint);
}

.card-actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.empty-state {
  margin-top: 40px;
}
</style>
