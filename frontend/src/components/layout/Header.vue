<script lang="ts" setup>
import { NButton, NTooltip } from 'naive-ui'
import { Settings, FolderOpen } from '@vicons/ionicons5'
import ConnectionStatus from '@/components/status/ConnectionStatus.vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

defineProps<{
  workspaceDrawerVisible: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-settings'): void
  (e: 'toggle-workspace-drawer'): void
}>()

function toggleLocale() {
  locale.value = locale.value === 'en' ? 'zh' : 'en'
  localStorage.setItem('locale', locale.value)
}
</script>

<template>
  <header class="app-header wails-drag" role="banner">
    <div class="header-left">
      <div class="app-title" aria-label="Starxo">
        <span class="title-icon" aria-hidden="true">&#x25C8;</span>
        <span class="title-text">{{ t('header.title') }}</span>
      </div>
    </div>

    <div class="header-center">
      <ConnectionStatus />
    </div>

    <div class="header-right">
      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="header-btn"
            :aria-label="workspaceDrawerVisible ? t('header.workspaceClose') : t('header.workspaceOpen')"
            @click="emit('toggle-workspace-drawer')"
          >
            <template #icon>
              <FolderOpen />
            </template>
          </NButton>
        </template>
        {{ workspaceDrawerVisible ? t('header.workspaceClose') : t('header.workspaceOpen') }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            size="small"
            class="header-btn lang-btn"
            :aria-label="locale === 'en' ? '切换到中文' : 'Switch to English'"
            @click="toggleLocale"
          >
            {{ locale === 'en' ? '中' : 'EN' }}
          </NButton>
        </template>
        {{ locale === 'en' ? '切换到中文' : 'Switch to English' }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="small"
            class="header-btn"
            :aria-label="t('header.settings')"
            @click="emit('toggle-settings')"
          >
            <template #icon>
              <Settings />
            </template>
          </NButton>
        </template>
        {{ t('header.settings') }}
      </NTooltip>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 var(--space-lg);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  z-index: var(--z-sticky, 20);
  position: relative;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.app-title {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  user-select: none;
}

.title-icon {
  font-size: var(--fs-lg);
  color: var(--accent-cyan);
  filter: drop-shadow(0 0 6px rgba(34, 211, 238, 0.4));
}

.title-text {
  font-family: var(--font-brand);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  color: var(--text-primary);
  letter-spacing: 0.8px;
  text-transform: uppercase;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
  min-width: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.header-btn {
  color: var(--text-muted) !important;
  transition: color var(--transition-ui), background var(--transition-ui) !important;
}

.header-btn:hover {
  color: var(--text-primary) !important;
}

.lang-btn {
  font-family: var(--font-brand);
  font-size: var(--fs-xs) !important;
  font-weight: var(--fw-bold) !important;
  letter-spacing: 0.5px;
  min-width: 32px;
}
</style>
