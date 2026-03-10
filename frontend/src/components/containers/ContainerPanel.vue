<script lang="ts" setup>
import { computed, onMounted } from 'vue'
import { NButton, NIcon, NEmpty, NSpin, NCollapse, NCollapseItem, NTag, NPopconfirm, NProgress } from 'naive-ui'
import { Refresh, Play, Stop, Trash, Server, Add, RadioButtonOn, RadioButtonOff } from '@vicons/ionicons5'
import { useContainerStore } from '@/stores/containerStore'
import { useConnectionStore } from '@/stores/connectionStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useI18n } from 'vue-i18n'
import { useUiFeedback } from '@/composables/useUiFeedback'
import type { ContainerInfo } from '@/types/session'

const { t } = useI18n()
const containerStore = useContainerStore()
const connectionStore = useConnectionStore()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()
const panelBusy = computed(() => containerStore.loading || containerStore.creatingContainer)

onMounted(() => {
  containerStore.loadContainers().catch((e) => feedback.error(t('containers.title'), e))
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
  return containerStore.activeContainerID === container.id
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



async function createContainer() {
  try {
    await containerStore.createContainer()
    feedback.success(t('feedback.containerCreated'))
  } catch (e) {
    feedback.error(t('feedback.actions.createContainer'), e)
  }
}

async function refreshContainers() {
  try {
    await containerStore.loadContainers()
  } catch (e) {
    feedback.error(t('containers.title'), e)
  }
}

async function startContainer(id: string) {
  try {
    await containerStore.startContainer(id)
    feedback.success(t('feedback.containerStarted'))
  } catch (e) {
    feedback.error(t('feedback.actions.startContainer'), e)
  }
}

async function stopContainer(id: string) {
  try {
    await containerStore.stopContainer(id)
    feedback.success(t('feedback.containerStopped'))
  } catch (e) {
    feedback.error(t('feedback.actions.stopContainer'), e)
  }
}

async function destroyContainer(id: string) {
  try {
    await containerStore.destroyContainer(id)
    feedback.success(t('feedback.containerDestroyed'))
  } catch (e) {
    feedback.error(t('feedback.actions.destroyContainer'), e)
  }
}

async function activateContainer(id: string) {
  try {
    await containerStore.activateContainer(id)
    feedback.success(t('feedback.containerActivated'))
  } catch (e) {
    feedback.error(t('feedback.actions.activateContainer'), e)
  }
}

async function deactivateContainer() {
  try {
    await containerStore.deactivateContainer()
    feedback.success(t('feedback.containerDeactivated'))
  } catch (e) {
    feedback.error(t('feedback.actions.deactivateContainer'), e)
  }
}

async function refreshStatus(id: string) {
  try {
    await containerStore.refreshStatus(id)
  } catch (e) {
    feedback.error(t('containers.title'), e)
  }
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
      <div class="header-actions">
        <NButton
          size="tiny"
          type="primary"
          :disabled="!connectionStore.sshConnected || panelBusy"
          :loading="containerStore.creatingContainer"
          @click="createContainer"
        >
          <template #icon><NIcon size="14"><Add /></NIcon></template>
          {{ t('containers.createContainer') }}
        </NButton>
        <NButton quaternary circle size="tiny" @click="refreshContainers" :loading="containerStore.loading" :disabled="panelBusy">
          <template #icon><NIcon size="14"><Refresh /></NIcon></template>
        </NButton>
      </div>
    </div>

    <!-- Container creation progress -->
    <div v-if="containerStore.creatingContainer" class="creation-progress">
      <NProgress :percentage="containerStore.containerProgress" :show-indicator="false" type="line" status="info" />
      <span class="progress-step">{{ containerStore.containerStep || t('containers.creating') }}</span>
    </div>

    <NSpin :show="containerStore.loading" class="panel-body">
      <!-- Current Session Containers -->
      <div class="section">
        <div class="section-label">{{ t('containers.currentSession') }} ({{ containerStore.activeSessionContainers.length }})</div>

        <div v-if="containerStore.activeSessionContainers.length === 0" class="empty-hint">
          <template v-if="connectionStore.sshConnected">
            {{ t('containers.sshReadyHint') }}
          </template>
          <template v-else>
            {{ t('containers.sshRequired') }}
          </template>
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
            <!-- Activate / Deactivate -->
            <NButton
              v-if="!isActive(c) && c.status === 'running' && connectionStore.sshConnected"
              quaternary size="tiny" type="info"
              @click="activateContainer(c.id)" :loading="containerStore.isActionPending(`activate:${c.id}`)" :disabled="panelBusy"
            >
              <template #icon><NIcon size="14"><RadioButtonOn /></NIcon></template>
              {{ t('containers.activate') }}
            </NButton>
            <NButton
              v-if="isActive(c)"
              quaternary size="tiny"
              @click="deactivateContainer()" :loading="containerStore.isActionPending('deactivate')" :disabled="panelBusy"
            >
              <template #icon><NIcon size="14"><RadioButtonOff /></NIcon></template>
              {{ t('containers.deactivate') }}
            </NButton>
            <NButton v-if="c.status === 'stopped'" quaternary size="tiny" type="success" @click="startContainer(c.id)" :loading="containerStore.isActionPending(`start:${c.id}`)" :disabled="panelBusy">
              <template #icon><NIcon size="14"><Play /></NIcon></template>
              {{ t('containers.start') }}
            </NButton>
            <NButton v-if="c.status === 'running' && !isActive(c)" quaternary size="tiny" type="warning" @click="stopContainer(c.id)" :loading="containerStore.isActionPending(`stop:${c.id}`)" :disabled="panelBusy">
              <template #icon><NIcon size="14"><Stop /></NIcon></template>
              {{ t('containers.stop') }}
            </NButton>
            <NButton quaternary size="tiny" @click="refreshStatus(c.id)" :loading="containerStore.isActionPending(`refresh:${c.id}`)" :disabled="panelBusy">
              <template #icon><NIcon size="14"><Refresh /></NIcon></template>
            </NButton>
            <NPopconfirm @positive-click="destroyContainer(c.id)">
              <template #trigger>
                <NButton quaternary size="tiny" type="error" :loading="containerStore.isActionPending(`destroy:${c.id}`)" :disabled="panelBusy">
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
                <NButton
                  v-if="c.status === 'running' && connectionStore.sshConnected"
                  quaternary size="tiny" type="info"
                  @click="activateContainer(c.id)" :loading="containerStore.isActionPending(`activate:${c.id}`)" :disabled="panelBusy"
                >
                  <template #icon><NIcon size="14"><RadioButtonOn /></NIcon></template>
                  {{ t('containers.activate') }}
                </NButton>
                <NButton v-if="c.status === 'stopped'" quaternary size="tiny" type="success" @click="startContainer(c.id)" :loading="containerStore.isActionPending(`start:${c.id}`)" :disabled="panelBusy">
                  <template #icon><NIcon size="14"><Play /></NIcon></template>
                  {{ t('containers.start') }}
                </NButton>
                <NButton v-if="c.status === 'running'" quaternary size="tiny" type="warning" @click="stopContainer(c.id)" :loading="containerStore.isActionPending(`stop:${c.id}`)" :disabled="panelBusy">
                  <template #icon><NIcon size="14"><Stop /></NIcon></template>
                  {{ t('containers.stop') }}
                </NButton>
                <NButton quaternary size="tiny" @click="refreshStatus(c.id)" :loading="containerStore.isActionPending(`refresh:${c.id}`)" :disabled="panelBusy">
                  <template #icon><NIcon size="14"><Refresh /></NIcon></template>
                </NButton>
                <NPopconfirm @positive-click="destroyContainer(c.id)">
                  <template #trigger>
                    <NButton quaternary size="tiny" type="error" :loading="containerStore.isActionPending(`destroy:${c.id}`)" :disabled="panelBusy">
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.panel-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.creation-progress {
  padding: 8px 14px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.progress-step {
  display: block;
  font-size: 11px;
  color: var(--accent-amber);
  font-style: italic;
  margin-top: 4px;
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
