<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { NButton, NIcon, NEmpty, NSpin, NCollapse, NCollapseItem, NTag, NProgress, NInput } from 'naive-ui'
// Note: NPopconfirm replaced by type-to-confirm destroy dialog below
import { Refresh, Play, Stop, Trash, Server, Add, RadioButtonOn, RadioButtonOff, Close } from '@vicons/ionicons5'
import { useFocusTrap } from '@/composables/useFocusTrap'
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

const destroyTarget = ref<ContainerInfo | null>(null)
const destroyInput = ref('')
const destroyDialogRef = ref<HTMLElement | null>(null)
const destroyActive = computed(() => !!destroyTarget.value)
useFocusTrap(destroyDialogRef, destroyActive)

function openDestroy(container: ContainerInfo) {
  destroyTarget.value = container
  destroyInput.value = ''
}

function closeDestroy() {
  destroyTarget.value = null
  destroyInput.value = ''
}

const destroyName = computed(() => {
  const c = destroyTarget.value
  if (!c) return ''
  return c.name || c.id.substring(0, 12)
})

const destroyMatches = computed(() => destroyInput.value.trim() === destroyName.value)

async function confirmDestroy() {
  if (!destroyTarget.value || !destroyMatches.value) return
  const id = destroyTarget.value.id
  closeDestroy()
  await destroyContainer(id)
}

function onDestroyKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    closeDestroy()
  } else if (e.key === 'Enter' && destroyMatches.value) {
    e.preventDefault()
    confirmDestroy()
  }
}

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
          :class="['container-card', `status-${c.status}`, { active: isActive(c) }]"
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
            <NButton
              quaternary
              size="tiny"
              type="error"
              :loading="containerStore.isActionPending(`destroy:${c.id}`)"
              :disabled="panelBusy"
              @click="openDestroy(c)"
            >
              <template #icon><NIcon size="14"><Trash /></NIcon></template>
              {{ t('containers.destroy') }}
            </NButton>
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
              :class="['container-card', `status-${c.status}`]"
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

    <!-- Destroy confirmation with type-to-confirm -->
    <Teleport to="body">
      <Transition name="destroy-modal">
        <div
          v-if="destroyTarget"
          class="destroy-overlay"
          @mousedown.self="closeDestroy"
        >
          <div
            ref="destroyDialogRef"
            class="destroy-dialog"
            role="alertdialog"
            aria-modal="true"
            :aria-label="t('containers.destroyConfirmTitle')"
            tabindex="-1"
            @keydown="onDestroyKeydown"
          >
            <header class="destroy-header">
              <span class="destroy-icon"><NIcon size="18"><Trash /></NIcon></span>
              <h3 class="destroy-heading">{{ t('containers.destroyConfirmTitle') }}</h3>
              <NButton quaternary circle size="tiny" :aria-label="t('common.cancel')" @click="closeDestroy">
                <template #icon><NIcon size="14"><Close /></NIcon></template>
              </NButton>
            </header>
            <div class="destroy-body">
              <p class="destroy-warning">{{ t('containers.destroyWarning') }}</p>
              <p class="destroy-prompt">
                {{ t('containers.destroyTypeToConfirm') }}
                <code class="destroy-name">{{ destroyName }}</code>
              </p>
              <NInput
                v-model:value="destroyInput"
                :placeholder="destroyName"
                autofocus
                size="small"
                class="destroy-input"
              />
            </div>
            <footer class="destroy-footer">
              <NButton size="small" @click="closeDestroy">{{ t('common.cancel') }}</NButton>
              <NButton
                size="small"
                type="error"
                :disabled="!destroyMatches"
                @click="confirmDestroy"
              >
                {{ t('containers.destroy') }}
              </NButton>
            </footer>
          </div>
        </div>
      </Transition>
    </Teleport>
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
  position: relative;
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px 12px 10px 15px;
  margin-bottom: 8px;
  transition: border-color var(--transition-ui), transform var(--transition-ui);
}

.container-card::before {
  content: "";
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 3px;
  border-radius: 2px;
  background: var(--text-faint);
  transition: background var(--transition-ui);
}

.container-card.status-running::before { background: var(--accent-emerald); }
.container-card.status-stopped::before { background: var(--accent-amber); }
.container-card.status-destroyed::before { background: var(--accent-rose); }

.container-card:hover {
  border-color: var(--accent-cyan-dim);
  transform: translateY(-1px);
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

/* Destroy confirmation dialog — lives at body level via Teleport */
</style>

<style>
.destroy-overlay {
  position: fixed;
  inset: 0;
  z-index: 2100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(2px);
  padding: var(--space-lg);
  animation: destroyFadeIn 160ms var(--ease-out);
}

@keyframes destroyFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.destroy-dialog {
  width: min(440px, 92vw);
  background: var(--bg-surface);
  border: 1px solid var(--accent-rose);
  border-radius: var(--radius-lg);
  box-shadow: var(--elev-3);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: destroyZoomIn 180ms var(--ease-out);
}

@keyframes destroyZoomIn {
  from { opacity: 0; transform: scale(0.94); }
  to { opacity: 1; transform: scale(1); }
}

.destroy-dialog:focus {
  outline: none;
}

.destroy-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-md) var(--space-lg);
  border-bottom: 1px solid var(--border-subtle);
}

.destroy-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: rgba(244, 63, 94, 0.12);
  color: var(--accent-rose);
  flex-shrink: 0;
}

.destroy-heading {
  margin: 0;
  flex: 1;
  font-family: var(--font-brand);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
  letter-spacing: 0.4px;
  text-transform: uppercase;
}

.destroy-body {
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.destroy-warning {
  margin: 0;
  font-size: var(--fs-sm);
  color: var(--text-secondary);
  line-height: var(--lh-normal);
}

.destroy-prompt {
  margin: 0;
  font-size: var(--fs-xs);
  color: var(--text-muted);
}

.destroy-name {
  display: inline-block;
  margin: 0 2px;
  padding: 2px 6px;
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--accent-cyan);
}

.destroy-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-sm);
  padding: var(--space-md) var(--space-lg);
  background: var(--bg-elevated);
  border-top: 1px solid var(--border-subtle);
}

.destroy-modal-enter-active,
.destroy-modal-leave-active {
  transition: opacity 160ms var(--ease-out);
}

.destroy-modal-enter-from,
.destroy-modal-leave-to {
  opacity: 0;
}
</style>
