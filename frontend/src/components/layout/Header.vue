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
  <header class="app-header wails-drag">
    <div class="header-left">
      <div class="app-title">
        <span class="title-icon">&#x25C8;</span>
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
  height: 52px;
  padding: 0 16px;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  z-index: 10;
  position: relative;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-title {
  display: flex;
  align-items: center;
  gap: 8px;
  user-select: none;
}

.title-icon {
  font-size: 18px;
  color: var(--accent-cyan);
  filter: drop-shadow(0 0 6px rgba(34, 211, 238, 0.4));
}

.title-text {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.5px;
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
  gap: 4px;
}

.header-btn {
  color: var(--text-muted) !important;
  transition: color var(--transition-fast) !important;
}

.header-btn:hover {
  color: var(--text-primary) !important;
}

.lang-btn {
  font-size: 12px !important;
  font-weight: 700 !important;
  letter-spacing: 0.3px;
  min-width: 32px;
}
</style>
