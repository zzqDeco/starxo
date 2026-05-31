<script lang="ts" setup>
import { computed } from 'vue'
import { NButton, NTooltip } from 'naive-ui'
import ConnectionStatus from '@/components/status/ConnectionStatus.vue'
import SxToolbarItem from '@/components/ui/SxToolbarItem.vue'
import { resolveSxIcon } from '@/components/ui/icons'
import { useI18n } from 'vue-i18n'
import { useSessionStore } from '@/stores/sessionStore'
import { toggleWindowZoom } from '@/composables/useNativeWindow'

const { t, locale } = useI18n()
const sessionStore = useSessionStore()
const SearchIcon = resolveSxIcon('search')

function displaySessionTitle(title?: string) {
  const normalized = (title || '').trim()
  if (!normalized || normalized === 'Default Session') return t('sidebar.untitled')
  return normalized
}

const activeSessionTitle = computed(() => displaySessionTitle(sessionStore.activeSession?.title))

defineProps<{
  workspaceDrawerVisible: boolean
  runtimeTasksVisible: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-settings'): void
  (e: 'toggle-workspace-drawer'): void
  (e: 'toggle-runtime-tasks'): void
  (e: 'toggle-terminal'): void
  (e: 'open-command-palette'): void
}>()

function toggleLocale() {
  locale.value = locale.value === 'en' ? 'zh' : 'en'
  localStorage.setItem('locale', locale.value)
}

function isInteractiveTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false
  return !!target.closest('button, a, input, textarea, select, [role="button"], .n-button, .n-input')
}

function handleTitlebarDoubleClick(event: MouseEvent) {
  if (isInteractiveTarget(event.target)) return
  void toggleWindowZoom()
}
</script>

<template>
  <header class="app-header wails-drag" role="banner" @dblclick="handleTitlebarDoubleClick">
    <div class="header-left">
      <div class="app-title" aria-label="Starxo">
        <span class="title-text">{{ t('header.title') }}</span>
      </div>
    </div>

    <div class="header-center">
      <button
        type="button"
        class="command-trigger"
        :aria-label="t('header.commandPalette')"
        @click="emit('open-command-palette')"
      >
        <SearchIcon :size="14" class="command-icon" />
        <span class="command-copy">
          <span class="command-main">{{ activeSessionTitle }}</span>
          <span class="command-sub">{{ t('header.commandPlaceholder') }}</span>
        </span>
        <kbd class="command-kbd" aria-hidden="true">⌘K</kbd>
      </button>
    </div>

    <div class="header-right">
      <ConnectionStatus />

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <SxToolbarItem
            icon="tasks"
            compact
            :label="runtimeTasksVisible ? t('header.runtimeTasksClose') : t('header.runtimeTasksOpen')"
            :active="runtimeTasksVisible"
            @click="emit('toggle-runtime-tasks')"
          />
        </template>
        {{ runtimeTasksVisible ? t('header.runtimeTasksClose') : t('header.runtimeTasksOpen') }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <SxToolbarItem
            icon="terminal"
            compact
            :label="t('terminal.terminal')"
            @click="emit('toggle-terminal')"
          />
        </template>
        {{ t('terminal.terminal') }}
      </NTooltip>

      <NTooltip trigger="hover" placement="bottom">
        <template #trigger>
          <SxToolbarItem
            icon="workspace"
            compact
            :label="workspaceDrawerVisible ? t('header.workspaceClose') : t('header.workspaceOpen')"
            :active="workspaceDrawerVisible"
            @click="emit('toggle-workspace-drawer')"
          />
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
          <SxToolbarItem
            icon="settings"
            compact
            :label="t('header.settings')"
            @click="emit('toggle-settings')"
          />
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
  height: 50px;
  padding: 0 12px;
  background: var(--platform-bg-toolbar);
  backdrop-filter: blur(20px) saturate(1.15);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
  z-index: var(--z-sticky, 20);
  position: relative;
}

:global(:root[data-platform="macos"] .app-header){
  height: 48px;
  background: color-mix(in srgb, var(--platform-bg-toolbar) 90%, transparent);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  --wails-draggable: no-drag;
}

.app-title {
  display: flex;
  align-items: center;
  gap: 6px;
  user-select: none;
}

.title-text {
  font-family: var(--font-brand);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--text-muted);
  letter-spacing: 0;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
  min-width: 0;
  padding: 0 12px;
  --wails-draggable: no-drag;
}

.command-trigger {
  width: min(460px, 100%);
  height: 34px;
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
  background: color-mix(in srgb, var(--platform-bg-raised) 74%, transparent);
  color: var(--text-secondary);
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-sm);
  padding: 0 10px;
  text-align: left;
  transition: border-color var(--transition-ui), background var(--transition-ui), box-shadow var(--transition-ui);
}

:global(:root[data-platform="macos"] .command-trigger){
  height: 32px;
  max-width: 440px;
  border-radius: 7px;
  background: color-mix(in srgb, var(--platform-bg-raised) 80%, transparent);
  box-shadow: inset 0 0 0 0.5px color-mix(in srgb, var(--border-subtle) 80%, transparent);
}

.command-trigger:hover,
.command-trigger:focus-visible {
  background: var(--platform-bg-raised);
  border-color: var(--border-strong);
  box-shadow: none;
}

.command-icon {
  color: var(--text-faint);
  flex-shrink: 0;
}

.command-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.command-main,
.command-sub {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-main {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--text-secondary);
}

:global(:root[data-platform="macos"] .command-main){
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

:global(:root[data-platform="macos"] .command-sub){
  font-size: 10.5px;
}

.command-sub {
  font-size: var(--fs-2xs);
  color: var(--text-faint);
}

.command-kbd {
  font-family: var(--font-mono);
  font-size: var(--fs-2xs);
  color: var(--text-faint);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 78%, transparent);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
  line-height: 1.2;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  --wails-draggable: no-drag;
}

.header-btn {
  color: var(--text-muted) !important;
  transition: color var(--transition-ui), background var(--transition-ui) !important;
}

.header-btn:hover {
  color: var(--text-primary) !important;
}

.lang-btn {
  font-family: var(--font-sans);
  font-size: var(--fs-xs) !important;
  font-weight: var(--fw-medium) !important;
  letter-spacing: 0;
  min-width: 32px;
}

@media (max-width: 860px) {
  .command-sub {
    display: none;
  }

  .command-trigger {
    height: 34px;
  }
}

@media (max-width: 640px) {
  .title-text,
  .command-kbd {
    display: none;
  }

  .header-center {
    padding: 0 var(--space-sm);
  }
}
</style>
