<script lang="ts" setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { Close, Terminal, ShieldCheckmark, Cloud, Apps } from '@vicons/ionicons5'
import { useSettingsStore } from '@/stores/settingsStore'
import SSHConfigForm from './SSHConfig.vue'
import SandboxConfigForm from './SandboxConfig.vue'
import LLMConfigForm from './LLMConfig.vue'
import MCPConfigForm from './MCPConfig.vue'
import { useI18n } from 'vue-i18n'
import { useFocusTrap } from '@/composables/useFocusTrap'

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

const dialogRef = ref<HTMLElement | null>(null)
const trapActive = computed(() => props.show)
useFocusTrap(dialogRef, trapActive)

const tabs = computed(() => [
  { name: 'ssh', label: t('settings.ssh.tab'), icon: Terminal },
  { name: 'sandbox', label: t('settings.sandbox.tab'), icon: ShieldCheckmark },
  { name: 'llm', label: t('settings.llm.tab'), icon: Cloud },
  { name: 'mcp', label: t('settings.mcp.tab'), icon: Apps },
])

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
      <div
        ref="dialogRef"
        class="settings-dialog"
        role="dialog"
        aria-modal="true"
        :aria-label="t('settings.title')"
        tabindex="-1"
      >
        <header class="settings-header">
          <h2 class="settings-heading">{{ t('settings.title') }}</h2>
          <NButton quaternary circle size="small" :aria-label="t('common.cancel')" @click="handleClose">
            <template #icon>
              <NIcon><Close /></NIcon>
            </template>
          </NButton>
        </header>

        <div class="settings-body">
          <nav class="settings-nav" role="tablist" aria-label="Settings sections">
            <button
              v-for="tab in tabs"
              :key="tab.name"
              :id="`settings-tab-${tab.name}`"
              :class="['settings-nav-item', { active: activeTab === tab.name }]"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.name"
              :aria-controls="`settings-pane-${tab.name}`"
              :tabindex="activeTab === tab.name ? 0 : -1"
              @click="activeTab = tab.name"
            >
              <NIcon size="16"><component :is="tab.icon" /></NIcon>
              <span class="nav-label">{{ tab.label }}</span>
            </button>
          </nav>

          <section
            class="settings-pane"
            role="tabpanel"
            :id="`settings-pane-${activeTab}`"
            :aria-labelledby="`settings-tab-${activeTab}`"
            tabindex="0"
          >
            <Transition name="fade-fast" mode="out-in">
              <SSHConfigForm v-if="activeTab === 'ssh'" key="ssh" />
              <SandboxConfigForm v-else-if="activeTab === 'sandbox'" key="sandbox" />
              <LLMConfigForm v-else-if="activeTab === 'llm'" key="llm" />
              <MCPConfigForm v-else-if="activeTab === 'mcp'" key="mcp" />
            </Transition>
          </section>
        </div>

        <footer class="settings-footer">
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
        </footer>
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
  backdrop-filter: blur(2px);
  padding: var(--space-lg);
}

.settings-dialog {
  position: relative;
  z-index: 2001;
  width: min(720px, 92vw);
  height: min(620px, 85vh);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  box-shadow: var(--elev-3);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.settings-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-md) var(--space-xl);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.settings-heading {
  margin: 0;
  font-family: var(--font-brand);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-primary);
  letter-spacing: 0.6px;
  text-transform: uppercase;
}

.settings-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.settings-nav {
  width: 180px;
  flex-shrink: 0;
  padding: var(--space-md) var(--space-sm);
  background: var(--bg-deepest);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: var(--space-2xs);
  overflow-y: auto;
}

.settings-nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background: transparent;
  border: none;
  border-left: 2px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-family: var(--font-sans);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
  text-align: left;
  transition: background var(--transition-ui), color var(--transition-ui), border-color var(--transition-ui);
}

.settings-nav-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.settings-nav-item:focus-visible {
  outline: 2px solid var(--accent-cyan);
  outline-offset: -2px;
}

.settings-nav-item.active {
  background: var(--bg-elevated);
  color: var(--accent-cyan);
  border-left-color: var(--accent-cyan);
}

.nav-label {
  flex: 1;
}

.settings-pane {
  flex: 1;
  min-width: 0;
  padding: var(--space-xl);
  overflow-y: auto;
}

.settings-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-md) var(--space-xl);
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
  flex-shrink: 0;
}

.footer-right {
  display: flex;
  gap: var(--space-sm);
}

/* Fade transition for tab switch */
.fade-fast-enter-active,
.fade-fast-leave-active {
  transition: opacity 120ms var(--ease-out);
}

.fade-fast-enter-from,
.fade-fast-leave-to {
  opacity: 0;
}

/* Modal transition */
.settings-modal-enter-active,
.settings-modal-leave-active {
  transition: opacity 200ms var(--ease-out);
}

.settings-modal-enter-active .settings-dialog,
.settings-modal-leave-active .settings-dialog {
  transition: transform 200ms var(--ease-out), opacity 200ms var(--ease-out);
}

.settings-modal-enter-from,
.settings-modal-leave-to {
  opacity: 0;
}

.settings-modal-enter-from .settings-dialog,
.settings-modal-leave-to .settings-dialog {
  transform: scale(0.96);
  opacity: 0;
}

/* Responsive — collapse nav to top row on narrow windows */
@media (max-width: 640px) {
  .settings-body {
    flex-direction: column;
  }
  .settings-nav {
    width: 100%;
    flex-direction: row;
    overflow-x: auto;
    padding: var(--space-sm);
    border-right: none;
    border-bottom: 1px solid var(--border-subtle);
  }
  .settings-nav-item {
    border-left: none;
    border-bottom: 2px solid transparent;
    white-space: nowrap;
  }
  .settings-nav-item.active {
    border-left: none;
    border-bottom-color: var(--accent-cyan);
  }
}
</style>
