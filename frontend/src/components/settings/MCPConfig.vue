<script lang="ts" setup>
import { ref } from 'vue'
import {
  NButton, NIcon, NInput, NSelect, NSwitch, NCard,
  NSpace, NEmpty, NPopconfirm
} from 'naive-ui'
import { Add, TrashOutline } from '@vicons/ionicons5'
import { useSettingsStore } from '@/stores/settingsStore'
import type { MCPServerConfig } from '@/types/config'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()

const transportOptions = [
  { label: 'Stdio', value: 'stdio' },
  { label: 'SSE', value: 'sse' },
]

function addServer() {
  const newServer: MCPServerConfig = {
    name: '',
    transport: 'stdio',
    command: '',
    args: [],
    url: '',
    enabled: true
  }
  settingsStore.addMCPServer(newServer)
}

function removeServer(index: number) {
  settingsStore.removeMCPServer(index)
}

function updateArgs(index: number, value: string) {
  settingsStore.settings.mcp.servers[index].args = value
    .split('\n')
    .map(s => s.trim())
    .filter(s => s.length > 0)
}

function getArgsText(server: MCPServerConfig): string {
  return (server.args || []).join('\n')
}
</script>

<template>
  <div class="config-form">
    <div class="server-list">
      <NEmpty
        v-if="settingsStore.settings.mcp.servers.length === 0"
        :description="t('settings.mcp.noServers')"
        size="small"
        class="empty-servers"
      >
        <template #extra>
          <NButton size="small" @click="addServer">
            <template #icon><NIcon><Add /></NIcon></template>
            {{ t('settings.mcp.addServer') }}
          </NButton>
        </template>
      </NEmpty>

      <template v-for="(server, i) in settingsStore.settings.mcp.servers" :key="i">
        <NCard size="small" class="server-card" :bordered="true">
          <div class="server-header">
            <div class="server-enable">
              <NSwitch v-model:value="server.enabled" size="small" />
            </div>
            <NInput
              v-model:value="server.name"
              :placeholder="t('settings.mcp.serverNamePlaceholder')"
              size="small"
              class="server-name-input"
            />
            <NPopconfirm
              @positive-click="removeServer(i)"
              :positive-text="t('common.remove')"
              :negative-text="t('common.cancel')"
            >
              <template #trigger>
                <NButton
                  quaternary
                  circle
                  size="tiny"
                  type="error"
                >
                  <template #icon>
                    <NIcon size="14"><TrashOutline /></NIcon>
                  </template>
                </NButton>
              </template>
              {{ server.name ? t('settings.mcp.removeConfirm', { name: server.name }) : t('settings.mcp.removeThis') }}
            </NPopconfirm>
          </div>

          <div class="server-fields">
            <div class="field-row">
              <span class="field-label">{{ t('settings.mcp.transport') }}</span>
              <NSelect
                v-model:value="server.transport"
                :options="transportOptions"
                size="tiny"
                style="width: 120px;"
              />
            </div>

            <template v-if="server.transport === 'stdio'">
              <div class="field-row">
                <span class="field-label">{{ t('settings.mcp.command') }}</span>
                <NInput
                  v-model:value="server.command"
                  :placeholder="t('settings.mcp.commandPlaceholder')"
                  size="tiny"
                  class="mono-input"
                />
              </div>
              <div class="field-row">
                <span class="field-label">{{ t('settings.mcp.args') }}</span>
                <NInput
                  :value="getArgsText(server)"
                  @update:value="updateArgs(i, $event)"
                  type="textarea"
                  :autosize="{ minRows: 1, maxRows: 3 }"
                  :placeholder="t('settings.mcp.argsPlaceholder')"
                  size="tiny"
                  class="mono-input"
                />
              </div>
            </template>

            <template v-if="server.transport === 'sse'">
              <div class="field-row">
                <span class="field-label">{{ t('settings.mcp.url') }}</span>
                <NInput
                  v-model:value="server.url"
                  :placeholder="t('settings.mcp.urlPlaceholder')"
                  size="tiny"
                  class="mono-input"
                />
              </div>
            </template>
          </div>
        </NCard>
      </template>
    </div>

    <div v-if="settingsStore.settings.mcp.servers.length > 0" class="add-more">
      <NButton size="small" dashed block @click="addServer">
        <template #icon><NIcon><Add /></NIcon></template>
        {{ t('settings.mcp.addServer') }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.config-form {
  padding: 12px 0;
}

.server-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.empty-servers {
  padding: 24px 0;
}

.server-card {
  background: var(--bg-deepest) !important;
  border-color: var(--border-subtle) !important;
  border-radius: var(--radius-md) !important;
}

.server-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.server-enable {
  flex-shrink: 0;
}

.server-name-input {
  flex: 1;
}

.server-fields {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.field-label {
  width: 70px;
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  padding-top: 4px;
}

.field-row .mono-input {
  flex: 1;
}

.mono-input :deep(input),
.mono-input :deep(textarea) {
  font-family: var(--font-mono) !important;
  font-size: 11px !important;
}

.add-more {
  margin-top: 10px;
}
</style>
