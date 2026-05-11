<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NButton, NForm, NFormItem, NIcon, NInput, NInputNumber, NSwitch, NTag } from 'naive-ui'
import { Add, CodeSlash, Refresh, Trash } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settingsStore'
import { useSessionStore } from '@/stores/sessionStore'
import { useUiFeedback } from '@/composables/useUiFeedback'
import { GetRuntimeLSPStatus } from '../../../wailsjs/go/service/ChatService'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const sessionStore = useSessionStore()
const feedback = useUiFeedback()
const loadingStatus = ref(false)
const status = ref<any | null>(null)

const lsp = computed(() => {
  if (!settingsStore.settings.agent.lsp) {
    settingsStore.settings.agent.lsp = { enabled: true, requestTimeoutMs: 15000, maxResultBytes: 16384, servers: [] }
  }
  if (!settingsStore.settings.agent.lsp.servers) {
    settingsStore.settings.agent.lsp.servers = []
  }
  return settingsStore.settings.agent.lsp
})

function addServer() {
  lsp.value.servers!.push({
    language: '',
    executable: '',
    command: [],
    extensions: [],
  })
}

function removeServer(index: number) {
  lsp.value.servers!.splice(index, 1)
}

function commandText(server: any) {
  return (server.command || []).join(' ')
}

function updateCommand(server: any, value: string) {
  server.command = value.split(/\s+/).map((item) => item.trim()).filter(Boolean)
}

function extensionsText(server: any) {
  return (server.extensions || []).join(', ')
}

function updateExtensions(server: any, value: string) {
  server.extensions = value.split(',').map((item) => item.trim()).filter(Boolean)
}

async function refreshStatus() {
  loadingStatus.value = true
  try {
    status.value = await GetRuntimeLSPStatus(sessionStore.activeSessionId || '')
  } catch (e) {
    feedback.error(t('settings.lsp.status'), e)
  } finally {
    loadingStatus.value = false
  }
}
</script>

<template>
  <div class="lsp-config">
    <NForm label-placement="top" size="small" class="stacked-form">
      <div class="u-form-grid-2col">
        <NFormItem :label="t('settings.lsp.enabled')">
          <NSwitch v-model:value="lsp.enabled" />
        </NFormItem>
        <NFormItem :label="t('settings.lsp.timeout')">
          <NInputNumber v-model:value="lsp.requestTimeoutMs" :min="1000" :step="1000" class="full-width" />
        </NFormItem>
      </div>
      <NFormItem :label="t('settings.lsp.maxResultBytes')">
        <NInputNumber v-model:value="lsp.maxResultBytes" :min="1024" :step="1024" class="full-width" />
      </NFormItem>
    </NForm>

    <section class="servers-section">
      <header class="section-head">
        <strong>{{ t('settings.lsp.customServers') }}</strong>
        <NButton size="small" @click="addServer">
          <template #icon><NIcon><Add /></NIcon></template>
          {{ t('settings.lsp.addServer') }}
        </NButton>
      </header>

      <div v-if="!lsp.servers?.length" class="empty-line">{{ t('settings.lsp.noCustomServers') }}</div>
      <div v-for="(server, index) in lsp.servers" :key="index" class="server-row">
        <div class="server-grid">
          <NInput v-model:value="server.language" :placeholder="t('settings.lsp.language')" class="mono-input" />
          <NInput v-model:value="server.executable" :placeholder="t('settings.lsp.executable')" class="mono-input" />
          <NInput :value="commandText(server)" :placeholder="t('settings.lsp.command')" class="mono-input" @update:value="(value: string) => updateCommand(server, value)" />
          <NInput :value="extensionsText(server)" :placeholder="t('settings.lsp.extensions')" class="mono-input" @update:value="(value: string) => updateExtensions(server, value)" />
        </div>
        <div class="server-actions">
          <NSwitch v-model:value="server.disabled" :checked-value="true" :unchecked-value="false" />
          <NButton quaternary circle size="small" :aria-label="t('common.remove')" @click="removeServer(index)">
            <template #icon><Trash /></template>
          </NButton>
        </div>
      </div>
    </section>

    <div class="form-actions">
      <NButton size="small" :loading="loadingStatus" @click="refreshStatus">
        <template #icon><NIcon><Refresh /></NIcon></template>
        {{ t('settings.lsp.status') }}
      </NButton>
    </div>

    <section v-if="status" class="status-section">
      <div class="status-head">
        <NIcon size="18"><CodeSlash /></NIcon>
        <strong>{{ status.enabled ? t('settings.lsp.enabled') : t('settings.lsp.disabled') }}</strong>
        <NTag size="small">{{ status.servers?.length || 0 }} {{ t('settings.lsp.runningServers') }}</NTag>
      </div>
      <div class="status-list">
        <div v-for="server in status.servers || []" :key="`${server.sessionId}:${server.workspace}:${server.language}`" class="status-row">
          <span class="mono">{{ server.language }}</span>
          <NTag size="small" :type="server.alive ? 'success' : 'error'">{{ server.alive ? 'alive' : 'stopped' }}</NTag>
          <span>{{ server.workspace }}</span>
          <span>{{ server.requestCount }} req</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.lsp-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.full-width {
  width: 100%;
}

.servers-section,
.status-section {
  border-top: 1px solid var(--border-subtle);
  padding-top: 10px;
}

.section-head,
.status-head,
.server-row,
.status-row,
.form-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-head,
.form-actions {
  justify-content: space-between;
}

.server-row {
  align-items: flex-start;
  margin-top: 8px;
}

.server-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.server-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.mono-input :deep(input),
.mono {
  font-family: var(--font-mono) !important;
  font-size: 12px !important;
}

.empty-line,
.status-row {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.status-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 8px;
}

.status-row {
  min-height: 28px;
  overflow-wrap: anywhere;
}

@media (max-width: 720px) {
  .server-grid {
    grid-template-columns: 1fr;
  }
}
</style>
