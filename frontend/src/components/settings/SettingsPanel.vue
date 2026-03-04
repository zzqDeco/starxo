<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NCard, NTabs, NTabPane, NButton, NIcon } from 'naive-ui'
import { Close } from '@vicons/ionicons5'
import { useSettingsStore } from '@/stores/settingsStore'
import SSHConfigForm from './SSHConfig.vue'
import DockerConfigForm from './DockerConfig.vue'
import LLMConfigForm from './LLMConfig.vue'
import MCPConfigForm from './MCPConfig.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
}>()

const settingsStore = useSettingsStore()
const activeTab = ref('ssh')
const saving = ref(false)

async function handleSave() {
  saving.value = true
  try {
    await settingsStore.saveSettings()
    emit('update:show', false)
  } catch (e) {
    console.error('Save failed:', e)
  } finally {
    saving.value = false
  }
}

function handleClose() {
  emit('update:show', false)
}

function onBackdropClick(e: MouseEvent) {
  // Only close if clicking the backdrop itself, not the card
  if (e.target === e.currentTarget) {
    handleClose()
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.show) {
    handleClose()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Transition name="settings-modal">
    <div v-if="show" class="settings-overlay" @mousedown="onBackdropClick">
      <div class="settings-dialog">
        <NCard
          :bordered="false"
          size="large"
          role="dialog"
          aria-modal="true"
          class="settings-card"
          :title="t('settings.title')"
          :segmented="{ content: true, footer: 'soft' }"
        >
          <template #header-extra>
            <NButton quaternary circle size="small" @click="handleClose">
              <template #icon>
                <NIcon><Close /></NIcon>
              </template>
            </NButton>
          </template>

          <NTabs v-model:value="activeTab" type="line" animated class="settings-tabs">
            <NTabPane name="ssh" :tab="t('settings.ssh.tab')">
              <SSHConfigForm />
            </NTabPane>
            <NTabPane name="docker" :tab="t('settings.docker.tab')">
              <DockerConfigForm />
            </NTabPane>
            <NTabPane name="llm" :tab="t('settings.llm.tab')">
              <LLMConfigForm />
            </NTabPane>
            <NTabPane name="mcp" :tab="t('settings.mcp.tab')">
              <MCPConfigForm />
            </NTabPane>
          </NTabs>

          <template #footer>
            <div class="settings-footer">
              <NButton size="small" quaternary @click="settingsStore.resetToDefaults()">
                {{ t('settings.resetDefaults') }}
              </NButton>
              <div class="footer-right">
                <NButton size="small" @click="handleClose">{{ t('common.cancel') }}</NButton>
                <NButton
                  type="primary"
                  size="small"
                  :loading="saving"
                  @click="handleSave"
                >
                  {{ t('settings.saveSettings') }}
                </NButton>
              </div>
            </div>
          </template>
        </NCard>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.settings-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
}

.settings-dialog {
  position: relative;
  z-index: 2001;
}

.settings-card {
  width: 600px;
  max-height: 80vh;
  border-radius: var(--radius-xl) !important;
  background: var(--bg-surface) !important;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.4);
}

.settings-tabs {
  min-height: 320px;
}

.settings-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-right {
  display: flex;
  gap: 8px;
}

/* Transition */
.settings-modal-enter-active,
.settings-modal-leave-active {
  transition: opacity 200ms ease;
}

.settings-modal-enter-active .settings-dialog,
.settings-modal-leave-active .settings-dialog {
  transition: transform 200ms ease, opacity 200ms ease;
}

.settings-modal-enter-from,
.settings-modal-leave-to {
  opacity: 0;
}

.settings-modal-enter-from .settings-dialog {
  transform: scale(0.95);
  opacity: 0;
}

.settings-modal-leave-to .settings-dialog {
  transform: scale(0.95);
  opacity: 0;
}
</style>
